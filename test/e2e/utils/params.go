//go:build e2e
// +build e2e

package utils

import (
	"flag"
	"strings"
)

type Params struct {
	E2EConfigPath      string
	FlintlockHosts     stringSlice
	ArtefactDir        string
	KubernetesVersion  string
	VIPAddress         string
	UseExistingCluster bool
	SkipCleanup        bool
}

func NewParams() *Params {
	params := &Params{}

	flag.StringVar(&params.E2EConfigPath, "e2e.config", DefaultE2EConfig,
		"Path to e2e config for this suite.")
	flag.Var(&params.FlintlockHosts, "e2e.flintlock-hosts",
		"Comma separated list of addresses to flintlock servers. eg '1.2.3.4:9090,5.6.7.8:9090'")
	flag.StringVar(&params.ArtefactDir, "e2e.artefact-dir", DefaultArtefactDir(),
		"Location to store test yamls, logs, etc.")
	flag.StringVar(&params.KubernetesVersion, "e2e.capmvm.kubernetes-version", DefaultKubernetesVersion,
		"Version of k8s to run in the workload cluster(s)")
	flag.StringVar(&params.VIPAddress, "e2e.capmvm.vip-address", DefaultVIPAddress,
		"Address for the kubevip load balancer.")
	flag.BoolVar(&params.SkipCleanup, "e2e.skip-cleanup", DefaultSkipCleanup,
		"Do not delete test-created workload clusters or the management kind cluster")
	flag.BoolVar(&params.UseExistingCluster, "e2e.existing-cluster", DefaultExistingCluster,
		"If true, uses the current context for the management cluster and will not create a new one.")

	return params
}

type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, " ")
}

func (s *stringSlice) Set(value string) error {
	*s = strings.Split(value, ",")

	return nil
}
