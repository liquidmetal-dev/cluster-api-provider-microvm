// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package scope

import (
	"context"
	"encoding/base64"
	"fmt"
	"hash/crc32"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/controllers/noderefutil"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/weaveworks-liquidmetal/cluster-api-provider-microvm/api/v1alpha1"
	"github.com/weaveworks-liquidmetal/cluster-api-provider-microvm/internal/defaults"
)

var _ Scoper = &MachineScope{}

const ProviderPrefix = "microvm://"

const (
	tlsCert = "tls.crt"
	tlsKey  = "tls.key"
	caCert  = "ca.crt"
)

type MachineScopeParams struct {
	Cluster        *clusterv1.Cluster
	MicroVMCluster *infrav1.MicrovmCluster

	Machine        *clusterv1.Machine
	MicroVMMachine *infrav1.MicrovmMachine

	Client  client.Client
	Context context.Context //nolint: containedctx // don't care
}

func NewMachineScope(params MachineScopeParams, opts ...MachineScopeOption) (*MachineScope, error) {
	if params.Cluster == nil {
		return nil, errClusterRequired
	}

	if params.MicroVMCluster == nil {
		return nil, errMicrovmClusterRequired
	}

	if params.Machine == nil {
		return nil, errMachineRequired
	}

	if params.MicroVMMachine == nil {
		return nil, errMicrovmMachineRequied
	}

	if params.Client == nil {
		return nil, errClientRequired
	}

	patchHelper, err := patch.NewHelper(params.MicroVMMachine, params.Client)
	if err != nil {
		return nil, fmt.Errorf("creating patch helper for microvm machine: %w", err)
	}

	scope := &MachineScope{
		Cluster:        params.Cluster,
		MvmCluster:     params.MicroVMCluster,
		Machine:        params.Machine,
		MvmMachine:     params.MicroVMMachine,
		client:         params.Client,
		controllerName: defaults.ManagerName,
		Logger:         klogr.New(),
		patchHelper:    patchHelper,
		ctx:            params.Context,
	}

	for _, opt := range opts {
		opt(scope)
	}

	return scope, nil
}

type MachineScopeOption func(*MachineScope)

func WithMachineLogger(logger logr.Logger) MachineScopeOption {
	return func(s *MachineScope) {
		s.Logger = logger
	}
}

func WithMachineControllerName(name string) MachineScopeOption {
	return func(s *MachineScope) {
		s.controllerName = name
	}
}

type MachineScope struct {
	logr.Logger

	Cluster    *clusterv1.Cluster
	MvmCluster *infrav1.MicrovmCluster

	Machine    *clusterv1.Machine
	MvmMachine *infrav1.MicrovmMachine

	client         client.Client
	patchHelper    *patch.Helper
	controllerName string
	ctx            context.Context
}

// Name returns the MicrovmMachine name.
func (m *MachineScope) Name() string {
	return m.MvmMachine.Name
}

// Namespace returns the namespace name.
func (m *MachineScope) Namespace() string {
	return m.MvmMachine.Namespace
}

// ClusterName returns the name of the cluster.
func (m *MachineScope) ClusterName() string {
	return m.Cluster.Name
}

// ControllerName returns the name of the controller that created the scope.
func (m *MachineScope) ControllerName() string {
	return m.controllerName
}

// IsControlPlane returns true if the machine is a control plane.
func (m *MachineScope) IsControlPlane() bool {
	return util.IsControlPlaneMachine(m.Machine)
}

// Patch persists the resource and status.
func (m *MachineScope) Patch() error {
	applicableConditions := []clusterv1.ConditionType{
		infrav1.MicrovmReadyCondition,
	}

	conditions.SetSummary(m.MvmMachine,
		conditions.WithConditions(applicableConditions...),
		conditions.WithStepCounterIf(m.MvmMachine.DeletionTimestamp.IsZero()),
		conditions.WithStepCounter(),
	)

	err := m.patchHelper.Patch(
		m.ctx,
		m.MvmMachine,
		patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
			infrav1.MicrovmReadyCondition,
		}})
	if err != nil {
		return fmt.Errorf("unable to patch machine: %w", err)
	}

	return nil
}

func (m *MachineScope) GetFailureDomain() (string, error) {
	if m.Machine.Spec.FailureDomain != nil && *m.Machine.Spec.FailureDomain != "" {
		return *m.Machine.Spec.FailureDomain, nil
	}

	providerID := m.GetProviderID()
	if providerID != "" {
		return m.getFailureDomainFromProviderID(providerID), nil
	}

	// If we've got this far then we need to work out how to get a failure domain. In the future we will make
	// the strategy configurable for static placement and also add support for the scheduler.
	failureDomainNames := make([]string, 0, len(m.Cluster.Status.FailureDomains))
	for fdName := range m.Cluster.Status.FailureDomains {
		failureDomainNames = append(failureDomainNames, fdName)
	}

	if len(failureDomainNames) == 0 {
		return "", errFailureDomainNotFound
	}

	if len(failureDomainNames) == 1 {
		return failureDomainNames[0], nil
	}

	sort.Strings(failureDomainNames)
	pos := int(crc32.ChecksumIEEE([]byte(m.MvmMachine.Name))) % len(failureDomainNames)

	return failureDomainNames[pos], nil
}

