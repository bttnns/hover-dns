// Package ddns runs the dynamic-DNS loop: keep configured records pointed at the
// host's current external IP. It is driven by serve (alongside the API) and by
// the standalone `ddns` CLI command.
package ddns

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bttnns/hover-dns/internal/hover"
	"github.com/bttnns/hover-dns/internal/util"
)

const defaultInterval = 46800

// Run blocks until ctx is cancelled. It returns a non-nil error only for a fatal
// misconfiguration; transient failures are logged and retried.
func Run(ctx context.Context, c *hover.Client, cfg *hover.Config) error {
	if cfg.Domain == "" {
		return fmt.Errorf("domain required: set in config")
	}
	if len(cfg.RecordNames) == 0 {
		return fmt.Errorf("record_names required: set in config")
	}

	interval := cfg.Interval
	if interval <= 0 {
		interval = defaultInterval
	}

	log.Printf("ddns starting: domain=%s names=%v interval=%ds", cfg.Domain, cfg.RecordNames, interval)

	// current holds live state: name -> DNSRecord (with current ID and value)
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
		if !sleep(ctx, 30*time.Second) {
			return nil
		}
	}

	for {
		ip, err := util.ExternalIP()
		if err != nil {
			log.Printf("error: get external IP: %v", err)
			if !sleep(ctx, time.Duration(interval)*time.Second) {
				return nil
			}
			continue
		}

		updated := 0
		for _, name := range cfg.RecordNames {
			rec, ok := current[name]
			if !ok {
				continue
			}
			if rec.Value == ip {
				continue
			}
			log.Printf("updating %s.%s (%s): %s -> %s", name, cfg.Domain, rec.ID, rec.Value, ip)
			_, newRec, err := c.SetRecord(cfg.Domain, name, ip, "A")
			if err != nil {
				log.Printf("error: update %s: %v", name, err)
				continue
			}
			// Set is delete+create, so the id changed; keep current in sync from
			// the returned record instead of a separate full re-fetch.
			current[name] = *newRec
			updated++
			log.Printf("updated %s.%s -> %s", name, cfg.Domain, ip)
		}
		if updated == 0 {
			log.Printf("ip=%s all records current, sleeping %ds", ip, interval)
		}

		if !sleep(ctx, time.Duration(interval)*time.Second) {
			return nil
		}
	}
}

// sleep waits for d or until ctx is cancelled. It returns false if cancelled.
func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
