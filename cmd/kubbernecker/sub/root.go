package sub

import (
	"errors"
	"flag"
	"os"

	"github.com/zoetrope/kubbernecker"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const defaultConfigPath = "/etc/kubbernecker/kubbernecker-config.yaml"

type options struct {
	configFile       string
	metricsAddr      string
	probeAddr        string
	leaderElectionID string
	zapOpts          zap.Options

	podNamespace string
	logger       logr.Logger
}

// NewCmd creates the root *cobra.Command of `kubectl-rubberneck`.
func NewCmd() *cobra.Command {

	opts := &options{}
	cmd := &cobra.Command{
		Use:     "kubbernecker",
		Short:   "kubbernecker",
		Long:    `kubbernecker`,
		Version: kubbernecker.Version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cmd.SilenceUsage = true
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Fill(cmd, args); err != nil {
				return err
			}
			if err := opts.Run(cmd, args); err != nil {
				return err
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&opts.configFile, "config-file", defaultConfigPath, "Configuration file path")
	fs.StringVar(&opts.metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to")
	fs.StringVar(&opts.probeAddr, "health-probe-addr", ":8081", "Listen address for health probes")
	fs.StringVar(&opts.leaderElectionID, "leader-election-id", "kubbernecker", "ID for leader election by controller-runtime")

	goflags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(goflags)
	opts.zapOpts.BindFlags(goflags)
	fs.AddGoFlagSet(goflags)

	return cmd
}

func (o *options) Fill(cmd *cobra.Command, args []string) error {

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&o.zapOpts)))
	o.logger = ctrl.Log.WithName("setup")

	o.logger.Info("Fill")

	ns := os.Getenv("POD_NAMESPACE")
	if ns == "" {
		return errors.New("no environment variable POD_NAMESPACE")
	}
	o.podNamespace = ns

	return nil
}

// rootCmd represents the base command when called without any subcommands

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	defer klog.Flush()
	cmd := NewCmd()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
