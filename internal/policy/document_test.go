package policy

import (
	"testing"
	"time"
)

func TestEvaluateGovernance(t *testing.T) {
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	base := `{"id":"a","title":"Policy","text":"Current policy","source_url":"https://docs.example/policy","audience":"support","visibility":"internal","state":"approved","updated_at":"2026-10-01T00:00:00Z"}`
	policy := IngestPolicy{Audience: "support", MaxVisibility: "internal", Now: now}
	decision, err := policy.Evaluate([]byte(base))
	if err != nil || !decision.Include {
		t.Fatalf("approved document rejected: decision=%+v err=%v", decision, err)
	}

	cases := []struct {
		name   string
		raw    string
		reason string
	}{
		{"superseded", `{"id":"a","title":"P","text":"T","source_url":"https://e.test/a","audience":"support","visibility":"internal","state":"superseded","updated_at":"2026-10-01T00:00:00Z"}`, "state_superseded"},
		{"audience", `{"id":"a","title":"P","text":"T","source_url":"https://e.test/a","audience":"hr","visibility":"internal","state":"approved","updated_at":"2026-10-01T00:00:00Z"}`, "audience_mismatch"},
		{"visibility", `{"id":"a","title":"P","text":"T","source_url":"https://e.test/a","audience":"support","visibility":"restricted","state":"approved","updated_at":"2026-10-01T00:00:00Z"}`, "visibility_excluded"},
		{"expired", `{"id":"a","title":"P","text":"T","source_url":"https://e.test/a","audience":"support","visibility":"internal","state":"approved","updated_at":"2026-10-01T00:00:00Z","expires_at":"2026-10-07T11:00:00Z"}`, "expired_by_date"},
		{"future", `{"id":"a","title":"P","text":"T","source_url":"https://e.test/a","audience":"support","visibility":"internal","state":"approved","updated_at":"2026-10-01T00:00:00Z","effective_at":"2026-10-08T00:00:00Z"}`, "not_yet_effective"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decision, err := policy.Evaluate([]byte(tc.raw))
			if err != nil || decision.Include || decision.Reason != tc.reason {
				t.Fatalf("decision=%+v err=%v", decision, err)
			}
		})
	}
}

func TestEvaluateRejectsMissingGovernance(t *testing.T) {
	policy := IngestPolicy{Audience: "support", MaxVisibility: "internal", Now: time.Now()}
	_, err := policy.Evaluate([]byte(`{"id":"a","title":"P","text":"T","source_url":"http://example.test","audience":"support","state":"approved"}`))
	if err == nil {
		t.Fatal("expected invalid governance metadata to fail")
	}
}
