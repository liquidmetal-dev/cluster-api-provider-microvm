// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0.

package controllers_test

import (
	"encoding/base64"
	"errors"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/pointer"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	flintlocktypes "github.com/weaveworks-liquidmetal/flintlock/api/types"

	"github.com/weaveworks-liquidmetal/cluster-api-provider-microvm/api/v1alpha1"
	infrav1 "github.com/weaveworks-liquidmetal/cluster-api-provider-microvm/api/v1alpha1"
	"github.com/weaveworks-liquidmetal/cluster-api-provider-microvm/internal/services/microvm/mock_client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMachineReconcileMissingMvmMachine(t *testing.T) {
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()
	apiObjects.MvmMachine = nil

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	result, err := reconcileMachine(client, nil)
	g.Expect(err).NotTo(HaveOccurred(), "Reconciling when microvm machine doesn't exist should not error")
	g.Expect(result.IsZero()).To(BeTrue(), "Expect no requeue to be requested")
}

func TestMachineReconcileNoMachineOwnerRef(t *testing.T) {
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()
	apiObjects.MvmMachine.ObjectMeta.OwnerReferences = nil

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	result, err := reconcileMachine(client, nil)
	g.Expect(err).NotTo(HaveOccurred(), "Reconciling when microvm machine has no owner ref should not error")
	g.Expect(result.IsZero()).To(BeTrue(), "Expect no requeue to be requested")
}

func TestMachineReconcileMissingCluster(t *testing.T) {
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()
	apiObjects.Cluster = nil

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	result, err := reconcileMachine(client, nil)
	g.Expect(err).NotTo(HaveOccurred(), "Reconciling when cluster is missing should not error")
	g.Expect(result.IsZero()).To(BeTrue(), "Expect no requeue to be requested")
}

func TestMachineReconcileMachineMissingClusterLabel(t *testing.T) {
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()
	apiObjects.Machine.ObjectMeta.Labels = map[string]string{}

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	result, err := reconcileMachine(client, nil)
	g.Expect(err).NotTo(HaveOccurred(), "Reconciling when machine is missing capi labels should not error")
	g.Expect(result.IsZero()).To(BeTrue(), "Expect no requeue to be requested")
}

func TestMachineReconcileMvmMachinePaused(t *testing.T) {
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()
	apiObjects.MvmMachine.ObjectMeta.Annotations = map[string]string{
		clusterv1.PausedAnnotation: "true",
	}

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	result, err := reconcileMachine(client, nil)
	g.Expect(err).NotTo(HaveOccurred(), "Reconciling when microvm machine is paused should not error")
	g.Expect(result.IsZero()).To(BeTrue(), "Expect no requeue to be requested")
}

func TestMachineReconcileClusterPaused(t *testing.T) {
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()
	apiObjects.Cluster.Spec.Paused = true

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	result, err := reconcileMachine(client, nil)
	g.Expect(err).NotTo(HaveOccurred(), "Reconciling when cluster is paused should not error")
	g.Expect(result.IsZero()).To(BeTrue(), "Expect no requeue to be requested")
}

func TestMachineReconcileMissingMachine(t *testing.T) {
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()
	apiObjects.Machine = nil

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	result, err := reconcileMachine(client, nil)
	g.Expect(err).To(HaveOccurred(), "Reconciling when capi machine doesn't exist should error")
	g.Expect(result.IsZero()).To(BeTrue(), "Expect no requeue to be requested")
}

func TestMachineReconcileMvmClusterMissing(t *testing.T) {
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()
	apiObjects.MvmCluster = nil

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	result, err := reconcileMachine(client, nil)
	g.Expect(err).NotTo(HaveOccurred(), "Reconciling when microvm cluster missing should not error")
	g.Expect(result.IsZero()).To(BeTrue(), "Expect no requeue to be requested")
}

func TestMachineReconcileClusterInfraNotReady(t *testing.T) {
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()
	apiObjects.Cluster.Status.InfrastructureReady = false

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	result, err := reconcileMachine(client, nil)
	g.Expect(err).NotTo(HaveOccurred(), "Reconciling when cluster infrastructure is not ready should not error")
	g.Expect(result.IsZero()).To(BeTrue(), "Expect no requeue to be requested")

	reconciled, err := getMicrovmMachine(client, testMachineName, testClusterNamespace)
	g.Expect(err).NotTo(HaveOccurred(), "Getting microvm machine should not fail")

	assertConditionFalse(g, reconciled, infrav1.MicrovmReadyCondition, infrav1.WaitingForClusterInfraReason)
}

