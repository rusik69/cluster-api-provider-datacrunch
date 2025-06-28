/*
Copyright 2024.

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
	"os"
	"time"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	infrav1beta1 "github.com/rusik69/cluster-api-provider-datacrunch/api/v1beta1"
	controllers "github.com/rusik69/cluster-api-provider-datacrunch/internal/controller"
	"github.com/rusik69/cluster-api-provider-datacrunch/version"
)

var (
	myscheme    = runtime.NewScheme()
	setupLog    = ctrl.Log.WithName("setup")
	showVersion = flag.Bool("version", false, "Show version and exit")
)

func init() {
	utilruntime.Must(scheme.AddToScheme(myscheme))
	utilruntime.Must(clusterv1.AddToScheme(myscheme))
	utilruntime.Must(infrav1beta1.AddToScheme(myscheme))
}

func main() {
	var (
		metricsAddr                  string
		enableLeaderElection         bool
		leaderElectionLeaseDuration  time.Duration
		leaderElectionRenewDeadline  time.Duration
		leaderElectionRetryPeriod    time.Duration
		watchFilterValue             string
		profilerAddress              string
		dataCrunchClusterConcurrency int
		dataCrunchMachineConcurrency int
		syncPeriod                   time.Duration
		healthAddr                   string
		webhookPort                  int
		webhookCertDir               string
		logLevel                     string
	)

	flag.StringVar(&metricsAddr, "metrics-bind-addr", ":8080",
		"The address the metric endpoint binds to.")

	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")

	flag.DurationVar(&leaderElectionLeaseDuration, "leader-elect-lease-duration", 15*time.Second,
		"Interval at which non-leader candidates will wait to force acquire leadership (duration string)")

	flag.DurationVar(&leaderElectionRenewDeadline, "leader-elect-renew-deadline", 10*time.Second,
		"Duration that the leading controller manager will retry refreshing leadership before giving up (duration string)")

	flag.DurationVar(&leaderElectionRetryPeriod, "leader-elect-retry-period", 2*time.Second,
		"Duration the LeaderElector clients should wait between tries of actions (duration string)")

	flag.StringVar(&watchFilterValue, "watch-filter", "",
		fmt.Sprintf("Label value that the controller watches to reconcile cluster-api objects. Label key is always %s. If unspecified, the controller watches for all cluster-api objects.", clusterv1.WatchLabel))

	flag.StringVar(&profilerAddress, "profiler-address", "",
		"Bind address to expose the pprof profiler (e.g. localhost:6060)")

	flag.IntVar(&dataCrunchClusterConcurrency, "datacrunchcluster-concurrency", 10,
		"Number of DataCrunchClusters to process simultaneously")

	flag.IntVar(&dataCrunchMachineConcurrency, "datacrunchmachine-concurrency", 10,
		"Number of DataCrunchMachines to process simultaneously")

	flag.DurationVar(&syncPeriod, "sync-period", 10*time.Minute,
		"The minimum interval at which watched resources are reconciled (e.g. 15m)")

	flag.StringVar(&healthAddr, "health-addr", ":9440",
		"The address the health endpoint binds to.")

	flag.IntVar(&webhookPort, "webhook-port", 9443,
		"Webhook Server port")

	flag.StringVar(&webhookCertDir, "webhook-cert-dir", "/tmp/k8s-webhook-server/serving-certs/",
		"Webhook cert dir, only used when webhook-port is specified.")

	flag.StringVar(&logLevel, "log-level", "info",
		"Log level for the controller (debug, info, warn, error)")

	// Add flags registered by imported packages (e.g. klog-v2, controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	if *showVersion {
		fmt.Println(version.Get().String())
		os.Exit(0)
	}

	ctrl.SetLogger(klogr.New())

	ctx := ctrl.SetupSignalHandler()

	setupLog.Info("Version", "version", version.Get().String())

	restConfig := ctrl.GetConfigOrDie()
	restConfig.UserAgent = "cluster-api-provider-datacrunch-manager"
	mgr, err := ctrl.NewManager(restConfig, ctrl.Options{
		Scheme:                     myscheme,
		LeaderElection:             enableLeaderElection,
		LeaderElectionID:           "controller-leader-election-capd",
		LeaderElectionResourceLock: "leases",
		LeaseDuration:              &leaderElectionLeaseDuration,
		RenewDeadline:              &leaderElectionRenewDeadline,
		RetryPeriod:                &leaderElectionRetryPeriod,
		HealthProbeBindAddress:     healthAddr,
		Logger:                     log.FromContext(ctx),
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		WebhookServer: webhook.NewServer(webhook.Options{
			Port:    webhookPort,
			CertDir: webhookCertDir,
		}),
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Setup the context that's going to be used in controllers and for the manager.
	ctx = ctrl.LoggerInto(ctx, ctrl.Log)

	setupReconcilers(ctx, mgr, controller.Options{
		MaxConcurrentReconciles: dataCrunchClusterConcurrency,
	}, controller.Options{
		MaxConcurrentReconciles: dataCrunchMachineConcurrency,
	}, watchFilterValue)

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func setupReconcilers(ctx context.Context, mgr ctrl.Manager, dataCrunchClusterOptions, dataCrunchMachineOptions controller.Options, watchFilterValue string) {
	if err := (&controllers.DataCrunchClusterReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		Recorder:         mgr.GetEventRecorderFor("datacrunchcluster-controller"),
		Log:              ctrl.Log.WithName("controllers").WithName("DataCrunchCluster"),
		WatchFilterValue: watchFilterValue,
	}).SetupWithManager(ctx, mgr, dataCrunchClusterOptions); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DataCrunchCluster")
		os.Exit(1)
	}

	if err := (&controllers.DataCrunchMachineReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		Recorder:         mgr.GetEventRecorderFor("datacrunchmachine-controller"),
		Log:              ctrl.Log.WithName("controllers").WithName("DataCrunchMachine"),
		WatchFilterValue: watchFilterValue,
	}).SetupWithManager(ctx, mgr, dataCrunchMachineOptions); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DataCrunchMachine")
		os.Exit(1)
	}
}
