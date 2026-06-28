package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <domain> <name> <type> <value>",
	Short: "Add a new DNS record",
	Example: `  hover-dns add example.com @ A 1.2.3.4
  hover-dns add example.com www CNAME example.com
  hover-dns add example.com @ TXT "v=spf1 ~all"`,
	Args: cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClientFromConfig()
		if err != nil {
			return err
		}
		domain, rec, err := c.AddRecord(args[0], args[1], args[2], args[3])
		if err != nil {
			return err
		}
		fmt.Printf("added %s %s %s -> %s (id %s)\n", domain.DomainName, args[1], rec.Type, rec.Value, rec.ID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
