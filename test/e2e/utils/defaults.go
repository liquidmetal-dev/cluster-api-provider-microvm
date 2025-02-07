//go:build e2e
// +build e2e

package utils

import (
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/runtime"
	cgscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/cluster-api/test/framework"

	infrav1 "github.com/liquidmetal-dev/cluster-api-provider-microvm/api/v1alpha1"
)

const (
	// DefaultE2EConfig is the default loction for the E2E config file.
	DefaultE2EConfig = "config/e2e_conf.yaml"
	// DefaultKubernetesVersion is the default version of Kubernetes which will
	// the workload cluster will run.
	DefaultKubernetesVersion = "1.21.8"
	// DefaultVIPAddress is the default address which the workload cluster's
	// load balancer will use.
	DefaultVIPAddress = "192.168.1.25"

	DefaultSkipCleanup     = false
	DefaultExistingCluster = false
)

// Flavour consts.
const (
	Vanilla = ""
	Cilium  = "cilium"
)

// DefaultScheme returns the default scheme to use for testing.
func DefaultScheme() *runtime.Scheme {
	sc := runtime.NewScheme()
	framework.TryAddDefaultSchemes(sc)
	_ = infrav1.AddToScheme(sc)
	_ = cgscheme.AddToScheme(sc)

	return sc
}

func DefaultArtefactDir() string {
	pwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	return filepath.Join(pwd, "_artefacts")
}
