/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/spf13/pflag"
	flintlockv1 "github.com/weaveworks/flintlock/api/services/microvm/v1alpha1"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	cgrecord "k8s.io/client-go/tools/record"
	"k8s.io/klog/klogr"
	"k8s.io/klog/v2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	expclusterv1 "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	//+kubebuilder:scaffold:imports
	infrav1 "github.com/weaveworks/cluster-api-provider-microvm/api/v1alpha1"
	"github.com/weaveworks/cluster-api-provider-microvm/controllers"
	"github.com/weaveworks/cluster-api-provider-microvm/internal/services/microvm"
	mvmgrpc "github.com/weaveworks/cluster-api-provider-microvm/pkg/grpc"
	"github.com/weaveworks/cluster-api-provider-microvm/version"
)

//nolint:gochecknoinits // Maybe we can remove it, now just ignore.
func init() {
	_ = infrav1.AddToScheme(scheme)
	_ = clientgoscheme.AddToScheme(scheme)
	_ = clusterv1.AddToScheme(scheme)
	_ = expclusterv1.AddToScheme(scheme)
	//+kubebuilder:scaffold:scheme

	_ = "comment can't be at the end of the function"
}

//nolint:gochecknoglobals // Maybe we can remove them, now just ignore.
var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")

	enableLeaderElection        bool
	metricsAddr                 string
	leaderElectionNamespace     string
	watchNamespace              string
	profilerAddress             string
	healthAddr                  string
	watchFilterValue            string
	webhookCertDir              string
	microvmClusterConcurrency   int
	microvmMachineConcurrency   int
	webhookPort                 int
	syncPeriod                  time.Duration
	leaderElectionLeaseDuration time.Duration
	leaderElectionRenewDeadline time.Duration
	leaderElectionRetryPeriod   time.Duration
	flintlockProxy              string
)

const (
	defaultLeaderElectionDur   = 15 * time.Second
	defaultLeaderElectRenew    = 10 * time.Second
	defaultLeaderElectionRetry = 2 * time.Second
	defaultSyncPeriod          = 10 * time.Minute
	defaultWebhookPort         = 9443
	defaultEventBurstSize      = 100
)

func initFlags(fs *pflag.FlagSet) {
	fs.StringVar(
		&metricsAddr,
		"metrics-bind-addr",
		"localhost:8080",
		"The address the metric endpoint binds to.",
	)

	fs.BoolVar(
		&enableLeaderElection,
		"leader-elect",
		false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.",
	)

	fs.DurationVar(
		&leaderElectionLeaseDuration,
		"leader-elect-lease-duration",
		defaultLeaderElectionDur,
		"Interval at which non-leader candidates will wait to force acquire leadership (duration string)",
	)

	fs.DurationVar(
		&leaderElectionRenewDeadline,
		"leader-elect-renew-deadline",
		defaultLeaderElectRenew,
		"Duration that the leading controller manager will retry refreshing leadership before giving up (duration string)",
	)

	fs.DurationVar(
		&leaderElectionRetryPeriod,
		"leader-elect-retry-period",
		defaultLeaderElectionRetry,
		"Duration the LeaderElector clients should wait between tries of actions (duration string)",
	)

	fs.StringVar(
		&watchNamespace,
		"namespace",
		"",
		"Namespace that the controller watches to reconcile cluster-api objects. "+
			"If unspecified, the controller watches for cluster-api objects across all namespaces.",
	)

	fs.StringVar(
		&leaderElectionNamespace,
		"leader-election-namespace",
		"",
		"Namespace that the controller performs leader election in. "+
			"If unspecified, the controller will discover which namespace it is running in.",
	)

	fs.StringVar(
		&profilerAddress,
		"profiler-address",
		"",
		"Bind address to expose the pprof profiler (e.g. localhost:6060)",
	)

	fs.StringVar(
		&watchFilterValue,
		"watch-filter",
		"",
		fmt.Sprintf(
			"Label value that the controller watches to reconcile cluster-api objects. Label key is always %s. "+
				"If unspecified, the controller watches for all cluster-api objects.",
			clusterv1.WatchLabel,
		),
	)

	fs.IntVar(&microvmClusterConcurrency,
		"microvmcluster-concurrency",
		1,
		"Number of MicrovmClusters to process simultaneously",
	)

	fs.IntVar(&microvmMachineConcurrency,
		"microvmmachine-concurrency",
		1,
		"Number of MicrovmMachines to process simultaneously",
	)

	fs.DurationVar(&syncPeriod,
		"sync-period",
		defaultSyncPeriod,
		"The minimum interval at which watched resources are reconciled (e.g. 15m)",
	)

	fs.IntVar(&webhookPort,
		"webhook-port",
		defaultWebhookPort,
		"Webhook Server port",
	)

	fs.StringVar(&webhookCertDir,
		"webhook-cert-dir",
		"/tmp/k8s-webhook-server/serving-certs",
		"Webhook Server Certificate Directory, is the directory that contains the server key and certificate",
	)

	fs.StringVar(&healthAddr,
		"health-addr",
		":9440",
		"The address the health endpoint binds to.",
	)

	fs.StringVar(&flintlockProxy,
		"flintlock-proxy",
		"",
		"The URL of a proxy to use when calling flintlock.",
	)
}

