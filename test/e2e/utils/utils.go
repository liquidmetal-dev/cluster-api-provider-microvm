//go:build e2e
// +build e2e

package utils

import (
	"context"
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/liquidmetal-dev/cluster-api-provider-microvm/internal/scope"
)

// SetEnvVar adds some logging around which vars are set for the test.
func SetEnvVar(key, value string, private bool) {
	printableValue := "*******"
	if !private {
		printableValue = value
	}

	By(fmt.Sprintf("Setting environment variable: key=%s, value=%s", key, printableValue))
	os.Setenv(key, value)
}

// FailureDomainSpread processes Control Plane and Deployment Machines to get
// a list of unique flintlock addresses. The total of that list is returned.
func FailureDomainSpread(proxy framework.ClusterProxy, namespace, clusterName string) int {
	lister := proxy.GetClient()
	inClustersNamespaceListOption := client.InNamespace(namespace)
	matchClusterListOption := client.MatchingLabels{
		clusterv1.ClusterNameLabel: clusterName,
	}

	machineList := &clusterv1.MachineList{}
	Expect(lister.List(context.Background(), machineList, inClustersNamespaceListOption, matchClusterListOption)).
		To(Succeed(), "Couldn't list machines for the cluster %q", clusterName)

	failureDomainCounts := map[string]int{}

	for _, machine := range machineList.Items {
		// ControlPlane machines will have the FailureDomain explicitly set.
		if machine.Spec.FailureDomain != nil {
			failureDomainCounts[*machine.Spec.FailureDomain]++

			continue
		}

		// Deployment machines will not have a FailureDomain, but CAPMVM writes it
		// into the ProviderID so we can extract it from there.
		if machine.Spec.ProviderID != nil {
			providerID := strings.ReplaceAll(*machine.Spec.ProviderID, scope.ProviderPrefix, "")
			parts := strings.Split(providerID, "/")
			failureDomainCounts[parts[0]]++

			continue
		}
	}

	return len(failureDomainCounts)
}

// Nginx returns a simple nginx deployment for testing.
func Nginx(name, namespace string, reps int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Replicas: pointer.Int32Ptr(reps),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "nginx",
						Image: "nginx:1.14.2",
						Ports: []corev1.ContainerPort{{
							ContainerPort: 80, //nolint: gomnd // this is fine.
						}},
					}},
				},
			},
		},
	}
}
