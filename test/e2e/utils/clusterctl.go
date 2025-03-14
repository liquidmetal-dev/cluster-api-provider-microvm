//go:build e2e
// +build e2e

package utils

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util/yaml"

	"github.com/liquidmetal-dev/cluster-api-provider-microvm/api/v1alpha1"
)

// ApplyClusterInput tidies up the input params for ApplyClusterTemplateAndWait
// because they were getting a bit messy.
type ApplyClusterInput struct {
	Hosts  []string
	Input  clusterctl.ApplyClusterTemplateAndWaitInput
	Result *clusterctl.ApplyClusterTemplateAndWaitResult
}

// This is a dupe of clusterctl.ApplyClusterTemplateAndWait().
// We needed to do more with the template after clusterctl.ConfigCluster so
// we copied the calling func over for now. Go see that func if you think there
// are pieces missing here.
//
// ApplyClusterTemplateAndWait gets a cluster template using clusterctl, and waits for the cluster to be ready.
// Important! this method assumes the cluster uses a KubeadmControlPlane and MachineDeployments.
func ApplyClusterTemplateAndWait(ctx context.Context, params ApplyClusterInput) {
	input := params.Input
	result := params.Result

	Expect(ctx).NotTo(BeNil(), "ctx is required for ApplyClusterTemplateAndWait")
	Expect(input.ClusterProxy).ToNot(BeNil(), "Invalid argument. input.ClusterProxy can't be nil when calling ApplyClusterTemplateAndWait")
	Expect(result).ToNot(BeNil(), "Invalid argument. result can't be nil when calling ApplyClusterTemplateAndWait")
	Expect(input.ConfigCluster.ControlPlaneMachineCount).ToNot(BeNil())
	Expect(input.ConfigCluster.WorkerMachineCount).ToNot(BeNil())

	// Ensure we have a Cluster for dump and cleanup steps in AfterEach even if ApplyClusterTemplateAndWait fails.
	result.Cluster = &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.ConfigCluster.ClusterName,
			Namespace: input.ConfigCluster.Namespace,
		},
	}

	By("Getting the cluster template yaml")

	workloadClusterTemplate := clusterctl.ConfigCluster(ctx, clusterctl.ConfigClusterInput{
		// pass reference to the management cluster hosting this test
		KubeconfigPath: input.ConfigCluster.KubeconfigPath,
		// pass the clusterctl config file that points to the local provider repository created for this test,
		ClusterctlConfigPath: input.ConfigCluster.ClusterctlConfigPath,
		// select template
		Flavor: input.ConfigCluster.Flavor,
		// define template variables
		Namespace:                input.ConfigCluster.Namespace,
		ClusterName:              input.ConfigCluster.ClusterName,
		KubernetesVersion:        input.ConfigCluster.KubernetesVersion,
		ControlPlaneMachineCount: input.ConfigCluster.ControlPlaneMachineCount,
		WorkerMachineCount:       input.ConfigCluster.WorkerMachineCount,
		InfrastructureProvider:   input.ConfigCluster.InfrastructureProvider,
		// setup clusterctl logs folder
		LogFolder: input.ConfigCluster.LogFolder,
	})
	Expect(workloadClusterTemplate).ToNot(BeNil(), "Failed to get the cluster template")

	By("Adding provided flintlock hosts to MicrovmCluster template")
	workloadClusterTemplate = addFlintlockHostsToTemplate(params.Hosts, workloadClusterTemplate)

	clusterctl.ApplyCustomClusterTemplateAndWait(ctx, clusterctl.ApplyCustomClusterTemplateAndWaitInput{
		ClusterProxy:                 input.ClusterProxy,
		CustomTemplateYAML:           workloadClusterTemplate,
		ClusterName:                  input.ConfigCluster.ClusterName,
		Namespace:                    input.ConfigCluster.Namespace,
		CNIManifestPath:              input.CNIManifestPath,
		Flavor:                       input.ConfigCluster.Flavor,
		WaitForClusterIntervals:      input.WaitForClusterIntervals,
		WaitForControlPlaneIntervals: input.WaitForControlPlaneIntervals,
		WaitForMachineDeployments:    input.WaitForMachineDeployments,
		WaitForMachinePools:          input.WaitForMachinePools,
		CreateOrUpdateOpts:           input.CreateOrUpdateOpts,
		PreWaitForCluster:            input.PreWaitForCluster,
		PostMachinesProvisioned:      input.PostMachinesProvisioned,
		ControlPlaneWaiters:          input.ControlPlaneWaiters,
	}, (*clusterctl.ApplyCustomClusterTemplateAndWaitResult)(result))
}

// This disaster was not copied over from anywhere, it is all ours!
// It exists because we need to alter the template after it is generated to add
// in the given flintlockAddresses of which there could be any number, so we can't
// rely on hardcoded template vars.
func addFlintlockHostsToTemplate(flintlockAddresses []string, ymlBytes []byte) []byte {
	// We receive the generated yaml template in raw bytes. This needs to be
	// converted to unstructured.Unstructured using the CAPI yaml lib.
	template, err := yaml.ToUnstructured(ymlBytes)
	Expect(err).NotTo(HaveOccurred())

	// From that we can get the [i]st Object (this is a map[string]interface)
	// which we know is the MicrovmCluster part of the template.
	// We put just that piece back into bytes.
	var (
		clusterBytes    []byte
		mvmClusterIndex int
	)

	for i, o := range template {
		if o.Object["kind"] == "MicrovmCluster" {
			clusterBytes, err = json.Marshal(template[i].Object)
			Expect(err).NotTo(HaveOccurred())

			// save this for later so we know where to slot this card back in the deck
			mvmClusterIndex = i

			break
		}

		continue
	}

	Expect(clusterBytes).NotTo(HaveLen(0), "MicrovmCluster object not found in generated template")

	// We can then Unmarshal those bytes into a MicrovmCluster object. Going
	// backwards and forwards like this is easier than trying to do what we need
	// with a map[string]interface.
	mvmCluster := v1alpha1.MicrovmCluster{}
	Expect(json.Unmarshal(clusterBytes, &mvmCluster)).To(Succeed())

	// Now we have an easy object to add flintlock host addresses to.
	hosts := []v1alpha1.MicrovmHost{}
	for _, addr := range flintlockAddresses {
		hosts = append(hosts, v1alpha1.MicrovmHost{
			Endpoint:            addr,
			ControlPlaneAllowed: true,
		})
	}

	mvmCluster.Spec.Placement.StaticPool.Hosts = hosts

	// Now we go back the other way: we Marshal the edited MicrovmCluster back
	// into bytes.
	editedClusterBytes, err := json.Marshal(mvmCluster)
	Expect(err).NotTo(HaveOccurred())

	// Then we Unmarshal to a new Unstructured object.
	editedTemplateObj := unstructured.Unstructured{}
	Expect(json.Unmarshal(editedClusterBytes, &editedTemplateObj)).To(Succeed())

	// We pop the new Unstructured object back into the original template.
	template[mvmClusterIndex] = editedTemplateObj

	// And finally we use the CAPI yaml lib to convert that last object into the
	// raw yaml byte template we need.
	ret, err := yaml.FromUnstructured(template)
	Expect(err).NotTo(HaveOccurred())

	// simples ;)
	return ret
}
