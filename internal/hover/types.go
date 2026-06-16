package hover

import (
	"net/http"
	"time"
)

type HoverDomain struct {
	ID         string `json:"id"`
	DomainName string `json:"domain_name"`
}

type DNSRecord struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"content"`
	TTL   int    `json:"ttl"`
}

type domainsResponse struct {
	Succeeded bool          `json:"succeeded"`
	Domains   []HoverDomain `json:"domains"`
}

type dnsResponse struct {
	Succeeded bool `json:"succeeded"`
	Domains   []struct {
		Entries []DNSRecord `json:"entries"`
	} `json:"domains"`
}

type Client struct {
	http        *http.Client
	baseURL     string
	verbose     bool
	sessionFile string
	cfg         *Config
}

type savedSession struct {
	Value   string    `json:"value"`
	Expires time.Time `json:"expires"`
}
