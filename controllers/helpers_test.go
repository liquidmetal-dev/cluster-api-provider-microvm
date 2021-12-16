// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0.

package controllers_test

import (
	"context"
	"encoding/base64"

	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	fakeremote "sigs.k8s.io/cluster-api/controllers/remote/fake"
	"sigs.k8s.io/cluster-api/util/conditions"

	infrav1 "github.com/weaveworks/cluster-api-provider-microvm/api/v1alpha1"
	"github.com/weaveworks/cluster-api-provider-microvm/controllers"
	"github.com/weaveworks/cluster-api-provider-microvm/internal/services/microvm"
	"github.com/weaveworks/cluster-api-provider-microvm/internal/services/microvm/mock_client"
	flintlockv1 "github.com/weaveworks/flintlock/api/services/microvm/v1alpha1"
	flintlocktypes "github.com/weaveworks/flintlock/api/types"
	"github.com/weaveworks/flintlock/client/cloudinit"
)

const (
	testClusterName         = "tenant1"
	testClusterNamespace    = "ns1"
	testMachineName         = "machine1"
	testBootstrapSecretName = "bootstrap"
	testVMID                = "id1234567890"
	testbootStrapData       = "somesamplebootstrapsdata"
)

func defaultClusterObjects() clusterObjects {
	return clusterObjects{
		Cluster:         createCluster(testClusterName, testClusterNamespace),
		MvmCluster:      createMicrovmCluster(testClusterName, testClusterNamespace),
		Machine:         createMachine(testMachineName, testClusterNamespace),
		MvmMachine:      createMicrovmMachine(testMachineName, testClusterNamespace),
		BootstrapSecret: createBootsrapSecret(testBootstrapSecretName, testClusterNamespace),
	}
}

type clusterObjects struct {
	Cluster    *clusterv1.Cluster
	MvmCluster *infrav1.MicrovmCluster

	Machine    *clusterv1.Machine
	MvmMachine *infrav1.MicrovmMachine

	BootstrapSecret *corev1.Secret
}

func (co clusterObjects) AsRuntimeObjects() []runtime.Object {
	objects := []runtime.Object{}

	if co.Cluster != nil {
		objects = append(objects, co.Cluster)
	}
	if co.MvmCluster != nil {
		objects = append(objects, co.MvmCluster)
	}
	if co.Machine != nil {
		objects = append(objects, co.Machine)
	}
	if co.MvmMachine != nil {
		objects = append(objects, co.MvmMachine)
	}
	if co.BootstrapSecret != nil {
		objects = append(objects, co.BootstrapSecret)
	}

	return objects
}

func reconcileMachine(client client.Client, mockAPIClient microvm.Client) (ctrl.Result, error) {
	machineController := &controllers.MicrovmMachineReconciler{
		Client: client,
		MvmClientFunc: func(address string) (microvm.Client, error) {
			return mockAPIClient, nil
		},
	}

	request := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      testMachineName,
			Namespace: testClusterNamespace,
		},
	}

	return machineController.Reconcile(context.TODO(), request)
}

func reconcileCluster(client client.Client) (ctrl.Result, error) {
	clusterController := &controllers.MicrovmClusterReconciler{
		Client:             client,
		RemoteClientGetter: fakeremote.NewClusterClient,
	}

	request := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "tenant1",
			Namespace: "ns1",
		},
	}

	return clusterController.Reconcile(context.TODO(), request)
}

func getCluster(ctx context.Context, c client.Client, name, namespace string) (*clusterv1.Cluster, error) {
	clusterKey := client.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}

	cluster := &clusterv1.Cluster{}
	err := c.Get(ctx, clusterKey, cluster)
	return cluster, err
}

func getMicrovmCluster(ctx context.Context, c client.Client, name, namespace string) (*infrav1.MicrovmCluster, error) {
	clusterKey := client.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}

	cluster := &infrav1.MicrovmCluster{}
	err := c.Get(ctx, clusterKey, cluster)
	return cluster, err
}

func getMicrovmMachine(c client.Client, name, namespace string) (*infrav1.MicrovmMachine, error) {
	clusterKey := client.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}

	machine := &infrav1.MicrovmMachine{}
	err := c.Get(context.TODO(), clusterKey, machine)
	return machine, err
}

