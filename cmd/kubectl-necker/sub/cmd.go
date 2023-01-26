package sub

import (
	"flag"
	"os"
	"strconv"

	"github.com/bombsimon/logrusr/v4"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

type rootOpts struct {
	loglevel int
	config   *genericclioptions.ConfigFlags
	streams  genericclioptions.IOStreams
	logger   *logr.Logger
}

// NewCmd creates the root *cobra.Command of `kubectl-necker`.
func NewCmd(streams genericclioptions.IOStreams) *cobra.Command {
	opts := &rootOpts{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "kubbernecker",
		Short: "A brief description of your application",
		Long:  `kubbernecker`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return opts.Fill(cmd, args)
		},
	}

	cmd.PersistentFlags().IntVarP(&opts.loglevel, "log-level", "v", -1, "number for the log level verbosity")
	config := genericclioptions.NewConfigFlags(true)
	config.AddFlags(cmd.PersistentFlags())
	opts.config = config

	cmd.AddCommand(newWatchCmd(opts))

	return cmd
}

func (o *rootOpts) Fill(cmd *cobra.Command, args []string) error {
	klog.InitFlags(nil)
	flag.Set("v", strconv.Itoa(o.loglevel))

	logruslog := logrus.New()
	logruslog.SetFormatter(&logrus.TextFormatter{})
	logruslog.SetLevel(logrus.Level(4 + o.loglevel))
	logrusLogger := logrusr.New(logruslog)

	o.logger = &logrusLogger
	ctrl.SetLogger(logrusLogger)
	klog.SetLogger(logrusLogger.WithName("client-go"))

	//klog.V(0).Info("klog info 0")
	//klog.V(3).Info("klog info 3")
	//logrusLogger.V(0).Info("logrus info 0", "level", o.loglevel)
	//logrusLogger.V(3).Info("logrus info 3", "level", o.loglevel)

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
