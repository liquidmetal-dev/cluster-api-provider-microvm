//go:build e2e
// +build e2e

package e2e_test

import (
	"testing"

	"github.com/liquidmetal-dev/cluster-api-provider-microvm/test/e2e/utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	mngr *utils.EnvManager
)

func init() {
	p := utils.NewParams()
	mngr = utils.NewEnvManager(p)
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)

	BeforeSuite(func() {
		// TODO: Use SynchronizedBeforeSuite to be able to use parallel nodes to run tests
		mngr.Setup()
	})

	AfterSuite(func() {
		mngr.Teardown()
	})

	RunSpecs(t, "Liquid Metal E2E Suite")
}
