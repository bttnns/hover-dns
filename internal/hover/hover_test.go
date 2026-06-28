package hover

import (
	"errors"
	"testing"
)

func TestNormalizeName(t *testing.T) {
	cases := []struct{ name, domain, want string }{
		{"www.example.com", "example.com", "www"},
		{"example.com", "example.com", "@"},
		{"@", "example.com", "@"},
		{"www", "example.com", "www"},
		{"a.b.example.com", "example.com", "a.b"},
	}
	for _, c := range cases {
		if got := NormalizeName(c.name, c.domain); got != c.want {
			t.Errorf("NormalizeName(%q,%q)=%q want %q", c.name, c.domain, got, c.want)
		}
	}
}

func TestFindByNameAndID(t *testing.T) {
	recs := []DNSRecord{
		{ID: "1", Name: "@", Type: "A", Value: "1.1.1.1"},
		{ID: "2", Name: "www", Type: "CNAME", Value: "example.com"},
	}
	if r := FindByName(recs, "www"); r == nil || r.ID != "2" {
		t.Errorf("FindByName(www) = %v", r)
	}
	if r := FindByName(recs, "nope"); r != nil {
		t.Errorf("FindByName(nope) = %v want nil", r)
	}
	if r := FindByID(recs, "1"); r == nil || r.Name != "@" {
		t.Errorf("FindByID(1) = %v", r)
	}
	if r := FindByID(recs, "9"); r != nil {
		t.Errorf("FindByID(9) = %v want nil", r)
	}
}

func TestValidateRecordType(t *testing.T) {
	if err := ValidateRecordType("a"); err != nil {
		t.Errorf("ValidateRecordType(a) = %v want nil", err)
	}
	err := ValidateRecordType("bogus")
	if err == nil {
		t.Fatal("ValidateRecordType(bogus) = nil want error")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("ValidateRecordType(bogus) error not ErrInvalidInput: %v", err)
	}
}

func TestNormType(t *testing.T) {
	if got, err := normType(""); got != "" || err != nil {
		t.Errorf(`normType("") = %q,%v want "",nil`, got, err)
	}
	if got, err := normType("a"); got != "A" || err != nil {
		t.Errorf(`normType("a") = %q,%v want "A",nil`, got, err)
	}
	if _, err := normType("nope"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("normType(nope) err = %v want ErrInvalidInput", err)
	}
}

func TestStatusError(t *testing.T) {
	if err := statusError("create failed", 429, "slow down"); !errors.Is(err, ErrRateLimit) {
		t.Errorf("statusError(429) not ErrRateLimit: %v", err)
	}
	if err := statusError("create failed", 500, "boom"); errors.Is(err, ErrRateLimit) {
		t.Errorf("statusError(500) should not be ErrRateLimit: %v", err)
	}
}

func TestMatchRecord(t *testing.T) {
	recs := []DNSRecord{
		{ID: "1", Name: "@", Type: "A", Value: "1.1.1.1"},
		{ID: "2", Name: "mail", Type: "MX", Value: "10 mail.example.com."},
		{ID: "3", Name: "multi", Type: "A", Value: "1.2.3.4"},
		{ID: "4", Name: "multi", Type: "A", Value: "5.6.7.8"},
	}

	if r := matchRecord(recs, "@", "A", "1.1.1.1"); r == nil || r.ID != "1" {
		t.Errorf("exact match = %v", r)
	}
	if r := matchRecord(recs, "@", "a", "1.1.1.1"); r == nil || r.ID != "1" {
		t.Errorf("case-insensitive type match = %v", r)
	}
	// Value normalized by Hover (trailing dot) -> fall back to unique name+type.
	if r := matchRecord(recs, "mail", "MX", "10 mail.example.com"); r == nil || r.ID != "2" {
		t.Errorf("normalized-value fallback = %v", r)
	}
	// Ambiguous name+type with no exact value match -> nil (don't guess).
	if r := matchRecord(recs, "multi", "A", "9.9.9.9"); r != nil {
		t.Errorf("ambiguous fallback = %v want nil", r)
	}
	// Exact value still wins among duplicates.
	if r := matchRecord(recs, "multi", "A", "5.6.7.8"); r == nil || r.ID != "4" {
		t.Errorf("exact among duplicates = %v", r)
	}
	if r := matchRecord(recs, "nope", "A", "1.1.1.1"); r != nil {
		t.Errorf("no match = %v want nil", r)
	}
}
