package evidence

import (
	"encoding/json"
	"fmt"
)

type Citation struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	UpdatedAt string `json:"updated_at"`
}

type Source struct {
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
	Score       float64  `json:"score,omitempty"`
	Citation    Citation `json:"citation"`
}

type SearchResult struct {
	Query     string   `json:"query"`
	Mode      string   `json:"mode"`
	TotalHits int      `json:"total_hits"`
	Hits      []Source `json:"hits"`
}

func NormalizeSearch(raw []byte, query, mode string) ([]byte, error) {
	var backend struct {
		TotalHits int `json:"total_hits"`
		Hits      []struct {
			Score      float64 `json:"score"`
			StoredJSON string  `json:"stored_json"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(raw, &backend); err != nil {
		return nil, fmt.Errorf("decode Antfly search response: %w", err)
	}
	result := SearchResult{Query: query, Mode: mode, TotalHits: backend.TotalHits, Hits: make([]Source, 0, len(backend.Hits))}
	for index, hit := range backend.Hits {
		source, err := decodeSource([]byte(hit.StoredJSON))
		if err != nil {
			return nil, fmt.Errorf("decode Antfly hit %d: %w", index, err)
		}
		source.Score = hit.Score
		result.Hits = append(result.Hits, source)
	}
	return json.Marshal(result)
}

func NormalizeSource(raw []byte) ([]byte, error) {
	source, err := decodeSource(raw)
	if err != nil {
		return nil, fmt.Errorf("decode Antfly source: %w", err)
	}
	return json.Marshal(map[string]Source{"source": source})
}

func decodeSource(raw []byte) (Source, error) {
	var source Source
	if err := json.Unmarshal(raw, &source); err != nil {
		return Source{}, err
	}
	if source.ID == "" || source.Title == "" || source.Text == "" || source.SourceURL == "" || source.State != "approved" {
		return Source{}, fmt.Errorf("stored source violates the approved evidence contract")
	}
	source.Citation = Citation{Title: source.Title, URL: source.SourceURL, UpdatedAt: source.UpdatedAt}
	return source, nil
}
