package hover

import (
	"fmt"
	"strings"
)

// ops.go is the high-level, name-friendly operations layer shared by the CLI
// commands and the HTTP API. It owns resolution (domain ref -> id), name
// normalization, and validation so neither front-end carries business logic.
// The low-level transport (Domains/DNS/Add/Delete in dns.go) stays id-keyed.

// ListDomains returns every domain on the account.
func (c *Client) ListDomains() ([]HoverDomain, error) {
	return c.Domains()
}

// ListRecords returns the DNS records for a domain given its name or id.
func (c *Client) ListRecords(domainRef string) (*HoverDomain, []DNSRecord, error) {
	domain, err := c.FindDomain(domainRef)
	if err != nil {
		return nil, nil, err
	}
	records, err := c.DNS(domain.ID)
	if err != nil {
		return nil, nil, err
	}
	return domain, records, nil
}

// AddRecord validates, resolves the domain, normalizes the name, creates the
// record, then re-fetches so the created record (with its new id) is returned.
func (c *Client) AddRecord(domainRef, name, recType, value string) (*HoverDomain, *DNSRecord, error) {
	if err := ValidateRecordType(recType); err != nil {
		return nil, nil, err
	}
	recType = strings.ToUpper(recType)
	domain, err := c.FindDomain(domainRef)
	if err != nil {
		return nil, nil, err
	}
	norm := NormalizeName(name, domain.DomainName)

	c.mu.Lock()
	addErr := c.Add(domain.ID, norm, recType, value)
	c.mu.Unlock()
	if addErr != nil {
		return nil, nil, addErr
	}

	// Re-fetch (a read) outside the write lock to recover the new record's id.
	records, err := c.DNS(domain.ID)
	if err != nil {
		return nil, nil, err
	}
	created := matchRecord(records, norm, recType, value)
	if created == nil {
		return nil, nil, fmt.Errorf("record created but not found on re-fetch")
	}
	return domain, created, nil
}

// SetRecord updates the value (and optionally the type) of the record named
// `name` in the referenced domain. It mirrors AddRecord: resolve, normalize, and
// validate in the ops layer so neither front-end carries business logic.
func (c *Client) SetRecord(domainRef, name, value, forceType string) (*HoverDomain, *DNSRecord, error) {
	forceType, err := normType(forceType)
	if err != nil {
		return nil, nil, err
	}
	domain, records, err := c.ListRecords(domainRef)
	if err != nil {
		return nil, nil, err
	}
	rec := FindByName(records, NormalizeName(name, domain.DomainName))
	if rec == nil {
		return nil, nil, fmt.Errorf("record %q not found in %s: %w", name, domain.DomainName, ErrNotFound)
	}
	updated, err := c.applySet(domain, rec, value, forceType)
	if err != nil {
		return nil, nil, err
	}
	return domain, updated, nil
}

// UpdateRecord updates recordID's value (and optionally type), but only after
// confirming the id belongs to the referenced domain (so a stray id can't update
// across domains). It is the id-keyed sibling of SetRecord, used by the HTTP API.
func (c *Client) UpdateRecord(domainRef, recordID, value, forceType string) (*HoverDomain, *DNSRecord, error) {
	forceType, err := normType(forceType)
	if err != nil {
		return nil, nil, err
	}
	domain, records, err := c.ListRecords(domainRef)
	if err != nil {
		return nil, nil, err
	}
	rec := FindByID(records, recordID)
	if rec == nil {
		return nil, nil, fmt.Errorf("record %q not found in %s: %w", recordID, domain.DomainName, ErrNotFound)
	}
	updated, err := c.applySet(domain, rec, value, forceType)
	if err != nil {
		return nil, nil, err
	}
	return domain, updated, nil
}

// applySet runs Set on rec then re-fetches the domain to return the updated
// record (Set is delete+create, so the id changes).
func (c *Client) applySet(domain *HoverDomain, rec *DNSRecord, value, forceType string) (*DNSRecord, error) {
	if err := c.Set(rec.ID, value, forceType); err != nil {
		return nil, err
	}
	records, err := c.DNS(domain.ID)
	if err != nil {
		return nil, err
	}
	recType := rec.Type
	if forceType != "" {
		recType = forceType
	}
	updated := matchRecord(records, rec.Name, recType, value)
	if updated == nil {
		return nil, fmt.Errorf("record updated but not found on re-fetch")
	}
	return updated, nil
}

// DeleteRecord deletes recordID, but only after confirming it belongs to the
// referenced domain (so a stray id can't delete across domains).
func (c *Client) DeleteRecord(domainRef, recordID string) error {
	domain, records, err := c.ListRecords(domainRef)
	if err != nil {
		return err
	}
	if FindByID(records, recordID) == nil {
		return fmt.Errorf("record %q not found in %s: %w", recordID, domain.DomainName, ErrNotFound)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Delete(recordID)
}

// normType validates and upper-cases an optional record type. An empty string
// means "leave the existing type unchanged" and is returned as-is.
func normType(t string) (string, error) {
	if t == "" {
		return "", nil
	}
	if err := ValidateRecordType(t); err != nil {
		return "", err
	}
	return strings.ToUpper(t), nil
}

// matchRecord returns the record matching name, type, and value. If no exact
// value match exists (Hover may store a normalized value, e.g. a trailing dot or
// requoted TXT), it falls back to the unique record matching name and type, and
// returns nil when that is ambiguous rather than guessing.
func matchRecord(records []DNSRecord, name, recType, value string) *DNSRecord {
	for i := range records {
		if records[i].Name == name && strings.EqualFold(records[i].Type, recType) && records[i].Value == value {
			return &records[i]
		}
	}
	var match *DNSRecord
	for i := range records {
		if records[i].Name == name && strings.EqualFold(records[i].Type, recType) {
			if match != nil {
				return nil
			}
			match = &records[i]
		}
	}
	return match
}
