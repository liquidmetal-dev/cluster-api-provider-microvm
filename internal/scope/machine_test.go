package scope_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	infrav1 "github.com/weaveworks-liquidmetal/cluster-api-provider-microvm/api/v1alpha1"
	"github.com/weaveworks-liquidmetal/cluster-api-provider-microvm/internal/scope"
)

func TestMachineProviderID(t *testing.T) {
	RegisterTestingT(t)

	scheme, err := setupScheme()
	Expect(err).NotTo(HaveOccurred())

	clusterName := "testcluster"
	cluster := newCluster(clusterName, []string{"fd1", "fd2"})
	mvmCluster := newMicrovmCluster(clusterName)

	machineName := "machine-1"
	machine := newMachine(clusterName, machineName)
	mvmMachine := newMicrovmMachine(clusterName, machineName, "")

	initObjects := []client.Object{
		cluster, mvmCluster, machine, mvmMachine,
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()
	machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
		Client:         client,
		Cluster:        cluster,
		MicroVMCluster: mvmCluster,
		Machine:        machine,
		MicroVMMachine: mvmMachine,
	})
	Expect(err).NotTo(HaveOccurred())

	machineScope.SetProviderID("fd1", "abcdef")
	providerID := machineScope.GetProviderID()

	Expect(providerID).To(Equal("microvm://fd1/abcdef"))
}

func TestMachineGetInstanceID(t *testing.T) {
	RegisterTestingT(t)

	scheme, err := setupScheme()
	Expect(err).NotTo(HaveOccurred())

	clusterName := "testcluster"
	cluster := newCluster(clusterName, []string{"fd1", "fd2"})
	mvmCluster := newMicrovmCluster(clusterName)

	machineName := "machine-1"
	machine := newMachine(clusterName, machineName)
	mvmMachine := newMicrovmMachine(clusterName, machineName, "microvm://fd1/abcdefg")

	initObjects := []client.Object{
		cluster, mvmCluster, machine, mvmMachine,
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()
	machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
		Client:         client,
		Cluster:        cluster,
		MicroVMCluster: mvmCluster,
		Machine:        machine,
		MicroVMMachine: mvmMachine,
	})
	Expect(err).NotTo(HaveOccurred())

	instanceID := machineScope.GetInstanceID()
	Expect(instanceID).To(Equal("abcdefg"))
}

func TestMachineGetBasicAuthToken(t *testing.T) {
	RegisterTestingT(t)

	scheme, err := setupScheme()
	Expect(err).NotTo(HaveOccurred())

	clusterName := "testcluster"
	secretName := "testsecret"
	hostName := "hostwiththemost"
	token := "foo"

	mvmCluster := newMicrovmClusterWithAuth(clusterName, secretName)
	otherCluster := newMicrovmCluster(clusterName)
	secret := newSecret(secretName, hostName, token)
	otherSecret := newSecret(secretName, "differentone", token)

	tt := []struct {
		name        string
		expected    string
		expectedErr func(error)
		initObjects []client.Object
	}{
		{
			name: "when the token is found in the secret, it is returned",
			initObjects: []client.Object{
				mvmCluster, secret,
			},
			expected: token,
			expectedErr: func(err error) {
				Expect(err).NotTo(HaveOccurred())
			},
		},
		{
			name:        "when the secret does not exist, returns the error",
			initObjects: []client.Object{mvmCluster},
			expected:    "",
			expectedErr: func(err error) {
				Expect(err).To(HaveOccurred())
			},
		},
		{
			name:        "when the secret does not contain hostname key, empty string is returned",
			initObjects: []client.Object{mvmCluster, otherSecret},
			expected:    "",
			expectedErr: func(err error) {
				Expect(err).NotTo(HaveOccurred())
			},
		},
		{
			name:        "when the secret name is not set on the cluster, empty string is returned",
			initObjects: []client.Object{otherCluster, otherSecret},
			expected:    "",
			expectedErr: func(err error) {
				Expect(err).NotTo(HaveOccurred())
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			RegisterTestingT(t)
			client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tc.initObjects...).Build()
			machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
				Client:         client,
				Cluster:        &clusterv1.Cluster{},
				MicroVMCluster: mvmCluster,
				Machine:        &clusterv1.Machine{},
				MicroVMMachine: &infrav1.MicrovmMachine{},
			})
			Expect(err).NotTo(HaveOccurred())

			token, err := machineScope.GetBasicAuthToken(hostName)
			tc.expectedErr(err)
			Expect(token).To(Equal(tc.expected))
		})
	}
}

