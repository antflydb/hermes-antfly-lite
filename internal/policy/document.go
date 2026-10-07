package policy

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

var audiences = map[string]bool{
	"shared": true, "support": true, "research": true, "sales": true, "marketing": true, "hr": true,
}

var visibilityRank = map[string]int{"public": 0, "internal": 1, "restricted": 2}

var lifecycleStates = map[string]bool{
	"draft": true, "approved": true, "superseded": true, "expired": true,
}

type Document struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Text        string   `json:"text"`
	SourceURL   string   `json:"source_url"`
	Audience    string   `json:"audience"`
	Visibility  string   `json:"visibility"`
	State       string   `json:"state"`
	EffectiveAt string   `json:"effective_at,omitempty"`
	ExpiresAt   string   `json:"expires_at,omitempty"`
	UpdatedAt   string   `json:"updated_at"`
	RiskLabels  []string `json:"risk_labels,omitempty"`
}

type IngestPolicy struct {
	Audience      string
	MaxVisibility string
	Now           time.Time
}

type Decision struct {
	Document Document
	Raw      []byte
	Include  bool
	Reason   string
}

func (p IngestPolicy) Validate() error {
	if !audiences[p.Audience] || p.Audience == "shared" {
		return fmt.Errorf("unsupported target audience %q", p.Audience)
	}
	if _, ok := visibilityRank[p.MaxVisibility]; !ok {
		return fmt.Errorf("unsupported maximum visibility %q", p.MaxVisibility)
	}
	return nil
}

func (p IngestPolicy) Evaluate(raw []byte) (Decision, error) {
	if err := p.Validate(); err != nil {
		return Decision{}, err
	}
	var doc Document
	if err := json.Unmarshal(raw, &doc); err != nil {
		return Decision{}, err
	}
	if strings.TrimSpace(doc.ID) == "" || strings.TrimSpace(doc.Title) == "" || strings.TrimSpace(doc.Text) == "" {
		return Decision{}, fmt.Errorf("id, title, and text are required")
	}
	if len(doc.ID) > 512 || len(doc.Title) > 300 || len(doc.Text) > 16*1024 || len(doc.SourceURL) > 2048 {
		return Decision{}, fmt.Errorf("document exceeds an ingestion size limit")
	}
	if len(doc.RiskLabels) > 16 {
		return Decision{}, fmt.Errorf("risk_labels exceeds 16 entries")
	}
	for _, label := range doc.RiskLabels {
		if strings.TrimSpace(label) == "" || len(label) > 64 {
			return Decision{}, fmt.Errorf("risk_labels entries must be non-empty and at most 64 bytes")
		}
	}
	if !lifecycleStates[doc.State] {
		return Decision{}, fmt.Errorf("state must be one of approved, draft, expired, or superseded")
	}
	if !audiences[doc.Audience] {
		return Decision{}, fmt.Errorf("unsupported audience %q", doc.Audience)
	}
	documentVisibility, ok := visibilityRank[doc.Visibility]
	if !ok {
		return Decision{}, fmt.Errorf("visibility must be public, internal, or restricted")
	}
	parsedURL, err := url.Parse(doc.SourceURL)
	if err != nil || parsedURL.Scheme != "https" || parsedURL.Host == "" || parsedURL.User != nil {
		return Decision{}, fmt.Errorf("source_url must be an absolute https URL")
	}
	updatedAt, err := parseRequiredTime("updated_at", doc.UpdatedAt)
	if err != nil {
		return Decision{}, err
	}
	if updatedAt.After(p.Now.Add(5 * time.Minute)) {
		return Decision{}, fmt.Errorf("updated_at is in the future")
	}
	if doc.State != "approved" {
		return Decision{Document: doc, Raw: append([]byte(nil), raw...), Reason: "state_" + doc.State}, nil
	}
	if doc.Audience != "shared" && doc.Audience != p.Audience {
		return Decision{Document: doc, Raw: append([]byte(nil), raw...), Reason: "audience_mismatch"}, nil
	}
	if documentVisibility > visibilityRank[p.MaxVisibility] {
		return Decision{Document: doc, Raw: append([]byte(nil), raw...), Reason: "visibility_excluded"}, nil
	}
	if doc.EffectiveAt != "" {
		effectiveAt, err := time.Parse(time.RFC3339, doc.EffectiveAt)
		if err != nil {
			return Decision{}, fmt.Errorf("effective_at must be RFC3339: %w", err)
		}
		if effectiveAt.After(p.Now) {
			return Decision{Document: doc, Raw: append([]byte(nil), raw...), Reason: "not_yet_effective"}, nil
		}
	}
	if doc.ExpiresAt != "" {
		expiresAt, err := time.Parse(time.RFC3339, doc.ExpiresAt)
		if err != nil {
			return Decision{}, fmt.Errorf("expires_at must be RFC3339: %w", err)
		}
		if !expiresAt.After(p.Now) {
			return Decision{Document: doc, Raw: append([]byte(nil), raw...), Reason: "expired_by_date"}, nil
		}
	}
	return Decision{Document: doc, Raw: append([]byte(nil), raw...), Include: true}, nil
}

func parseRequiredTime(name, value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("%s is required", name)
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be RFC3339: %w", name, err)
	}
	return parsed, nil
}
