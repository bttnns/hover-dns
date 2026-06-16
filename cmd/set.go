package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/bttnns/hover-dns/internal/hover"
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
		if setType != "" {
			if err := validateRecordType(setType); err != nil {
				return err
			}
		}
		c, err := newClientFromConfig()
		if err != nil {
			return err
		}
		records, err := c.DomainRecords(args[0])
		if err != nil {
			return err
		}
		rec := hover.FindByName(records, hover.NormalizeName(args[1], args[0]))
		if rec == nil {
			return fmt.Errorf("record %q not found in %s", args[1], args[0])
		}
		if err := c.Set(rec.ID, args[2], setType); err != nil {
			return err
		}
		fmt.Printf("updated %s.%s -> %s\n", rec.Name, args[0], args[2])
		return nil
	},
}

func init() {
	setCmd.Flags().StringVar(&setType, "type", "", "force record type (e.g. A, CNAME, TXT)")
	rootCmd.AddCommand(setCmd)
}