func TestMachineRandomFailureDomain(t *testing.T) {
	RegisterTestingT(t)

	scheme, err := setupScheme()
	Expect(err).NotTo(HaveOccurred())

	failureDomains := []string{"fd1", "fd2"}
	failureDomainCounts := map[string]int{
		"fd1": 0,
		"fd2": 1,
	}

	clusterName := "testcluster"
	cluster := newCluster(clusterName, failureDomains)
	mvmCluster := newMicrovmCluster(clusterName)

	for i := 0; i < 10; i++ {
		machineName := fmt.Sprintf("machine-%d", i)
		machine := newMachine(clusterName, machineName)
		mvmMachine := newMicrovmMachine(clusterName, machineName, "")

		initObjects := []client.Object{
			cluster, mvmCluster, machine, mvmMachine,
		}

		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()
		machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
			Client:         client,
			Cluster:        cluster,
			MicroVMCluster: mvmCluster,
			Machine:        machine,
			MicroVMMachine: mvmMachine,
		})
		Expect(err).NotTo(HaveOccurred())

		addr, err := machineScope.GetFailureDomain()
		Expect(err).NotTo(HaveOccurred())

		count, ok := failureDomainCounts[addr]
		Expect(ok).To(BeTrue(), "unexpected address selected")
		failureDomainCounts[addr] = count + 1
	}

	for _, fdCount := range failureDomainCounts {
		Expect(fdCount).To(BeNumerically(">", 3), "failuredomain count is expected to be greater than 3")
	}
}

func TestMachineFailureDomainFromMachine(t *testing.T) {
	RegisterTestingT(t)

	scheme, err := setupScheme()
	Expect(err).NotTo(HaveOccurred())

	clusterName := "testcluster"
	cluster := newCluster(clusterName, []string{"fd1", "fd2"})
	mvmCluster := newMicrovmCluster(clusterName)

	machineName := "machine-1"
	machine := newMachine(clusterName, machineName)
	machine.Spec.FailureDomain = pointer.String("fd2")
	mvmMachine := newMicrovmMachine(clusterName, machineName, "")

	initObjects := []client.Object{
		cluster, mvmCluster, machine, mvmMachine,
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()
	machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
		Client:         client,
		Cluster:        cluster,
		MicroVMCluster: mvmCluster,
		Machine:        machine,
		MicroVMMachine: mvmMachine,
	})
	Expect(err).NotTo(HaveOccurred())

	failureDomain, err := machineScope.GetFailureDomain()
	Expect(err).NotTo(HaveOccurred())
	Expect(failureDomain).To(Equal("fd2"))
}

func TestMachineFailureDomainFromProviderID(t *testing.T) {
	RegisterTestingT(t)

	scheme, err := setupScheme()
	Expect(err).NotTo(HaveOccurred())

	clusterName := "testcluster"
	cluster := newCluster(clusterName, []string{"fd1", "fd2"})
	mvmCluster := newMicrovmCluster(clusterName)

	machineName := "machine-1"
	machine := newMachine(clusterName, machineName)
	mvmMachine := newMicrovmMachine(clusterName, machineName, "microvm://fd2/abcdef")

	initObjects := []client.Object{
		cluster, mvmCluster, machine, mvmMachine,
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()
	machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
		Client:         client,
		Cluster:        cluster,
		MicroVMCluster: mvmCluster,
		Machine:        machine,
		MicroVMMachine: mvmMachine,
	})
	Expect(err).NotTo(HaveOccurred())

	failureDomain, err := machineScope.GetFailureDomain()
	Expect(err).NotTo(HaveOccurred())
	Expect(failureDomain).To(Equal("fd2"))
}

func setupScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := infrav1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := clusterv1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	return scheme, nil
}

func newMachine(clusterName, machineName string) *clusterv1.Machine {
	return &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				clusterv1.ClusterLabelName: clusterName,
			},
			Name:      machineName,
			Namespace: "default",
		},
		Spec: clusterv1.MachineSpec{
			ClusterName: clusterName,
			Bootstrap: clusterv1.Bootstrap{
				DataSecretName: pointer.StringPtr(machineName),
			},
		},
	}
}

func newCluster(name string, failureDomains []string) *clusterv1.Cluster {
	cluster := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
	}

	if len(failureDomains) > 0 {
		cluster.Status = clusterv1.ClusterStatus{
			FailureDomains: make(clusterv1.FailureDomains),
		}

		for i := range failureDomains {
			fd := failureDomains[i]
			cluster.Status.FailureDomains[fd] = clusterv1.FailureDomainSpec{
				ControlPlane: true,
			}
		}
	}

	return cluster
}

func newMicrovmCluster(name string) *infrav1.MicrovmCluster {
	return &infrav1.MicrovmCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
	}
}

func newMicrovmMachine(clusterName, machineName string, providerID string) *infrav1.MicrovmMachine {
	mvmMachine := &infrav1.MicrovmMachine{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				clusterv1.ClusterLabelName: clusterName,
			},
			Name:      machineName,
			Namespace: "default",
		},
	}
	if providerID != "" {
		mvmMachine.Spec.ProviderID = &providerID
	}

	return mvmMachine
}

func newMicrovmClusterWithAuth(name, secretname string) *infrav1.MicrovmCluster {
	cluster := newMicrovmCluster(name)
	cluster.Spec.Placement = infrav1.Placement{
		StaticPool: &infrav1.StaticPoolPlacement{
			BasicAuthSecret: secretname,
		},
	}
	return cluster
}

func newSecret(name, address, token string) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Data: map[string][]byte{
			address: []byte(token),
		},
	}
}
