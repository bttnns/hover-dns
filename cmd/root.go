package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bttnns/hover-dns/internal/hover"
	"github.com/spf13/cobra"
)

var cfgFile string
var verbose bool

var rootCmd = &cobra.Command{
	Use:   "hover-dns",
	Short: "Hover DNS management CLI and daemon",
	Long:  "Manage Hover DNS records from the command line, or run `serve` as a daemon: a DDNS loop that keeps records pointed at your current external IP and/or an authenticated HTTP API, each toggled in the config.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// notifyContext returns a context cancelled on SIGINT/SIGTERM, shared by the
// long-running ddns and serve commands.
func notifyContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
}

func newClientFromConfig() (*hover.Client, error) {
	cfg, err := hover.LoadConfig(cfgFile)
	if err != nil {
		return nil, err
	}
	return hover.NewClient(cfg, verbose)
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "config.yaml", "path to config file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose request/response logging")
}
