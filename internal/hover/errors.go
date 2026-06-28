package hover

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors let callers (the HTTP API in particular) map failures to the
// right status code via errors.Is, without string matching.
var (
	// ErrInvalidInput marks a caller mistake (bad record type, etc.) -> HTTP 400.
	ErrInvalidInput = errors.New("invalid input")
	// ErrNotFound marks a missing domain or record -> HTTP 404.
	ErrNotFound = errors.New("not found")
	// ErrRateLimit marks a Hover rate-limit response (HTTP 429) -> HTTP 429.
	ErrRateLimit = errors.New("rate limited")
)

// statusError builds an error for a non-2xx Hover HTTP response. A 429 is tagged
// with ErrRateLimit so the API maps it to HTTP 429 instead of a generic 500.
func statusError(context string, status int, detail string) error {
	msg := fmt.Sprintf("%s (HTTP %d)", context, status)
	if detail != "" {
		msg += ": " + detail
	}
	if status == 429 {
		return fmt.Errorf("%s: %w", msg, ErrRateLimit)
	}
	return errors.New(msg)
}

var validRecordTypes = map[string]bool{
	"A": true, "AAAA": true, "CNAME": true,
	"MX": true, "TXT": true, "SRV": true,
}

// ValidateRecordType reports whether t is a supported DNS record type. It is the
// single validator shared by the CLI and the API.
func ValidateRecordType(t string) error {
	if !validRecordTypes[strings.ToUpper(t)] {
		return fmt.Errorf("invalid record type %q: must be one of A, AAAA, CNAME, MX, TXT, SRV: %w", t, ErrInvalidInput)
	}
	return nil
}
