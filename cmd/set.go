package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var setType string

var setCmd = &cobra.Command{
	Use:   "set <domain> <name> <value>",
	Short: "Update a DNS record's value",
	Example: `  hover-dns set example.com @ 1.2.3.4
  hover-dns set example.com home 1.2.3.4
  hover-dns set --type TXT example.com @ "v=spf1 ~all"`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClientFromConfig()
		if err != nil {
			return err
		}
		domain, rec, err := c.SetRecord(args[0], args[1], args[2], setType)
		if err != nil {
			return err
		}
		fmt.Printf("updated %s %s %s -> %s (id %s)\n", domain.DomainName, args[1], rec.Type, rec.Value, rec.ID)
		return nil
	},
}

func init() {
	setCmd.Flags().StringVar(&setType, "type", "", "force record type (e.g. A, CNAME, TXT)")
	rootCmd.AddCommand(setCmd)
}
