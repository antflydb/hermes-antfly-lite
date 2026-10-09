package lite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/antflydb/antfly/go/pkg/antflylite"
	"github.com/antflydb/hermes-antfly/internal/corpus"
	"github.com/antflydb/hermes-antfly/internal/evidence"
)

const (
	fullTextIndex = "knowledge_text"
	textField     = "text"
)

type Store struct {
	db           *antflylite.DB
	manifestPath string
}

func OpenReadonly(path string) (*Store, error) {
	if err := antflylite.ValidateABI(); err != nil {
		return nil, fmt.Errorf("validate Antfly Lite ABI: %w", err)
	}
	db, err := antflylite.OpenReadonly(path)
	if err != nil {
		return nil, fmt.Errorf("open Antfly Lite database read-only: %w", err)
	}
	return &Store{db: db, manifestPath: corpus.PathForDatabase(path)}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Status(context.Context) (json.RawMessage, error) {
	value, err := s.db.StatusJSON()
	if err != nil {
		return nil, fmt.Errorf("read knowledge status: %w", err)
	}
	var status map[string]any
	if err := json.Unmarshal(value, &status); err != nil {
		return nil, fmt.Errorf("decode knowledge status: %w", err)
	}
	manifest, err := corpus.Read(s.manifestPath)
	if err == nil {
		status["corpus"] = manifest
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read knowledge corpus manifest: %w", err)
	}
	combined, err := json.Marshal(status)
	if err != nil {
		return nil, fmt.Errorf("encode knowledge status: %w", err)
	}
	return json.RawMessage(combined), nil
}

func (s *Store) Search(_ context.Context, query string, limit int) (json.RawMessage, error) {
	request, err := json.Marshal(map[string]any{
		"mode":            "full_text",
		"index_name":      fullTextIndex,
		"text_query_type": "match",
		"field":           textField,
		"text":            query,
		"limit":           limit,
	})
	if err != nil {
		return nil, fmt.Errorf("build knowledge query: %w", err)
	}
	value, err := s.db.SearchJSON(request)
	if err != nil {
		return nil, fmt.Errorf("search knowledge: %w", err)
	}
	normalized, err := evidence.NormalizeSearch(value, query, "full_text")
	if err != nil {
		return nil, fmt.Errorf("normalize knowledge search: %w", err)
	}
	return json.RawMessage(normalized), nil
}

func (s *Store) GetSource(_ context.Context, id string) (json.RawMessage, error) {
	value, err := s.db.LookupJSON(id)
	if err != nil {
		return nil, fmt.Errorf("get knowledge source %q: %w", id, err)
	}
	normalized, err := evidence.NormalizeSource(value)
	if err != nil {
		return nil, fmt.Errorf("normalize knowledge source %q: %w", id, err)
	}
	return json.RawMessage(normalized), nil
}
