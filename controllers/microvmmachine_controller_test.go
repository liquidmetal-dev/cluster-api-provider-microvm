// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0.

package controllers_test

import (
	"encoding/base64"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	flintlocktypes "github.com/weaveworks/flintlock/api/types"

	infrav1 "github.com/weaveworks/cluster-api-provider-microvm/api/v1alpha1"
	"github.com/weaveworks/cluster-api-provider-microvm/internal/services/microvm/mock_client"
)

func TestMachineReconcile(t *testing.T) {
	t.Parallel()

	t.Run("is not requeued when", func(t *testing.T) {
		t.Parallel()

		t.Run("microvm machinemissing", machineReconcileMissingMvmMachine)
		t.Run("machine owner ref not set", machineReconcileNoMachineOwnerRef)
		t.Run("cluster missing", machineReconcileMissingCluster)
		t.Run("machine has no cluster owned label", machineReconcileMachineMissingClusterLabel)
		t.Run("microvm machine is paused", machineReconcileMvmMachinePaused)
		t.Run("cluster is paused", machineReconcileClusterPaused)
		t.Run("microvm cluster missing", machineReconcileMvmClusterMissing)
		t.Run("cluster infra is not ready", machineReconcileClusterInfraNotReady)
		t.Run("bootstrap data not ready", machineReconcileBoostrapNotReady)
	})

	t.Run("reconciliation fails when", func(t *testing.T) {
		t.Parallel()

		t.Run("capi machine missing", machineReconcileMissingMachine)
		t.Run("microvm exists returns error", machineReconcileServiceGetError)
	})

	t.Run("microvm already exists", func(t *testing.T) {
		t.Parallel()

		t.Run("and microvm state is pending", machineReconcileMachineExistsAndPending)
		t.Run("and microvm state is failed", machineReconcileMachineExistsButFailed)
		t.Run("and microvm state is unknown", machineReconcileMachineExistsButUnknownState)
		t.Run("and microvm state is running", machineReconcileMachineExistsAndRunning)
	})

	t.Run("microvm non existing", func(t *testing.T) {
		t.Parallel()

		t.Run("and create microvm succeeds", machineReconcileNoVmCreateSucceeds)
		t.Run("and create microvm succeeds and reconciles again", machineReconcileNoVmCreateAdditionReconcile)
	})
}

func machineReconcileMissingMvmMachine(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()
	apiObjects.MvmMachine = nil

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	result, err := reconcileMachine(client, nil)
	g.Expect(err).NotTo(HaveOccurred(), "Reconciling when microvm machine doesn't exist should not error")
	g.Expect(result.IsZero()).To(BeTrue(), "Expect no requeue to be requested")
}

func machineReconcileNoMachineOwnerRef(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()
	apiObjects.MvmMachine.ObjectMeta.OwnerReferences = nil

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	result, err := reconcileMachine(client, nil)
	g.Expect(err).NotTo(HaveOccurred(), "Reconciling when microvm machine has no owner ref should not error")
	g.Expect(result.IsZero()).To(BeTrue(), "Expect no requeue to be requested")
}

func machineReconcileMissingCluster(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()
	apiObjects.Cluster = nil

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	result, err := reconcileMachine(client, nil)
	g.Expect(err).NotTo(HaveOccurred(), "Reconciling when cluster is missing should not error")
	g.Expect(result.IsZero()).To(BeTrue(), "Expect no requeue to be requested")
}

func machineReconcileMachineMissingClusterLabel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()
	apiObjects.Machine.ObjectMeta.Labels = map[string]string{}

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	result, err := reconcileMachine(client, nil)
	g.Expect(err).NotTo(HaveOccurred(), "Reconciling when machine is missing capi labels should not error")
	g.Expect(result.IsZero()).To(BeTrue(), "Expect no requeue to be requested")
}

func machineReconcileMvmMachinePaused(t *testing.T) {
	t.Parallel()
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

func machineReconcileClusterPaused(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()
	apiObjects.Cluster.Spec.Paused = true

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	result, err := reconcileMachine(client, nil)
	g.Expect(err).NotTo(HaveOccurred(), "Reconciling when cluster is paused should not error")
	g.Expect(result.IsZero()).To(BeTrue(), "Expect no requeue to be requested")
}

func machineReconcileMissingMachine(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()
	apiObjects.Machine = nil

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	result, err := reconcileMachine(client, nil)
	g.Expect(err).To(HaveOccurred(), "Reconciling when capi machine doesn't exist should error")
	g.Expect(result.IsZero()).To(BeTrue(), "Expect no requeue to be requested")
}

func machineReconcileMvmClusterMissing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()
	apiObjects.MvmCluster = nil

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	result, err := reconcileMachine(client, nil)
	g.Expect(err).NotTo(HaveOccurred(), "Reconciling when microvm cluster missing should not error")
	g.Expect(result.IsZero()).To(BeTrue(), "Expect no requeue to be requested")
}

func machineReconcileClusterInfraNotReady(t *testing.T) {
	t.Parallel()
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

func machineReconcileBoostrapNotReady(t *testing.T) {
	t.Parallel()
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

func machineReconcileServiceGetError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	objects := defaultClusterObjects()

	fakeAPIClient := mock_client.FakeClient{}
	fakeAPIClient.GetMicroVMReturns(nil, errors.New("something terrible happened"))

	client := createFakeClient(g, objects.AsRuntimeObjects())
	_, err := reconcileMachine(client, &fakeAPIClient)
	g.Expect(err).To(HaveOccurred(), "Reconciling when microvm service 'Get' errors should return error")
}

func machineReconcileMachineExistsAndRunning(t *testing.T) {
	t.Parallel()
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

func machineReconcileMachineExistsAndPending(t *testing.T) {
	t.Parallel()
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

func machineReconcileMachineExistsButFailed(t *testing.T) {
	t.Parallel()
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

func machineReconcileMachineExistsButUnknownState(t *testing.T) {
	t.Parallel()
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

func machineReconcileNoVmCreateSucceeds(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()

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
	expectedBootstrapData := base64.StdEncoding.EncodeToString([]byte(testbootStrapData))
	g.Expect(createReq.Microvm.Metadata).To(HaveKeyWithValue("meta-data", expectedBootstrapData))
}

func machineReconcileNoVmCreateAdditionReconcile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	apiObjects := defaultClusterObjects()

	fakeAPIClient := mock_client.FakeClient{}
	withMissingMicrovm(&fakeAPIClient)
	withCreateMicrovmSuccess(&fakeAPIClient)

	client := createFakeClient(g, apiObjects.AsRuntimeObjects())
	result, err := reconcileMachine(client, &fakeAPIClient)
	g.Expect(err).NotTo(HaveOccurred(), "Reconciling when creating microvm should not return error")
	g.Expect(result.IsZero()).To(BeFalse(), "Expect requeue to be requested after create")

	withExistingMicrovm(&fakeAPIClient, flintlocktypes.MicroVMStatus_CREATED)
	result, err = reconcileMachine(client, &fakeAPIClient)

	reconciled, err := getMicrovmMachine(client, testMachineName, testClusterNamespace)
	g.Expect(err).NotTo(HaveOccurred(), "Getting microvm machine should not fail")
	assertMachineReconciled(g, reconciled)
}
