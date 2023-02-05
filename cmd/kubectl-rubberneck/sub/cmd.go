package sub

import (
	"flag"
	"os"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/zoetrope/kubbernecker/pkg/cobwrap"
	"go.uber.org/zap/zapcore"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type rootOpts struct {
	loglevel int
	config   *genericclioptions.ConfigFlags
	streams  genericclioptions.IOStreams
	logger   logr.Logger
}

// NewCmd creates the root *cobra.Command of `kubectl-rubberneck`.
func NewCmd(streams genericclioptions.IOStreams) *cobwrap.Command[*rootOpts] {
	cmd := &cobwrap.Command[*rootOpts]{
		Command: &cobra.Command{
			Use:   "kubectl-rubberneck",
			Short: "A brief description of your application",
			Long:  `kubbernecker`,
			PersistentPreRun: func(cmd *cobra.Command, args []string) {
				cmd.SilenceUsage = true
			},
		},
		Options: &rootOpts{
			streams: streams,
		},
	}

	cmd.Command.PersistentFlags().IntVarP(&cmd.Options.loglevel, "log-level", "v", -1, "number for the log level verbosity")
	config := genericclioptions.NewConfigFlags(true)
	config.AddFlags(cmd.Command.PersistentFlags())
	cmd.Options.config = config

	cobwrap.AddCommand(cmd, newWatchCmd())
	cobwrap.AddCommand(cmd, newBlameCmd())

	return cmd
}

func (o *rootOpts) Fill(cmd *cobra.Command, args []string) error {
	klog.InitFlags(nil)
	flag.Set("v", strconv.Itoa(o.loglevel))

	//logruslog := logrus.New()
	//logruslog.SetFormatter(&logrus.TextFormatter{})
	//logruslog.SetLevel(logrus.Level(4 + o.loglevel))
	//logrusLogger := logrusr.New(logruslog)

	//o.logger = &logrusLogger
	//ctrl.SetLogger(logrusLogger)
	//klog.SetLogger(logrusLogger.WithName("client-go"))

	// zap
	zapopts := zap.Options{
		Development: true,
		Level:       zapcore.Level(-1 * o.loglevel),
	}
	//zapopts.BindFlags(flag.CommandLine)
	zapLogger := zap.New(zap.UseFlagOptions(&zapopts))
	o.logger = zapLogger
	ctrl.SetLogger(zapLogger)
	klog.SetLogger(zapLogger.WithName("client-go"))
	return nil
}

func (o *rootOpts) Run(cmd *cobra.Command, args []string) error {
	return nil
}

// rootCmd represents the base command when called without any subcommands

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	defer klog.Flush()
	cmd := NewCmd(genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	})
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
