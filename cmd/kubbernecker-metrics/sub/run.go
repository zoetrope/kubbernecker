package sub

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/zoetrope/kubbernecker/internal/controller"
	"github.com/zoetrope/kubbernecker/pkg/client"
	"github.com/zoetrope/kubbernecker/pkg/config"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

func (o *options) Run(cmd *cobra.Command, args []string) error {
	fmt.Println("run")

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return fmt.Errorf("unable to add client-go objects: %w", err)
	}

	cfgData, err := os.ReadFile(o.configFile)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", o.configFile, err)
	}
	cfg := &config.Config{}
	if err := cfg.Load(cfgData); err != nil {
		return fmt.Errorf("unable to load the configuration file: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configurations: %w", err)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                  scheme,
		NewClient:               client.NewCachingClient,
		MetricsBindAddress:      o.metricsAddr,
		HealthProbeBindAddress:  o.probeAddr,
		LeaderElection:          true,
		LeaderElectionID:        o.leaderElectionID,
		LeaderElectionNamespace: o.podNamespace,
		SyncPeriod:              pointer.Duration(0 * time.Second),
	})
	if err != nil {
		return fmt.Errorf("unable to start manager: %w", err)
	}

	kubeClient, err := client.MakeKubeClientFromCluster(mgr)
	if err != nil {
		return fmt.Errorf("failed to make KubeClient: %w", err)
	}
	wm := controller.NewWatcherManager(mgr.GetLogger(), kubeClient, cfg)
	if err = mgr.Add(wm); err != nil {
		return fmt.Errorf("failed to add WatcherManager: %w", err)
	}
	if err = controller.SetupMetrics(wm); err != nil {
		return fmt.Errorf("failed to setup metrics: %w", err)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up health check: %w", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up ready check: %w", err)
	}

	o.logger.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("problem running manager: %s", err)
	}
	return nil
}
