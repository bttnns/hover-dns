package cmd

import (
	"github.com/bttnns/hover-dns/internal/ddns"
	"github.com/bttnns/hover-dns/internal/hover"
	"github.com/spf13/cobra"
)

var ddnsCmd = &cobra.Command{
	Use:   "ddns",
	Short: "Run DDNS daemon, updating records to current external IP",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := hover.LoadConfig(cfgFile)
		if err != nil {
			return err
		}
		c, err := hover.NewClient(cfg, verbose)
		if err != nil {
			return err
		}

		ctx, stop := notifyContext()
		defer stop()

		return ddns.Run(ctx, c, cfg)
	},
}

func init() {
	rootCmd.AddCommand(ddnsCmd)
}