func TestMachineReconcileBoostrapNotReady(t *testing.T) {
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()
	apiObjects.Machine.Spec.Bootstrap.DataSecretName = nil

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	result, err := reconcileMachine(client, nil)
	g.Expect(err).NotTo(HaveOccurred(), "Reconciling when bootstrap data is not ready should not error")
	g.Expect(result.IsZero()).To(BeTrue(), "Expect no requeue to be requested")

	reconciled, err := getMicrovmMachine(client, testMachineName, testClusterNamespace)
	g.Expect(err).NotTo(HaveOccurred(), "Getting microvm machine should not fail")

	assertConditionFalse(g, reconciled, infrav1.MicrovmReadyCondition, infrav1.WaitingForBootstrapDataReason)
}

func TestMachineReconcileServiceGetError(t *testing.T) {
	g := NewWithT(t)

	objects := defaultClusterObjects()

	fakeAPIClient := mock_client.FakeClient{}
	fakeAPIClient.GetMicroVMReturns(nil, errors.New("something terrible happened"))

	client := createFakeClient(g, objects.AsRuntimeObjects())
	_, err := reconcileMachine(client, &fakeAPIClient)
	g.Expect(err).To(HaveOccurred(), "Reconciling when microvm service 'Get' errors should return error")
}

func TestMachineReconcileMachineExistsAndRunning(t *testing.T) {
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()

	fakeAPIClient := mock_client.FakeClient{}
	withExistingMicrovm(&fakeAPIClient, flintlocktypes.MicroVMStatus_CREATED)

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	result, err := reconcileMachine(client, &fakeAPIClient)
	g.Expect(err).NotTo(HaveOccurred(), "Reconciling when microvm service exists should not return error")
	g.Expect(result.IsZero()).To(BeTrue(), "Expect no requeue to be requested")

	reconciled, err := getMicrovmMachine(client, testMachineName, testClusterNamespace)
	g.Expect(err).NotTo(HaveOccurred(), "Getting microvm machine should not fail")
	assertMachineReconciled(g, reconciled)
}

func TestMachineReconcileMachineExistsAndPending(t *testing.T) {
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()

	fakeAPIClient := mock_client.FakeClient{}
	withExistingMicrovm(&fakeAPIClient, flintlocktypes.MicroVMStatus_PENDING)

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	result, err := reconcileMachine(client, &fakeAPIClient)
	g.Expect(err).NotTo(HaveOccurred(), "Reconciling when microvm service exists and state pending should not return error")
	g.Expect(result.IsZero()).To(BeFalse(), "Expect a requeue to be requested")

	reconciled, err := getMicrovmMachine(client, testMachineName, testClusterNamespace)
	g.Expect(err).NotTo(HaveOccurred(), "Getting microvm machine should not fail")

	assertConditionFalse(g, reconciled, infrav1.MicrovmReadyCondition, infrav1.MicrovmPendingReason)
	assertMachineVMState(g, reconciled, infrav1.VMStatePending)
	assertMachineFinalizer(g, reconciled)
}

func TestMachineReconcileMachineExistsButFailed(t *testing.T) {
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()

	fakeAPIClient := mock_client.FakeClient{}
	withExistingMicrovm(&fakeAPIClient, flintlocktypes.MicroVMStatus_FAILED)

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	_, err := reconcileMachine(client, &fakeAPIClient)
	g.Expect(err).To(HaveOccurred(), "Reconciling when microvm service exists and state failed should return an error")

	reconciled, err := getMicrovmMachine(client, testMachineName, testClusterNamespace)
	g.Expect(err).NotTo(HaveOccurred(), "Getting microvm machine should not fail")

	assertConditionFalse(g, reconciled, infrav1.MicrovmReadyCondition, infrav1.MicrovmProvisionFailedReason)
	assertMachineVMState(g, reconciled, infrav1.VMStateFailed)
	assertMachineFinalizer(g, reconciled)
}

