package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/spf13/cobra"
	"github.com/bttnns/hover-dns/internal/hover"
	"github.com/bttnns/hover-dns/internal/util"
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

		if cfg.Domain == "" {
			return fmt.Errorf("domain required: set in config")
		}
		if len(cfg.RecordNames) == 0 {
			return fmt.Errorf("record_names required: set in config")
		}

		interval := cfg.Interval
		if interval <= 0 {
			interval = 46800
		}

		log.Printf("ddns starting: domain=%s names=%v interval=%ds", cfg.Domain, cfg.RecordNames, interval)

		// current holds live state: name → DNSRecord (with current ID and value)
		current := make(map[string]hover.DNSRecord)

		loadRecords := func() bool {
			records, err := c.DomainRecords(cfg.Domain)
			if err != nil {
				log.Printf("error: fetch DNS records: %v", err)
				return false
			}
			for _, name := range cfg.RecordNames {
				rec := hover.FindByName(records, hover.NormalizeName(name, cfg.Domain))
				if rec == nil {
					log.Printf("warning: record %q not found in %s", name, cfg.Domain)
					continue
				}
				current[name] = *rec
				log.Printf("loaded: %s.%s (%s) = %s", name, cfg.Domain, rec.ID, rec.Value)
			}
			return true
		}

		// fetch initial state, retry every 30s until success
		for !loadRecords() {
			time.Sleep(30 * time.Second)
		}

		for {
			ip, err := util.ExternalIP()
			if err != nil {
				log.Printf("error: get external IP: %v", err)
				time.Sleep(time.Duration(interval) * time.Second)
				continue
			}

			anyUpdated := false
			for _, name := range cfg.RecordNames {
				rec, ok := current[name]
				if !ok {
					continue
				}
				if rec.Value == ip {
					continue
				}
				log.Printf("updating %s.%s (%s): %s -> %s", name, cfg.Domain, rec.ID, rec.Value, ip)
				if err := c.Set(rec.ID, ip, "A"); err != nil {
					log.Printf("error: update %s: %v", name, err)
					continue
				}
				log.Printf("updated %s.%s -> %s", name, cfg.Domain, ip)
				anyUpdated = true
			}
			if anyUpdated {
				// re-fetch to pick up new IDs (Set = delete+create, ID changes)
				loadRecords()
			} else {
				log.Printf("ip=%s all records current, sleeping %ds", ip, interval)
			}

			time.Sleep(time.Duration(interval) * time.Second)
		}
	},
}

func init() {
	rootCmd.AddCommand(ddnsCmd)
}
