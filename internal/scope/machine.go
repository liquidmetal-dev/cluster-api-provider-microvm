// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package scope

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/collections"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/failuredomains"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/weaveworks/cluster-api-provider-microvm/api/v1alpha1"
	"github.com/weaveworks/cluster-api-provider-microvm/internal/defaults"
)

var _ Scoper = &MachineScope{}

type MachineScopeParams struct {
	Cluster        *clusterv1.Cluster
	MicroVMCluster *infrav1.MicrovmCluster

	Machine        *clusterv1.Machine
	MicroVMMachine *infrav1.MicrovmMachine

	Client  client.Client
	Context context.Context
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
	return m.Cluster.ClusterName
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
		context.TODO(),
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

// MicrovmServiceAddress will return the address of the microvm service to call. Any precedence
// logic needs to sit here.
func (m *MachineScope) MicrovmServiceAddress() (string, error) {
	if m.Machine.Spec.FailureDomain != nil && *m.Machine.Spec.FailureDomain != "" {
		return *m.Machine.Spec.FailureDomain, nil
	}

	machinesList, err := m.getMachinesInCluster()
	if err != nil {
		return "", fmt.Errorf("getting machines in cluster: %w", err)
	}

	machines := collections.FromMachineList(machinesList)

	failureDomain := failuredomains.PickFewest(m.Cluster.Status.FailureDomains, machines)
	if failureDomain != nil {
		return *failureDomain, nil
	}

	return "", errNoServiceAddress
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

	if err := m.client.Get(context.TODO(), secretKey, bootstrapSecret); err != nil {
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

// GetSSHPublicKey will return the SSH public key for this machine. It will take into account
// precedence rules. If there is no key then an empty string will be returned.
func (m *MachineScope) GetSSHPublicKey() string {
	if m.MvmMachine.Spec.SSHPublicKey != "" {
		return m.MvmMachine.Spec.SSHPublicKey
	}

	if m.MvmCluster.Spec.SSHPublicKey != "" {
		return m.MvmCluster.Spec.SSHPublicKey
	}

	return ""
}

func (m *MachineScope) ProviderID() string {
	return fmt.Sprintf("microvm://%s", m.MvmMachine.Name)
}

func (m *MachineScope) getMachinesInCluster() (*clusterv1.MachineList, error) {
	list := &clusterv1.MachineList{}
	labels := map[string]string{clusterv1.ClusterLabelName: m.ClusterName()}

	err := m.client.List(m.ctx, list, client.InNamespace(m.Cluster.Namespace), client.MatchingLabels(labels))
	if err != nil {
		return nil, fmt.Errorf("unable to list resources: %w", err)
	}

	return list, nil
}