func TestMachineReconcileMachineExistsButUnknownState(t *testing.T) {
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()

	fakeAPIClient := mock_client.FakeClient{}
	withExistingMicrovm(&fakeAPIClient, flintlocktypes.MicroVMStatus_MicroVMState(42))

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	_, err := reconcileMachine(client, &fakeAPIClient)
	g.Expect(err).To(HaveOccurred(), "Reconciling when microvm service exists and state is unknown should return an error")

	reconciled, err := getMicrovmMachine(client, testMachineName, testClusterNamespace)
	g.Expect(err).NotTo(HaveOccurred(), "Getting microvm machine should not fail")

	assertConditionFalse(g, reconciled, infrav1.MicrovmReadyCondition, infrav1.MicrovmUnknownStateReason)
	assertMachineVMState(g, reconciled, infrav1.VMStateUnknown)
	assertMachineFinalizer(g, reconciled)
}

func TestMachineReconcileNoVmCreateSucceeds(t *testing.T) {
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()
	apiObjects.MvmMachine.Spec.ProviderID = nil

	fakeAPIClient := mock_client.FakeClient{}
	withMissingMicrovm(&fakeAPIClient)
	withCreateMicrovmSuccess(&fakeAPIClient)

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	result, err := reconcileMachine(client, &fakeAPIClient)

	g.Expect(err).NotTo(HaveOccurred(), "Reconciling when creating microvm should not return error")
	g.Expect(result.IsZero()).To(BeFalse(), "Expect requeue to be requested after create")

	_, createReq, _ := fakeAPIClient.CreateMicroVMArgsForCall(0)
	g.Expect(createReq.Microvm).ToNot(BeNil())
	g.Expect(createReq.Microvm.Labels).To(HaveLen(1))
	g.Expect(createReq.Microvm.Metadata).To(HaveLen(3))
	expectedBootstrapData := base64.StdEncoding.EncodeToString([]byte(testbootStrapData))
	g.Expect(createReq.Microvm.Metadata).To(HaveKeyWithValue("user-data", expectedBootstrapData))

	reconciled, err := getMicrovmMachine(client, testMachineName, testClusterNamespace)
	g.Expect(err).NotTo(HaveOccurred(), "Getting microvm machine should not fail")

	expectedProviderID := fmt.Sprintf("microvm://127.0.0.1:9090/%s", testMachineUID)
	g.Expect(reconciled.Spec.ProviderID).To(Equal(pointer.String(expectedProviderID)))

	assertConditionFalse(g, reconciled, infrav1.MicrovmReadyCondition, infrav1.MicrovmPendingReason)
	assertMachineVMState(g, reconciled, infrav1.VMStatePending)
	assertMachineFinalizer(g, reconciled)
}

func TestMachineReconcileNoMachineFailureDomainCreateSucceeds(t *testing.T) {
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()
	apiObjects.MvmMachine.Spec.ProviderID = nil
	apiObjects.Machine.Spec.FailureDomain = nil

	fakeAPIClient := mock_client.FakeClient{}
	withMissingMicrovm(&fakeAPIClient)
	withCreateMicrovmSuccess(&fakeAPIClient)

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	result, err := reconcileMachine(client, &fakeAPIClient)

	g.Expect(err).NotTo(HaveOccurred(), "Reconciling when creating microvm should not return error")
	g.Expect(result.IsZero()).To(BeFalse(), "Expect requeue to be requested after create")

	reconciled, err := getMicrovmMachine(client, testMachineName, testClusterNamespace)
	g.Expect(err).NotTo(HaveOccurred(), "Getting microvm machine should not fail")
	assertConditionFalse(g, reconciled, infrav1.MicrovmReadyCondition, infrav1.MicrovmPendingReason)
	assertMachineVMState(g, reconciled, infrav1.VMStatePending)
	assertMachineFinalizer(g, reconciled)
}

