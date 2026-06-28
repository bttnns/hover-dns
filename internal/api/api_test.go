package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bttnns/hover-dns/internal/hover"
)

func TestApiKeyFromRequest(t *testing.T) {
	cases := []struct {
		auth, xkey, want string
	}{
		{"Bearer secret", "", "secret"},
		{"Bearer  secret  ", "", "secret"},
		{"", "secret", "secret"},
		{"", "  secret ", "secret"},
		{"", "", ""},
		{"Basic xyz", "fallback", "fallback"},
	}
	for _, c := range cases {
		r := httptest.NewRequest("GET", "/", nil)
		if c.auth != "" {
			r.Header.Set("Authorization", c.auth)
		}
		if c.xkey != "" {
			r.Header.Set("X-API-Key", c.xkey)
		}
		if got := apiKeyFromRequest(r); got != c.want {
			t.Errorf("apiKeyFromRequest(auth=%q xkey=%q)=%q want %q", c.auth, c.xkey, got, c.want)
		}
	}
}

func TestRequireAPIKey(t *testing.T) {
	s := &Server{apiKey: "topsecret"}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := s.requireAPIKey(next)

	cases := []struct {
		name, header, value string
		want                int
	}{
		{"valid bearer", "Authorization", "Bearer topsecret", http.StatusOK},
		{"valid xkey", "X-API-Key", "topsecret", http.StatusOK},
		{"wrong key", "X-API-Key", "nope", http.StatusUnauthorized},
		{"no key", "", "", http.StatusUnauthorized},
	}
	for _, c := range cases {
		r := httptest.NewRequest("GET", "/v1/domains", nil)
		if c.header != "" {
			r.Header.Set(c.header, c.value)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != c.want {
			t.Errorf("%s: status=%d want %d", c.name, w.Code, c.want)
		}
	}
}

func TestWriteOpError(t *testing.T) {
	s := &Server{}
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"invalid", hover.ErrInvalidInput, http.StatusBadRequest},
		{"notfound", hover.ErrNotFound, http.StatusNotFound},
		{"ratelimit", hover.ErrRateLimit, http.StatusTooManyRequests},
		{"wrapped notfound", fmt.Errorf("x: %w", hover.ErrNotFound), http.StatusNotFound},
		{"other", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		s.writeOpError(w, c.err)
		if w.Code != c.want {
			t.Errorf("%s: status=%d want %d", c.name, w.Code, c.want)
		}
	}
}

func TestToAPIRecord(t *testing.T) {
	apex := toAPIRecord("example.com", hover.DNSRecord{ID: "1", Name: "@", Type: "A", Value: "1.1.1.1", TTL: 900})
	if apex.Name != "example.com" {
		t.Errorf("apex name = %q want example.com", apex.Name)
	}
	if apex.ID != "1" || apex.Type != "A" || apex.Value != "1.1.1.1" || apex.TTL != 900 {
		t.Errorf("apex fields wrong: %+v", apex)
	}
	sub := toAPIRecord("example.com", hover.DNSRecord{Name: "www", Type: "CNAME", Value: "example.com"})
	if sub.Name != "www.example.com" {
		t.Errorf("sub name = %q want www.example.com", sub.Name)
	}
}
