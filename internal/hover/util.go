package hover

import "strings"

// NormalizeName converts a FQDN or short name to the form Hover stores:
// "www.example.com" -> "www", "example.com" -> "@", "@" or "www" -> unchanged.
func NormalizeName(name, domain string) string {
	if name == domain {
		return "@"
	}
	suffix := "." + domain
	if strings.HasSuffix(name, suffix) {
		return strings.TrimSuffix(name, suffix)
	}
	return name
}

// FindByName returns the first record with the given name, or nil.
func FindByName(records []DNSRecord, name string) *DNSRecord {
	for i := range records {
		if records[i].Name == name {
			return &records[i]
		}
	}
	return nil
}