func TestMachineReconcileNoVmCreateClusterSSHSucceeds(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	expectedKeys := []v1alpha1.SSHPublicKey{{
		User:           "ubuntu",
		AuthorizedKeys: []string{"ClusterSSH"},
	}}

	apiObjects := defaultClusterObjects()
	apiObjects.MvmMachine.Spec.ProviderID = nil
	apiObjects.MvmMachine.Spec.SSHPublicKeys = expectedKeys

	fakeAPIClient := mock_client.FakeClient{}
	withMissingMicrovm(&fakeAPIClient)
	withCreateMicrovmSuccess(&fakeAPIClient)

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	result, err := reconcileMachine(client, &fakeAPIClient)

	g.Expect(err).NotTo(HaveOccurred(), "Reconciling when creating microvm should not return error")
	g.Expect(result.IsZero()).To(BeFalse(), "Expect requeue to be requested after create")

	_, createReq, _ := fakeAPIClient.CreateMicroVMArgsForCall(0)
	g.Expect(createReq.Microvm).ToNot(BeNil())
	g.Expect(createReq.Microvm.Labels).To(HaveLen(1))
	g.Expect(createReq.Microvm.Metadata).To(HaveLen(3))

	expectedBootstrapData := base64.StdEncoding.EncodeToString([]byte(testbootStrapData))
	g.Expect(createReq.Microvm.Metadata).To(HaveKeyWithValue("user-data", expectedBootstrapData))

	g.Expect(createReq.Microvm.Metadata).To(HaveKey("vendor-data"), "expect cloud-init vendor-data to be created")
	assertVendorData(g, createReq.Microvm.Metadata["vendor-data"], expectedKeys)
}

func TestMachineReconcileNoVmCreateClusterMachineSSHSucceeds(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	expectedKeys := []v1alpha1.SSHPublicKey{{
		AuthorizedKeys: []string{"MachineSSH"},
		User:           "root",
	}, {
		AuthorizedKeys: []string{"MachineSSH"},
		User:           "ubuntu",
	}}

	apiObjects := defaultClusterObjects()
	apiObjects.MvmMachine.Spec.ProviderID = nil
	apiObjects.MvmCluster.Spec.SSHPublicKeys = []v1alpha1.SSHPublicKey{{AuthorizedKeys: []string{"ClusterSSH"}}}
	apiObjects.MvmMachine.Spec.SSHPublicKeys = expectedKeys

	fakeAPIClient := mock_client.FakeClient{}
	withMissingMicrovm(&fakeAPIClient)
	withCreateMicrovmSuccess(&fakeAPIClient)

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	result, err := reconcileMachine(client, &fakeAPIClient)
	g.Expect(err).NotTo(HaveOccurred(), "Reconciling when creating microvm should not return error")
	g.Expect(result.IsZero()).To(BeFalse(), "Expect requeue to be requested after create")

	_, createReq, _ := fakeAPIClient.CreateMicroVMArgsForCall(0)
	g.Expect(createReq.Microvm).ToNot(BeNil())
	g.Expect(createReq.Microvm.Labels).To(HaveLen(1))
	g.Expect(createReq.Microvm.Metadata).To(HaveLen(3))

	expectedBootstrapData := base64.StdEncoding.EncodeToString([]byte(testbootStrapData))
	g.Expect(createReq.Microvm.Metadata).To(HaveKeyWithValue("user-data", expectedBootstrapData))

	g.Expect(createReq.Microvm.Metadata).To(HaveKey("vendor-data"), "expect cloud-init vendor-data to be created")
	assertVendorData(g, createReq.Microvm.Metadata["vendor-data"], expectedKeys)
}

func TestMachineReconcileNoVmCreateAdditionReconcile(t *testing.T) {
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()
	apiObjects.MvmMachine.Spec.ProviderID = nil

	fakeAPIClient := mock_client.FakeClient{}
	withMissingMicrovm(&fakeAPIClient)
	withCreateMicrovmSuccess(&fakeAPIClient)

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	result, err := reconcileMachine(client, &fakeAPIClient)
	g.Expect(err).NotTo(HaveOccurred(), "Reconciling when creating microvm should not return error")
	g.Expect(result.IsZero()).To(BeFalse(), "Expect requeue to be requested after create")

	withExistingMicrovm(&fakeAPIClient, flintlocktypes.MicroVMStatus_CREATED)
	_, err = reconcileMachine(client, &fakeAPIClient)
	g.Expect(err).NotTo(HaveOccurred(), "Reconciling should not return an error")

	reconciled, err := getMicrovmMachine(client, testMachineName, testClusterNamespace)
	g.Expect(err).NotTo(HaveOccurred(), "Getting microvm machine should not fail")
	assertMachineReconciled(g, reconciled)
}