func getMachine(c client.Client, name, namespace string) (*clusterv1.Machine, error) {
	clusterKey := client.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}

	machine := &clusterv1.Machine{}
	err := c.Get(context.TODO(), clusterKey, machine)
	return machine, err
}

func createFakeClient(g *WithT, objects []runtime.Object) client.Client {
	scheme := runtime.NewScheme()

	g.Expect(infrav1.AddToScheme(scheme)).To(Succeed())
	g.Expect(clusterv1.AddToScheme(scheme)).To(Succeed())
	g.Expect(corev1.AddToScheme(scheme)).To(Succeed())

	return fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objects...).Build()
}

func createMicrovmCluster(name, namespace string) *infrav1.MicrovmCluster {
	return &infrav1.MicrovmCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "cluster.x-k8s.io/v1beta1",
					Kind:       "Cluster",
					Name:       name,
				},
			},
		},
		Spec: infrav1.MicrovmClusterSpec{
			Placement: infrav1.Placement{
				StaticPool: &infrav1.StaticPoolPlacement{
					Hosts: []infrav1.MicrovmHost{
						{
							Name:                "host1",
							Endpoint:            "127.0.0.1:9090",
							ControlPlaneAllowed: true,
						},
					},
				},
			},
		},
		Status: infrav1.MicrovmClusterStatus{},
	}
}

func createCluster(name, namespace string) *clusterv1.Cluster {
	return &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: clusterv1.ClusterSpec{
			InfrastructureRef: &corev1.ObjectReference{
				Name:      name,
				Namespace: namespace,
			},
		},
		Status: clusterv1.ClusterStatus{
			InfrastructureReady: true,
		},
	}
}

func createMicrovmMachine(name, namespace string) *infrav1.MicrovmMachine {
	return &infrav1.MicrovmMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "cluster.x-k8s.io/v1beta1",
					Kind:       "Machine",
					Name:       name,
				},
			},
		},
		Spec: infrav1.MicrovmMachineSpec{
			MicrovmSpec: infrav1.MicrovmSpec{
				VCPU:     2,
				MemoryMb: 2048,
				RootVolume: infrav1.Volume{
					Image:    "docker.io/richardcase/ubuntu-bionic-test:cloudimage_v0.0.1",
					ReadOnly: false,
				},
				Kernel: infrav1.ContainerFileSource{
					Image:    "docker.io/richardcase/ubuntu-bionic-kernel:0.0.11",
					Filename: "vmlinuz",
				},
				Initrd: &infrav1.ContainerFileSource{
					Image:    "docker.io/richardcase/ubuntu-bionic-kernel:0.0.11",
					Filename: "initrd-generic",
				},
				NetworkInterfaces: []infrav1.NetworkInterface{
					{
						GuestDeviceName: "eth0",
						GuestMAC:        "",
						Type:            infrav1.IfaceTypeMacvtap,
						Address:         "",
					},
				},
			},
		},
	}
}

func createMachine(name, namespace string) *clusterv1.Machine {
	return &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				clusterv1.ClusterLabelName: testClusterName,
			},
		},
		Spec: clusterv1.MachineSpec{
			ClusterName: testClusterName,
			InfrastructureRef: corev1.ObjectReference{
				Name: name,
			},
			Bootstrap: clusterv1.Bootstrap{
				DataSecretName: pointer.String(testBootstrapSecretName),
			},
		},
	}
}

func createBootsrapSecret(name, namespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: clusterv1.GroupVersion.String(),
					Kind:       "Cluster",
					Name:       name,
				},
			},
		},
		Data: map[string][]byte{
			"value": []byte(testbootStrapData),
		},
	}
}

func withExistingMicrovm(fc *mock_client.FakeClient, mvmState flintlocktypes.MicroVMStatus_MicroVMState) {
	fc.GetMicroVMReturns(&flintlockv1.GetMicroVMResponse{
		Microvm: &flintlocktypes.MicroVM{
			Spec: &flintlocktypes.MicroVMSpec{
				Id: testVMID,
			},
			Status: &flintlocktypes.MicroVMStatus{
				State: mvmState,
			},
		},
	}, nil)
}

