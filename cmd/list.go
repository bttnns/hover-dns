package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/bttnns/hover-dns/internal/hover"
)

var listCmd = &cobra.Command{
	Use:   "list [domain]",
	Short: "List DNS records for all domains, or a specific one",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClientFromConfig()
		if err != nil {
			return err
		}

		filter := ""
		if len(args) > 0 {
			filter = args[0]
		}

		domains, err := c.Domains()
		if err != nil {
			return fmt.Errorf("fetching domains: %w", err)
		}

		found := false
		for _, d := range domains {
			if filter != "" && d.DomainName != filter && d.ID != filter {
				continue
			}
			found = true
			records, err := c.DNS(d.ID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: DNS fetch for %s: %v\n", d.DomainName, err)
				continue
			}
			printDNSTable(d, records)
		}

		if filter != "" && !found {
			return fmt.Errorf("domain %q not found", filter)
		}
		return nil
	},
}

func printDNSTable(domain hover.HoverDomain, records []hover.DNSRecord) {
	fmt.Printf("=== %s (%s) ===\n", domain.DomainName, domain.ID)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "RECORD ID\tNAME\tTYPE\tTTL\tVALUE")
	fmt.Fprintln(w, "---------\t----\t----\t---\t-----")
	for _, r := range records {
		name := r.Name
		if name == "@" {
			name = domain.DomainName
		} else {
			name += "." + domain.DomainName
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n", r.ID, name, r.Type, r.TTL, r.Value)
	}
	w.Flush()
	fmt.Println()
}

func init() {
	rootCmd.AddCommand(listCmd)
}
