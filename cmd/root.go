package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/bttnns/hover-dns/internal/hover"
)

var cfgFile string
var verbose bool

var validRecordTypes = map[string]bool{
	"A": true, "AAAA": true, "CNAME": true,
	"MX": true, "TXT": true, "SRV": true,
}

func validateRecordType(t string) error {
	if !validRecordTypes[strings.ToUpper(t)] {
		return fmt.Errorf("invalid record type %q: must be one of A, AAAA, CNAME, MX, TXT, SRV", t)
	}
	return nil
}

var rootCmd = &cobra.Command{
	Use:   "hover-dns",
	Short: "Hover DNS management CLI and DDNS daemon",
	Long:  "Manage Hover DNS records from the command line, or run as a DDNS daemon to keep records pointed at your current external IP.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newClientFromConfig() (*hover.Client, error) {
	cfg, err := hover.LoadConfig(cfgFile)
	if err != nil {
		return nil, err
	}
	return hover.NewClient(cfg, verbose)
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "config.json", "path to config file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose request/response logging")
}