func withMissingMicrovm(fc *mock_client.FakeClient) {
	fc.GetMicroVMReturns(&flintlockv1.GetMicroVMResponse{}, nil)
}

func withCreateMicrovmSuccess(fc *mock_client.FakeClient) {
	fc.CreateMicroVMReturns(&flintlockv1.CreateMicroVMResponse{}, nil)
}

func assertConditionTrue(g *WithT, from conditions.Getter, conditionType clusterv1.ConditionType) {
	c := conditions.Get(from, conditionType)
	g.Expect(c).ToNot(BeNil(), "Conditions expected to be set")
	g.Expect(c.Status).To(Equal(corev1.ConditionTrue), "Condition should be marked true")
}

func assertConditionFalse(g *WithT, from conditions.Getter, conditionType clusterv1.ConditionType, reason string) {
	c := conditions.Get(from, conditionType)
	g.Expect(c).ToNot(BeNil(), "Conditions expected to be set")
	g.Expect(c.Status).To(Equal(corev1.ConditionFalse), "Condition should be marked false")
	g.Expect(c.Reason).To(Equal(reason))
}

func assertMachineVMState(g *WithT, machine *infrav1.MicrovmMachine, expectedState infrav1.VMState) {
	g.Expect(machine.Status.VMState).NotTo(BeNil())
	g.Expect(*machine.Status.VMState).To(BeEquivalentTo(expectedState))
}

func assertMachineReconciled(g *WithT, reconciled *infrav1.MicrovmMachine) {
	assertConditionTrue(g, reconciled, infrav1.MicrovmReadyCondition)
	assertMachineVMState(g, reconciled, infrav1.VMStateRunning)
	assertMachineFinalizer(g, reconciled)
	g.Expect(reconciled.Spec.ProviderID).ToNot(BeNil())
	g.Expect(*reconciled.Spec.ProviderID).To(Equal(testVMID))
	g.Expect(reconciled.Status.Ready).To(BeTrue(), "The Ready property must be true when the machine has been reconciled")
}

func assertNoMachineFinalizer(g *WithT, reconciled *infrav1.MicrovmMachine) {
	g.Expect(hasMachineFinalizer(reconciled)).To(BeFalse(), "Expect not to have the mvm machine finalizer")
}

func assertMachineFinalizer(g *WithT, reconciled *infrav1.MicrovmMachine) {
	g.Expect(reconciled.ObjectMeta.Finalizers).NotTo(BeEmpty(), "Expected at least one finalizer to be set")
	g.Expect(hasMachineFinalizer(reconciled)).To(BeTrue(), "Expect the mvm machine finalizer")
}

func hasMachineFinalizer(machine *infrav1.MicrovmMachine) bool {
	if len(machine.ObjectMeta.Finalizers) == 0 {
		return false
	}

	for _, f := range machine.ObjectMeta.Finalizers {
		if f == infrav1.MachineFinalizer {
			return true
		}
	}

	return false
}

func assertMachineNotReady(g *WithT, machine *infrav1.MicrovmMachine) {
	g.Expect(machine.Status.Ready).To(BeFalse())
}

func assertVendorData(g *WithT, vendorDataRaw string, expectedSSHKey string) {
	g.Expect(vendorDataRaw).ToNot(Equal(""))

	data, err := base64.StdEncoding.DecodeString(vendorDataRaw)
	g.Expect(err).NotTo(HaveOccurred(), "expect vendor data to be base64 encoded")

	if expectedSSHKey != "" {
		vendorData := &cloudinit.UserData{}

		unmarshallErr := yaml.Unmarshal(data, vendorData)
		g.Expect(unmarshallErr).NotTo(HaveOccurred(), "expect vendor data to unmarshall to cloud-init userdata")
		g.Expect(vendorData.Users).NotTo(BeNil())
		users := vendorData.Users
		g.Expect(users).To(HaveLen(1))
		g.Expect(users[0].SSHAuthorizedKeys).NotTo(BeNil())
		keys := users[0].SSHAuthorizedKeys
		g.Expect(keys).To(HaveLen(1))
		g.Expect(keys[0]).To(Equal(expectedSSHKey))

		vendorDataStr := string(data)
		g.Expect(vendorDataStr).To(ContainSubstring("#cloud-config\n"))
	}
}