// GetRawBootstrapData will return the contents of the secret that has been created by the
// bootstrap provider that is being used for this cluster/machine. Initially this we will
// be using the Kubeadm bootstrap provider and so this will contain cloud-init configuration
// that will invoke kubeadm to create or join a cluster.
func (m *MachineScope) GetRawBootstrapData() ([]byte, error) {
	if m.Machine.Spec.Bootstrap.DataSecretName == nil {
		return nil, errMissingBootstrapDataSecret
	}

	bootstrapSecret := &corev1.Secret{}
	secretKey := types.NamespacedName{
		Namespace: m.Namespace(),
		Name:      *m.Machine.Spec.Bootstrap.DataSecretName,
	}

	if err := m.client.Get(m.ctx, secretKey, bootstrapSecret); err != nil {
		return nil, fmt.Errorf("getting bootstrap secret %s: %w", secretKey, err)
	}

	bootstrapData, ok := bootstrapSecret.Data["value"]
	if !ok {
		return nil, errMissingBootstrapSecretKey
	}

	return bootstrapData, nil
}

// SetReady sets any properties/conditions that are used to indicate that the MicrovmMachine is 'Ready'
// back to the upstream CAPI machine controllers.
func (m *MachineScope) SetReady() {
	conditions.MarkTrue(m.MvmMachine, infrav1.MicrovmReadyCondition)
	m.MvmMachine.Status.Ready = true
}

// SetNotReady sets any properties/conditions that are used to indicate that the MicrovmMachine is NOT 'Ready'
// back to the upstream CAPI machine controllers.
func (m *MachineScope) SetNotReady(
	reason string,
	severity clusterv1.ConditionSeverity,
	message string,
	messageArgs ...interface{},
) {
	conditions.MarkFalse(m.MvmMachine, infrav1.MicrovmReadyCondition, reason, severity, message, messageArgs...)
	m.MvmMachine.Status.Ready = false
}

// SetProviderID saves the unique microvm and object ID to the MvmMachine spec.
func (m *MachineScope) SetProviderID(failureDomain, mvmUID string) {
	providerID := fmt.Sprintf("%s%s/%s", ProviderPrefix, failureDomain, mvmUID)
	m.MvmMachine.Spec.ProviderID = &providerID
}

// GetProviderID returns the provider if for the machine. If there is no provider id
// then an empty string will be returned.
func (m *MachineScope) GetProviderID() string {
	if m.MvmMachine.Spec.ProviderID != nil {
		return *m.MvmMachine.Spec.ProviderID
	}

	return ""
}

// GetInstanceID gets the instance ID (i.e. UID) of the machine.
func (m *MachineScope) GetInstanceID() string {
	parsed, err := noderefutil.NewProviderID(m.GetProviderID())
	if err != nil {
		return ""
	}

	return parsed.ID()
}

// GetSSHPublicKeys will return the SSH public keys for this machine. It will take into account
// precedence rules. If there are no keys then nil will be returned.
func (m *MachineScope) GetSSHPublicKeys() []infrav1.SSHPublicKey {
	if len(m.MvmMachine.Spec.SSHPublicKeys) != 0 {
		return m.MvmMachine.Spec.SSHPublicKeys
	}

	if len(m.MvmCluster.Spec.SSHPublicKeys) != 0 {
		return m.MvmCluster.Spec.SSHPublicKeys
	}

	return nil
}

// GetBasicAuthToken will fetch the BasicAuthSecret on the MvmCluster and
// and return the token for the given host.
// If no secret or no value is found, an empty string is returned.
func (m *MachineScope) GetBasicAuthToken(addr string) (string, error) {
	placement := m.MvmCluster.Spec.Placement
	if placement.StaticPool == nil || placement.StaticPool.BasicAuthSecret == "" {
		return "", nil
	}

	tokenSecret := &corev1.Secret{}
	key := types.NamespacedName{
		Name:      m.MvmCluster.Spec.Placement.StaticPool.BasicAuthSecret,
		Namespace: m.MvmCluster.Namespace,
	}

	if err := m.client.Get(m.ctx, key, tokenSecret); err != nil {
		return "", err
	}

	host := strings.Split(addr, ":")[0]
	// If it's not there, that's fine; we will log and return an empty string
	token := string(tokenSecret.Data[host])

	if token == "" {
		m.Info(
			"basicAuthToken for host not found in secret", "secret", tokenSecret.Name, "host", host,
		)
	}

	return token, nil
}

// GetTLSConfig will fetch the TLSSecretRef and CASecretRef on the MvmCluster
// and return the TLS config for the client.
// If either are not set, it will be assumed that the hosts are not
// configured will TLS and all client calls will be made without credentials.
func (m *MachineScope) GetTLSConfig() (*infrav1.TLSConfig, error) {
	if m.MvmCluster.Spec.TLSSecretRef == "" {
		m.Info("no TLS configuration found. will create insecure connection")

		return nil, nil
	}

	secretKey := types.NamespacedName{
		Name:      m.MvmCluster.Spec.TLSSecretRef,
		Namespace: m.MvmCluster.Namespace,
	}

	tlsSecret := &corev1.Secret{}
	if err := m.client.Get(context.TODO(), secretKey, tlsSecret); err != nil {
		return nil, err
	}

	cert, err := decode(tlsSecret.Data, tlsCert)
	if err != nil {
		return nil, err
	}

	key, err := decode(tlsSecret.Data, tlsKey)
	if err != nil {
		return nil, err
	}

	ca, err := decode(tlsSecret.Data, caCert)
	if err != nil {
		return nil, err
	}

	return &infrav1.TLSConfig{
		Cert:   cert,
		Key:    key,
		CACert: ca,
	}, nil
}

func decode(data map[string][]byte, key string) (string, error) {
	val, ok := data[key]
	if !ok {
		return "", &tlsError{key}
	}

	dec, err := base64.StdEncoding.DecodeString(string(val))
	if err != nil {
		return "", err
	}

	return string(dec), nil
}

func (m *MachineScope) getFailureDomainFromProviderID(providerID string) string {
	if providerID == "" {
		return ""
	}

	providerID = strings.ReplaceAll(providerID, ProviderPrefix, "")
	parts := strings.Split(providerID, "/")

	return parts[0]
}
