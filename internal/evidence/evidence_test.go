package evidence

import (
	"encoding/json"
	"testing"
)

func TestNormalizeSearch(t *testing.T) {
	raw := []byte(`{"total_hits":1,"hits":[{"score":2.5,"stored_json":"{\"id\":\"support:a\",\"title\":\"Policy\",\"text\":\"Use managed auth.\",\"source_url\":\"https://docs.example/a\",\"audience\":\"support\",\"visibility\":\"internal\",\"state\":\"approved\",\"updated_at\":\"2026-10-01T00:00:00Z\"}"}]}`)
	encoded, err := NormalizeSearch(raw, "managed auth", "full_text")
	if err != nil {
		t.Fatal(err)
	}
	var result SearchResult
	if err := json.Unmarshal(encoded, &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Hits) != 1 || result.Hits[0].ID != "support:a" || result.Hits[0].Citation.URL != "https://docs.example/a" {
		t.Fatalf("unexpected normalized result: %+v", result)
	}
	if string(encoded) == string(raw) || result.Hits[0].Score != 2.5 {
		t.Fatalf("backend response was not normalized: %s", encoded)
	}
}

func TestNormalizeSourceRejectsNonApprovedRecord(t *testing.T) {
	_, err := NormalizeSource([]byte(`{"id":"a","title":"Old","text":"Do not use","source_url":"https://docs.example/old","state":"superseded"}`))
	if err == nil {
		t.Fatal("expected non-approved source to fail closed")
	}
}
