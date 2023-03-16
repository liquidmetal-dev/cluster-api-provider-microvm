//go:build e2e
// +build e2e

package utils

import (
	"context"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/bootstrap"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/test/infrastructure/container"
)

// EnvManager holds various objects required for running the tests and the test
// environment.
type EnvManager struct {
	*Params
	Cfg             *clusterctl.E2EConfig
	ClusterProxy    framework.ClusterProxy
	ClusterProvider bootstrap.ClusterProvider
	ClusterctlCfg   string
	KubeconfigPath  string

	ctx context.Context //nolint: containedctx // don't care.
}

// NewEnvManager returns a new EnvManager.
func NewEnvManager(p *Params) *EnvManager {
	return &EnvManager{
		Params: p,
		ctx:    context.TODO(),
	}
}

// Setup will prepare a local Kind cluster as a CAPI management cluster.
func (m *EnvManager) Setup() {
	// Process the command line flags, fail fast if wrong.
	m.validateInput()

	// Read the file at m.E2EConfigPath to discover providers, versions, yamls, etc
	// to be used in this test.
	m.loadE2EConfig()
	Expect(m.Cfg).NotTo(BeNil())

	// Create a new kind cluster (if UseExistingCluster is not set) and load the
	// required images.
	// Set a ClusterProxy which can be used to interact with the resulting cluster.
	m.bootstrapLocalKind()
	Expect(m.ClusterProxy).NotTo(BeNil())
	if !m.UseExistingCluster {
		Expect(m.KubeconfigPath).NotTo(BeNil())
	}

	// Create a directory for clusterctl to store configuration yamls etc.
	m.createClusterctlRepo()
	Expect(m.ClusterctlCfg).NotTo(Equal(""))

	// Use clusterctl to init the kind cluster with CAPI and CAPMVM controllers.
	// After this the management cluster is ready to accept creation of workload
	// clusters.
	m.initKindCluster()
}

// Teardown will delete the local kind management cluster and remove any
// related artefacts.
func (m *EnvManager) Teardown() {
	if !m.SkipCleanup && !m.UseExistingCluster {
		if m.ClusterProvider != nil {
			m.ClusterProvider.Dispose(m.ctx)
		}

		if m.ClusterProxy != nil {
			m.ClusterProxy.Dispose(m.ctx)
		}
	}
}

// Ctx returns the EnvManager's context so that it can be used throughout the
// suite.
func (m *EnvManager) Ctx() context.Context {
	return m.ctx
}

func (m *EnvManager) validateInput() {
	By(fmt.Sprintf("Validating test params: %#v", m.Params))
	Expect(m.FlintlockHosts).ToNot(HaveLen(0), "At least one address for a flintlock server is required.")
	Expect(m.E2EConfigPath).To(BeAnExistingFile(), "A valid path to a clusterctl.E2EConfig is required.")
	Expect(m.ArtefactDir).NotTo(Equal(""), "A valid path for the test artefacts folder is required.")
}

func (m *EnvManager) loadE2EConfig() {
	m.Cfg = clusterctl.LoadE2EConfig(m.ctx, clusterctl.LoadE2EConfigInput{
		ConfigPath: m.E2EConfigPath,
	})
}

func (m *EnvManager) bootstrapLocalKind() {
	kubeconfigPath := ""

	if !m.UseExistingCluster {
		m.pullImages()

		bootInput := bootstrap.CreateKindBootstrapClusterAndLoadImagesInput{
			Name:               m.Cfg.ManagementClusterName,
			RequiresDockerSock: m.Cfg.HasDockerProvider(),
			Images:             m.Cfg.Images,
			LogFolder:          m.ArtefactDir + "/logs/bootstrap",
		}
		m.ClusterProvider = bootstrap.CreateKindBootstrapClusterAndLoadImages(m.ctx, bootInput)

		kubeconfigPath = m.ClusterProvider.GetKubeconfigPath()
	}

	m.ClusterProxy = framework.NewClusterProxy("bootstrap", kubeconfigPath, DefaultScheme())
	m.KubeconfigPath = kubeconfigPath
}

// This can go once we have bumped cluster-api to 1.2.0.
// bootstrap.CreateKindBootstrapClusterAndLoadImages will do this for us then.
func (m *EnvManager) pullImages() {
	By("Pulling management cluster images")

	containerRuntime, err := container.NewDockerClient()
	Expect(err).NotTo(HaveOccurred())

	for _, image := range m.Cfg.Images {
		if strings.Contains(image.Name, "microvm:e2e") {
			continue
		}

		By("Pulling image " + image.Name)
		Expect(containerRuntime.PullContainerImageIfNotExists(m.ctx, image.Name)).To(Succeed())
	}
}

func (m *EnvManager) createClusterctlRepo() {
	m.ClusterctlCfg = clusterctl.CreateRepository(m.ctx,
		clusterctl.CreateRepositoryInput{
			E2EConfig:        m.Cfg,
			RepositoryFolder: m.ArtefactDir + "/clusterctl",
		})
}

func (m *EnvManager) initKindCluster() {
	initInput := clusterctl.InitManagementClusterAndWatchControllerLogsInput{
		ClusterProxy:            m.ClusterProxy,
		ClusterctlConfigPath:    m.ClusterctlCfg,
		BootstrapProviders:      providers(m.Cfg, clusterctlv1.BootstrapProviderType),
		ControlPlaneProviders:   providers(m.Cfg, clusterctlv1.ControlPlaneProviderType),
		InfrastructureProviders: m.Cfg.InfrastructureProviders(),
		LogFolder:               m.ArtefactDir + "/logs",
	}
	clusterctl.InitManagementClusterAndWatchControllerLogs(m.ctx, initInput,
		m.Cfg.GetIntervals(m.ClusterProxy.GetName(), "wait-controllers")...)
}

func providers(cfg *clusterctl.E2EConfig, providerType clusterctlv1.ProviderType) []string {
	pList := []string{}

	for _, provider := range cfg.Providers {
		if provider.Type == string(providerType) {
			pList = append(pList, provider.Name)
		}
	}

	return pList
}