func TestMachineReconcileDeleteVmSucceeds(t *testing.T) {
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()
	apiObjects.MvmMachine.DeletionTimestamp = &metav1.Time{
		Time: time.Now(),
	}
	apiObjects.MvmMachine.Spec.ProviderID = pointer.String(fmt.Sprintf("microvm://127.0.0.1:9090/%s", testMachineUID))
	apiObjects.MvmMachine.Finalizers = []string{v1alpha1.MachineFinalizer}

	fakeAPIClient := mock_client.FakeClient{}
	withExistingMicrovm(&fakeAPIClient, flintlocktypes.MicroVMStatus_CREATED)

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())

	result, err := reconcileMachine(client, &fakeAPIClient)
	g.Expect(err).NotTo(HaveOccurred(), "Reconciling when deleting microvm should not return error")
	g.Expect(result.Requeue).To(BeFalse())
	g.Expect(result.RequeueAfter).To(BeNumerically(">", time.Duration(0)))

	g.Expect(fakeAPIClient.DeleteMicroVMCallCount()).To(Equal(1))
	_, deleteReq, _ := fakeAPIClient.DeleteMicroVMArgsForCall(0)
	g.Expect(deleteReq.Uid).To(Equal(testMachineUID))

	_, err = getMicrovmMachine(client, testMachineName, testClusterNamespace)
	g.Expect(apierrors.IsNotFound(err)).To(BeFalse())
}

func TestMachineReconcileDeleteGetReturnsNil(t *testing.T) {
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()
	apiObjects.MvmMachine.DeletionTimestamp = &metav1.Time{
		Time: time.Now(),
	}
	apiObjects.MvmMachine.Finalizers = []string{v1alpha1.MachineFinalizer}

	fakeAPIClient := mock_client.FakeClient{}
	withMissingMicrovm(&fakeAPIClient)

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())

	result, err := reconcileMachine(client, &fakeAPIClient)
	g.Expect(err).NotTo(HaveOccurred(), "Reconciling when deleting microvm should not return error")
	g.Expect(result.Requeue).To(BeFalse())
	g.Expect(result.RequeueAfter).To(Equal(time.Duration(0)))

	g.Expect(fakeAPIClient.DeleteMicroVMCallCount()).To(Equal(0))

	_, err = getMicrovmMachine(client, testMachineName, testClusterNamespace)
	g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
}

func TestMachineReconcileDeleteGetErrors(t *testing.T) {
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()
	apiObjects.MvmMachine.DeletionTimestamp = &metav1.Time{
		Time: time.Now(),
	}
	apiObjects.MvmMachine.Finalizers = []string{v1alpha1.MachineFinalizer}

	fakeAPIClient := mock_client.FakeClient{}
	withExistingMicrovm(&fakeAPIClient, flintlocktypes.MicroVMStatus_CREATED)
	fakeAPIClient.GetMicroVMReturns(nil, errors.New("something terrible happened"))

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	_, err := reconcileMachine(client, &fakeAPIClient)
	g.Expect(err).To(HaveOccurred(), "Reconciling when microvm service exists errors should return error")
}

func TestMachineReconcileDeleteDeleteErrors(t *testing.T) {
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()
	apiObjects.MvmMachine.DeletionTimestamp = &metav1.Time{
		Time: time.Now(),
	}
	apiObjects.MvmMachine.Finalizers = []string{v1alpha1.MachineFinalizer}

	fakeAPIClient := mock_client.FakeClient{}
	withExistingMicrovm(&fakeAPIClient, flintlocktypes.MicroVMStatus_CREATED)
	fakeAPIClient.DeleteMicroVMReturns(nil, errors.New("something terrible happened"))

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	_, err := reconcileMachine(client, &fakeAPIClient)
	g.Expect(err).To(HaveOccurred(), "Reconciling when deleting microvm errors should return error")

	reconciled, err := getMicrovmMachine(client, testMachineName, testClusterNamespace)
	g.Expect(err).NotTo(HaveOccurred(), "Getting microvm machine should not fail")

	assertConditionFalse(g, reconciled, infrav1.MicrovmReadyCondition, infrav1.MicrovmDeleteFailedReason)
	assertMachineNotReady(g, reconciled)
}
