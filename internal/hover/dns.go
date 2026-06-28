package hover

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

const setRetryDelay = 5 * time.Second

func (c *Client) Domains() ([]HoverDomain, error) {
	body, status, err := c.doRequest("GET", c.baseURL+"/api/domains", nil)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, statusError("domains API", status, "")
	}
	var result domainsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing domains response: %w", err)
	}
	if !result.Succeeded {
		return nil, fmt.Errorf("domains API returned failure")
	}
	return result.Domains, nil
}

func (c *Client) DNS(domainID string) ([]DNSRecord, error) {
	body, status, err := c.doRequest("GET", fmt.Sprintf("%s/api/domains/%s/dns", c.baseURL, domainID), nil)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, statusError(fmt.Sprintf("DNS API for %s", domainID), status, "")
	}
	var result dnsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing DNS response for %s: %w", domainID, err)
	}
	if !result.Succeeded {
		return nil, fmt.Errorf("DNS API returned failure for %s", domainID)
	}
	if len(result.Domains) == 0 {
		return nil, nil
	}
	return result.Domains[0].Entries, nil
}

// DomainRecords finds the domain by name or ID and returns its DNS records.
func (c *Client) DomainRecords(domainName string) ([]DNSRecord, error) {
	domain, err := c.FindDomain(domainName)
	if err != nil {
		return nil, err
	}
	return c.DNS(domain.ID)
}

func (c *Client) FindRecord(recordID string) (*DNSRecord, string, error) {
	domains, err := c.Domains()
	if err != nil {
		return nil, "", err
	}
	for _, d := range domains {
		records, err := c.DNS(d.ID)
		if err != nil {
			log.Printf("warning: skipping domain %s: %v", d.DomainName, err)
			continue
		}
		for i := range records {
			if records[i].ID == recordID {
				return &records[i], d.ID, nil
			}
		}
	}
	return nil, "", fmt.Errorf("record %s not found", recordID)
}

func (c *Client) FindDomain(name string) (*HoverDomain, error) {
	domains, err := c.Domains()
	if err != nil {
		return nil, err
	}
	for _, d := range domains {
		if d.DomainName == name || d.ID == name {
			return &d, nil
		}
	}
	return nil, fmt.Errorf("domain %q not found: %w", name, ErrNotFound)
}

func (c *Client) Add(domainID, name, recType, value string) error {
	payload, err := json.Marshal(map[string]string{
		"name":    name,
		"type":    recType,
		"content": value,
	})
	if err != nil {
		return fmt.Errorf("marshaling record: %w", err)
	}
	body, status, reqErr := c.doRequest("POST", fmt.Sprintf("%s/api/domains/%s/dns", c.baseURL, domainID), payload)
	if reqErr != nil {
		return fmt.Errorf("create failed: %w", reqErr)
	}
	if status < 200 || status >= 300 {
		var result struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			log.Printf("warning: could not parse error response: %v", err)
		}
		msg := result.Error
		if msg == "" {
			msg = string(body)
		}
		return statusError("create failed", status, msg)
	}
	return nil
}

func (c *Client) Delete(recordID string) error {
	_, status, err := c.doRequest("DELETE", fmt.Sprintf("%s/api/dns/%s", c.baseURL, recordID), nil)
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return statusError("delete failed", status, "")
	}
	return nil
}

func (c *Client) Set(recordID, value, forceType string) error {
	// FindRecord is a read that scans every domain's records (1+N HTTP calls).
	// Keep it out of the write lock so a concurrent writer isn't blocked behind
	// these round-trips.
	rec, domainID, err := c.FindRecord(recordID)
	if err != nil {
		return err
	}

	recType := rec.Type
	if forceType != "" {
		recType = forceType
	}
	if recType != rec.Type {
		log.Printf("converting %s from %s to %s", recordID, rec.Type, recType)
	}

	c.mu.Lock()
	if err := c.Delete(recordID); err != nil {
		c.mu.Unlock()
		return err
	}
	addErr := c.Add(domainID, rec.Name, recType, value)
	c.mu.Unlock()
	if addErr == nil {
		return nil
	}

	// The first create failed. Back off without holding the lock (so other
	// writers can proceed), then retry under a fresh lock.
	log.Printf("warning: add failed, retrying in %s: %v", setRetryDelay, addErr)
	time.Sleep(setRetryDelay)

	c.mu.Lock()
	defer c.mu.Unlock()
	if retryErr := c.Add(domainID, rec.Name, recType, value); retryErr != nil {
		if restoreErr := c.Add(domainID, rec.Name, rec.Type, rec.Value); restoreErr != nil {
			return fmt.Errorf("update failed: %w; restore also failed: %v", retryErr, restoreErr)
		}
		return fmt.Errorf("update failed (original restored): %w", retryErr)
	}
	return nil
}
