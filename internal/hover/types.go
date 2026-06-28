package hover

import (
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
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
	rc          *resty.Client
	baseURL     string
	verbose     bool
	sessionFile string
	cfg         *Config

	// mu serializes the write mutations (the delete+create in Set, the create in
	// AddRecord, the delete in DeleteRecord) so the ddns loop and the API don't
	// interleave changes against the shared session. Read-only fetches need no
	// locking; the cookie jar is concurrency-safe on its own.
	mu sync.Mutex

	// loginMu serializes the 401 re-login path (which rewrites the session file
	// and clears/sets the auth cookie) so concurrent callers don't stampede
	// Hover's auth endpoint or race on the session file.
	loginMu sync.Mutex
}

type savedSession struct {
	Value   string    `json:"value"`
	Expires time.Time `json:"expires"`
}
