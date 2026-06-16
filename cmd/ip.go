package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/bttnns/hover-dns/internal/util"
)

var ipCmd = &cobra.Command{
	Use:   "ip",
	Short: "Show current external IP address (no auth required)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ip, err := util.ExternalIP()
		if err != nil {
			return err
		}
		fmt.Println(ip)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(ipCmd)
}
