package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/bttnns/hover-dns/internal/hover"
)

var addCmd = &cobra.Command{
	Use:   "add <domain> <name> <type> <value>",
	Short: "Add a new DNS record",
	Example: `  hover-dns add example.com @ A 1.2.3.4
  hover-dns add example.com www CNAME example.com
  hover-dns add example.com @ TXT "v=spf1 ~all"`,
	Args: cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateRecordType(args[2]); err != nil {
			return err
		}
		c, err := newClientFromConfig()
		if err != nil {
			return err
		}
		domain, err := c.FindDomain(args[0])
		if err != nil {
			return err
		}
		if err := c.Add(domain.ID, hover.NormalizeName(args[1], args[0]), args[2], args[3]); err != nil {
			return err
		}
		fmt.Printf("added %s %s %s -> %s\n", args[0], args[1], args[2], args[3])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