func main() {
	klog.InitFlags(nil)

	initFlags(pflag.CommandLine)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	if watchNamespace != "" {
		setupLog.Info("Watching cluster-api objects only in namespace for reconciliation", "namespace", watchNamespace)
	}

	if flintlockProxy != "" {
		setupLog.Info("Using flintlock proxy", "address", flintlockProxy)
		if _, parseErr := url.Parse(flintlockProxy); parseErr != nil {
			setupLog.Error(parseErr, "unable to parse flintlock proxy url")
			os.Exit(1)
		}
	}

	if profilerAddress != "" {
		setupLog.Info("Profiler listening for requests", "profiler-address", profilerAddress)

		go func() {
			setupLog.Error(http.ListenAndServe(profilerAddress, nil), "listen and serve error")
		}()
	}

	ctrl.SetLogger(klogr.New())

	// Machine and cluster operations can create enough events to trigger the event recorder spam filter
	// Setting the burst size higher ensures all events will be recorded and submitted to the API
	broadcaster := cgrecord.NewBroadcasterWithCorrelatorOptions(cgrecord.CorrelatorOptions{
		BurstSize: defaultEventBurstSize,
	})

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                     scheme,
		MetricsBindAddress:         metricsAddr,
		LeaderElection:             enableLeaderElection,
		LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
		LeaderElectionID:           "controller-leader-elect-capmvm",
		LeaderElectionNamespace:    leaderElectionNamespace,
		SyncPeriod:                 &syncPeriod,
		Namespace:                  watchNamespace,
		EventBroadcaster:           broadcaster,
		Port:                       webhookPort,
		CertDir:                    webhookCertDir,
		HealthProbeBindAddress:     healthAddr,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Initialise event recorder.
	record.InitFromRecorder(mgr.GetEventRecorderFor("microvm-controller"))

	// Setup the context that's going to be used in controllers and for the manager.
	ctx := ctrl.SetupSignalHandler()

	if err := setupReconcilers(ctx, mgr); err != nil {
		setupLog.Error(err, "failed to add Microvm Reconcilers")
		os.Exit(1)
	}

	if err := setupWebhooks(mgr); err != nil {
		setupLog.Error(err, "failed to add Microvm Webhooks")
		os.Exit(1)
	}

	if err := addHealthChecks(mgr); err != nil {
		setupLog.Error(err, "failed to add health checks")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	setupLog.Info("starting manager", "version", version.Get().String())

	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func setupReconcilers(ctx context.Context, mgr ctrl.Manager) error {
	managerOptions := controller.Options{
		MaxConcurrentReconciles: microvmClusterConcurrency,
		RecoverPanic:            true,
	}

	if err := (&controllers.MicrovmClusterReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		Recorder:         mgr.GetEventRecorderFor("microvmcluster-controller"),
		WatchFilterValue: watchFilterValue,
	}).SetupWithManager(ctx, mgr, managerOptions); err != nil {
		return fmt.Errorf("unable to create microvm cluster controller: %w", err)
	}

	if err := (&controllers.MicrovmMachineReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		Recorder:         mgr.GetEventRecorderFor("microvmmachine-controller"),
		WatchFilterValue: watchFilterValue,
		MvmClientFunc:    createFlintlockClient,
	}).SetupWithManager(ctx, mgr, managerOptions); err != nil {
		return fmt.Errorf("unable to create microvm machine controller: %w", err)
	}

	return nil
}

func setupWebhooks(mgr ctrl.Manager) error {
	if err := (&infrav1.MicrovmCluster{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to setup MicrovmCluster webhook:%w", err)
	}

	if err := (&infrav1.MicrovmMachine{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to setup MicrovmMachine webhook:%w", err)
	}

	if err := (&infrav1.MicrovmMachineTemplate{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to setup MicrovmMachineTemplate webhook:%w", err)
	}

	return nil
}

func addHealthChecks(mgr ctrl.Manager) error {
	if err := mgr.AddReadyzCheck("webhook", mgr.GetWebhookServer().StartedChecker()); err != nil {
		return fmt.Errorf("unable to create ready check: %w", err)
	}

	if err := mgr.AddHealthzCheck("webhook", mgr.GetWebhookServer().StartedChecker()); err != nil {
		return fmt.Errorf("unable to create healthz check: %w", err)
	}

	return nil
}

func createFlintlockClient(address string) (microvm.Client, error) {
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
	}

	if flintlockProxy != "" {
		url, err := url.Parse(flintlockProxy)
		if err != nil {
			return nil, fmt.Errorf("parsing proxy url %s: %w", flintlockProxy, err)
		}
		opts = append(opts, mvmgrpc.WithProxy(url))
	}

	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating grpc connection: %w", err)
	}

	flClient := flintlockv1.NewMicroVMClient(conn)

	return flClient, nil
}
