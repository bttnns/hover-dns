package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/bttnns/hover-dns/internal/hover"
)

// maxBodyBytes caps request bodies; a single DNS-record JSON is well under this.
const maxBodyBytes = 64 * 1024

// apiRecord is the wire shape for a DNS record. It differs from hover.DNSRecord:
// `value` (not Hover's `content`) and a fully-qualified `name`.
type apiRecord struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	TTL   int    `json:"ttl"`
	Value string `json:"value"`
}

func toAPIRecord(domainName string, r hover.DNSRecord) apiRecord {
	name := r.Name
	if name == "@" {
		name = domainName
	} else {
		name += "." + domainName
	}
	return apiRecord{ID: r.ID, Name: name, Type: r.Type, TTL: r.TTL, Value: r.Value}
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListDomains(w http.ResponseWriter, r *http.Request) {
	domains, err := s.c.ListDomains()
	if err != nil {
		s.writeOpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, domains)
}

func (s *Server) handleListRecords(w http.ResponseWriter, r *http.Request) {
	ref := chi.URLParam(r, "domain")
	domain, records, err := s.c.ListRecords(ref)
	if err != nil {
		s.writeOpError(w, err)
		return
	}

	nameFilter := r.URL.Query().Get("name")
	if nameFilter != "" {
		nameFilter = hover.NormalizeName(nameFilter, domain.DomainName)
	}
	typeFilter := r.URL.Query().Get("type")

	out := make([]apiRecord, 0, len(records))
	for _, rec := range records {
		if nameFilter != "" && rec.Name != nameFilter {
			continue
		}
		if typeFilter != "" && !strings.EqualFold(rec.Type, typeFilter) {
			continue
		}
		out = append(out, toAPIRecord(domain.DomainName, rec))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleAddRecord(w http.ResponseWriter, r *http.Request) {
	ref := chi.URLParam(r, "domain")

	var body struct {
		Name  string `json:"name"`
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.Name == "" || body.Type == "" || body.Value == "" {
		writeError(w, http.StatusBadRequest, "name, type, and value are required")
		return
	}

	domain, rec, err := s.c.AddRecord(ref, body.Name, body.Type, body.Value)
	if err != nil {
		s.writeOpError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toAPIRecord(domain.DomainName, *rec))
}

func (s *Server) handleUpdateRecord(w http.ResponseWriter, r *http.Request) {
	ref := chi.URLParam(r, "domain")
	id := chi.URLParam(r, "id")

	var body struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.Value == "" {
		writeError(w, http.StatusBadRequest, "value is required")
		return
	}

	domain, rec, err := s.c.UpdateRecord(ref, id, body.Value, body.Type)
	if err != nil {
		s.writeOpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPIRecord(domain.DomainName, *rec))
}

func (s *Server) handleDeleteRecord(w http.ResponseWriter, r *http.Request) {
	ref := chi.URLParam(r, "domain")
	id := chi.URLParam(r, "id")

	if err := s.c.DeleteRecord(ref, id); err != nil {
		s.writeOpError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// writeOpError maps hover sentinel errors to status codes.
func (s *Server) writeOpError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, hover.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, hover.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, hover.ErrRateLimit):
		writeError(w, http.StatusTooManyRequests, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}
