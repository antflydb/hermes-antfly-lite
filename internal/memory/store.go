package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/antflydb/antfly/go/pkg/antflylite"
)

const (
	indexName       = "memory_text"
	vectorIndexName = "memory_embedding_v1"
	maxText         = 16 * 1024
)

const schemaJSON = `{"version":1,"default_type":"memory","document_schemas":{"memory":{"schema":{"type":"object","required":["id","kind","text","scope","created_at","updated_at"],"additionalProperties":true}}}}`
const indexJSON = `{"name":"memory_text","kind":"full_text","config_json":"{\"fields\":[\"text\"]}"}`

type Context struct {
	AgentID   string `json:"agent_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	Workspace string `json:"workspace,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

type Record struct {
	ID         string         `json:"id"`
	Kind       string         `json:"kind"`
	Text       string         `json:"text"`
	Scope      string         `json:"scope"`
	Importance float64        `json:"importance"`
	AgentID    string         `json:"agent_id,omitempty"`
	UserID     string         `json:"user_id,omitempty"`
	Workspace  string         `json:"workspace,omitempty"`
	SessionID  string         `json:"session_id,omitempty"`
	Source     string         `json:"source,omitempty"`
	CreatedAt  string         `json:"created_at"`
	UpdatedAt  string         `json:"updated_at"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type Request struct {
	Method     string         `json:"method"`
	ID         string         `json:"id,omitempty"`
	Query      string         `json:"query,omitempty"`
	Text       string         `json:"text,omitempty"`
	Kind       string         `json:"kind,omitempty"`
	Scope      string         `json:"scope,omitempty"`
	Importance float64        `json:"importance,omitempty"`
	Source     string         `json:"source,omitempty"`
	Limit      int            `json:"limit,omitempty"`
	Context    Context        `json:"context,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	Embedding  []float64      `json:"embedding,omitempty"`
}

type SearchHit struct {
	Record
	Score float64 `json:"score"`
}

type SearchResult struct {
	Query string      `json:"query"`
	Hits  []SearchHit `json:"hits"`
}

func Handle(path string, request Request) (any, error) {
	switch request.Method {
	case "status":
		return status(path)
	case "remember":
		return remember(path, request)
	case "search":
		return search(path, request)
	case "forget":
		return forget(path, request)
	case "forget_text":
		return forgetText(path, request)
	default:
		return nil, fmt.Errorf("unknown method %q", request.Method)
	}
}

func status(path string) (any, error) {
	if err := ensureDatabase(path); err != nil {
		return nil, err
	}
	db, err := antflylite.OpenStatusOnly(path)
	if err != nil {
		return nil, fmt.Errorf("open memory database status: %w", err)
	}
	defer db.Close()
	raw, err := db.StatusJSON()
	if err != nil {
		return nil, fmt.Errorf("read memory database status: %w", err)
	}
	var full map[string]any
	if err := json.Unmarshal(raw, &full); err != nil {
		return nil, fmt.Errorf("decode memory database status: %w", err)
	}
	result := map[string]any{"ready": true}
	if storage, ok := full["storage"]; ok {
		result["storage"] = storage
	}
	if stats, ok := full["stats"].(map[string]any); ok {
		result["documents"] = stats["doc_count"]
		result["indexes"] = stats["indexes"]
	}
	return result, nil
}

func remember(path string, request Request) (Record, error) {
	request.Text = strings.TrimSpace(request.Text)
	if request.Text == "" || !utf8.ValidString(request.Text) || len(request.Text) > maxText {
		return Record{}, fmt.Errorf("text must be valid UTF-8 between 1 and %d bytes", maxText)
	}
	if request.Kind == "" {
		request.Kind = "memory"
	}
	if request.Scope == "" {
		request.Scope = defaultScope(request.Context)
	}
	if err := validateScope(request.Scope, request.Context); err != nil {
		return Record{}, err
	}
	if request.Importance == 0 {
		request.Importance = 0.5
	}
	if request.Importance < 0 || request.Importance > 1 {
		return Record{}, errors.New("importance must be between 0 and 1")
	}
	if request.ID == "" {
		request.ID = stableID(request.Kind, request.Scope, request.Text, request.Context)
	}
	if err := validateEmbedding(request.Embedding); err != nil {
		return Record{}, err
	}
	if len(request.ID) > 512 || strings.ContainsAny(request.ID, "\r\n\x00") {
		return Record{}, errors.New("id is invalid")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	record := Record{
		ID: request.ID, Kind: request.Kind, Text: request.Text, Scope: request.Scope,
		Importance: request.Importance, AgentID: request.Context.AgentID, UserID: request.Context.UserID,
		Workspace: request.Context.Workspace, SessionID: request.Context.SessionID,
		Source: request.Source, CreatedAt: now, UpdatedAt: now, Metadata: request.Metadata,
	}
	raw, err := marshalRecord(record, request.Embedding)
	if err != nil {
		return Record{}, fmt.Errorf("encode memory: %w", err)
	}
	db, err := openWriter(path)
	if err != nil {
		return Record{}, err
	}
	defer db.Close()
	if len(request.Embedding) > 0 {
		if err := ensureVectorIndex(db, len(request.Embedding)); err != nil {
			return Record{}, err
		}
	}
	if previous, lookupErr := db.Raw(record.ID); lookupErr == nil && len(previous) > 0 {
		var existing Record
		if json.Unmarshal(previous, &existing) == nil && existing.CreatedAt != "" {
			record.CreatedAt = existing.CreatedAt
			raw, _ = marshalRecord(record, request.Embedding)
		}
	}
	if err := db.Batch([]antflylite.WriteIntent{{Key: record.ID, Value: raw}}, uint64(time.Now().UnixNano())); err != nil {
		return Record{}, fmt.Errorf("write memory: %w", err)
	}
	if err := db.RunUntilIdle(); err != nil {
		return Record{}, fmt.Errorf("index memory: %w", err)
	}
	return record, nil
}

func search(path string, request Request) (SearchResult, error) {
	request.Query = strings.TrimSpace(request.Query)
	if request.Query == "" || len(request.Query) > 4096 {
		return SearchResult{}, errors.New("query must be between 1 and 4096 bytes")
	}
	if request.Limit == 0 {
		request.Limit = 6
	}
	if request.Limit < 1 || request.Limit > 20 {
		return SearchResult{}, errors.New("limit must be between 1 and 20")
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return SearchResult{Query: request.Query, Hits: []SearchHit{}}, nil
	} else if err != nil {
		return SearchResult{}, fmt.Errorf("inspect memory database: %w", err)
	}
	db, err := antflylite.OpenReadonly(path)
	if err != nil {
		return SearchResult{}, fmt.Errorf("open memory database read-only: %w", err)
	}
	defer db.Close()
	if err := validateEmbedding(request.Embedding); err != nil {
		return SearchResult{}, err
	}
	searchBody := map[string]any{
		"mode": "full_text", "index_name": indexName, "text_query_type": "match",
		"field": "text", "text": request.Query, "limit": 100,
	}
	if len(request.Embedding) > 0 {
		ready, readyErr := vectorIndexMatches(db, len(request.Embedding))
		if readyErr != nil {
			return SearchResult{}, readyErr
		}
		if ready {
			searchBody = map[string]any{
				"full_text_search": map[string]any{"match": map[string]any{"field": "text", "text": request.Query}},
				"embeddings":       map[string]any{vectorIndexName: request.Embedding},
				"indexes":          []string{vectorIndexName}, "merge_config": map[string]any{"strategy": "rrf"},
				"limit": 100,
			}
		}
	}
	searchRequest, _ := json.Marshal(searchBody)
	raw, err := db.SearchJSON(searchRequest)
	if err != nil {
		return SearchResult{}, fmt.Errorf("search memory: %w", err)
	}
	var envelope struct {
		Hits []struct {
			Score      float64 `json:"score"`
			StoredJSON string  `json:"stored_json"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return SearchResult{}, fmt.Errorf("decode memory search: %w", err)
	}
	result := SearchResult{Query: request.Query, Hits: []SearchHit{}}
	seen := map[string]struct{}{}
	for _, rawHit := range envelope.Hits {
		var record Record
		if json.Unmarshal([]byte(rawHit.StoredJSON), &record) != nil || !canAccess(record, request.Context) {
			continue
		}
		if _, duplicate := seen[record.ID]; duplicate {
			continue
		}
		seen[record.ID] = struct{}{}
		result.Hits = append(result.Hits, SearchHit{Record: record, Score: rawHit.Score})
		if len(result.Hits) == request.Limit {
			break
		}
	}
	if len(request.Embedding) > 0 {
		var hybrid struct {
			Responses []struct {
				Hits struct {
					Hits []struct {
						Score  float64 `json:"_score"`
						Source Record  `json:"_source"`
					} `json:"hits"`
				} `json:"hits"`
			} `json:"responses"`
		}
		if err := json.Unmarshal(raw, &hybrid); err != nil {
			return SearchResult{}, fmt.Errorf("decode hybrid memory search: %w", err)
		}
		for _, response := range hybrid.Responses {
			for _, hit := range response.Hits.Hits {
				if !canAccess(hit.Source, request.Context) {
					continue
				}
				if _, duplicate := seen[hit.Source.ID]; duplicate {
					continue
				}
				seen[hit.Source.ID] = struct{}{}
				result.Hits = append(result.Hits, SearchHit{Record: hit.Source, Score: hit.Score})
				if len(result.Hits) == request.Limit {
					return result, nil
				}
			}
		}
	}
	return result, nil
}

func forget(path string, request Request) (map[string]any, error) {
	if request.ID == "" {
		return nil, errors.New("id is required")
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return map[string]any{"deleted": false, "id": request.ID}, nil
	}
	db, err := openWriter(path)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	raw, err := db.Raw(request.ID)
	if errors.Is(err, antflylite.NotFound) || len(raw) == 0 {
		return map[string]any{"deleted": false, "id": request.ID}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read memory before deletion: %w", err)
	}
	var record Record
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, fmt.Errorf("decode stored memory: %w", err)
	}
	if !canAccess(record, request.Context) {
		return nil, errors.New("memory is outside the active scope")
	}
	if err := db.Batch([]antflylite.WriteIntent{{Key: request.ID, Delete: true}}, uint64(time.Now().UnixNano())); err != nil {
		return nil, fmt.Errorf("delete memory: %w", err)
	}
	if err := db.RunUntilIdle(); err != nil {
		return nil, fmt.Errorf("index memory deletion: %w", err)
	}
	return map[string]any{"deleted": true, "id": request.ID}, nil
}

func forgetText(path string, request Request) (map[string]any, error) {
	request.Text = strings.TrimSpace(request.Text)
	if request.Text == "" {
		return nil, errors.New("text is required")
	}
	if request.Kind == "" {
		request.Kind = "memory"
	}
	if request.Scope == "" {
		request.Scope = defaultScope(request.Context)
	}
	if err := validateScope(request.Scope, request.Context); err != nil {
		return nil, err
	}
	request.ID = stableID(request.Kind, request.Scope, request.Text, request.Context)
	return forget(path, request)
}

func stableID(kind, scope, text string, context Context) string {
	identity := ""
	switch scope {
	case "agent":
		identity = context.AgentID
	case "user":
		identity = context.UserID
	case "workspace":
		identity = context.Workspace
	case "session":
		identity = context.SessionID
	}
	digest := sha256.Sum256([]byte(strings.Join([]string{kind, scope, identity, text}, "\x00")))
	return "memory:" + hex.EncodeToString(digest[:16])
}

func marshalRecord(record Record, embedding []float64) ([]byte, error) {
	raw, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("encode memory: %w", err)
	}
	if len(embedding) == 0 {
		return raw, nil
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, fmt.Errorf("prepare embedded memory: %w", err)
	}
	document["_embeddings"] = map[string]any{vectorIndexName: embedding}
	raw, err = json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("encode embedded memory: %w", err)
	}
	return raw, nil
}

func validateEmbedding(embedding []float64) error {
	if len(embedding) == 0 {
		return nil
	}
	if len(embedding) < 2 || len(embedding) > 4096 {
		return errors.New("embedding dimensions must be between 2 and 4096")
	}
	for _, value := range embedding {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return errors.New("embedding contains a non-finite value")
		}
	}
	return nil
}

func ensureVectorIndex(db *antflylite.DB, dims int) error {
	matches, err := vectorIndexMatches(db, dims)
	if err != nil || matches {
		return err
	}
	config, _ := json.Marshal(map[string]any{
		"field": "embedding", "dims": dims, "metric": "cosine", "external": true,
	})
	definition, _ := json.Marshal(map[string]any{
		"name": vectorIndexName, "kind": "dense_vector", "config_json": string(config),
	})
	if err := db.AddIndexJSON(definition); err != nil {
		return fmt.Errorf("create memory vector index: %w", err)
	}
	return nil
}

func vectorIndexMatches(db *antflylite.DB, dims int) (bool, error) {
	raw, err := db.IndexesJSON()
	if err != nil {
		return false, fmt.Errorf("inspect memory indexes: %w", err)
	}
	var catalog any
	if err := json.Unmarshal(raw, &catalog); err != nil {
		return false, fmt.Errorf("decode memory indexes: %w", err)
	}
	definition := findNamedIndex(catalog, vectorIndexName)
	if definition == nil {
		return false, nil
	}
	configRaw, _ := definition["config_json"].(string)
	var config struct {
		Dims int `json:"dims"`
	}
	if json.Unmarshal([]byte(configRaw), &config) != nil || config.Dims == 0 {
		return false, errors.New("memory vector index has invalid configuration")
	}
	if config.Dims != dims {
		return false, fmt.Errorf("embedding dimensions %d do not match existing memory index dimensions %d", dims, config.Dims)
	}
	return true, nil
}

func findNamedIndex(value any, name string) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		if typed["name"] == name {
			return typed
		}
		for _, child := range typed {
			if found := findNamedIndex(child, name); found != nil {
				return found
			}
		}
	case []any:
		for _, child := range typed {
			if found := findNamedIndex(child, name); found != nil {
				return found
			}
		}
	}
	return nil
}

func ensureDatabase(path string) error {
	if err := antflylite.ValidateABI(); err != nil {
		return fmt.Errorf("validate Antfly Lite ABI: %w", err)
	}
	if _, err := os.Stat(path); err == nil {
		return waitForDatabase(path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect memory database: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create memory directory: %w", err)
	}
	db, err := antflylite.Create(path)
	if err != nil {
		if waitErr := waitForDatabase(path); waitErr == nil {
			return nil
		}
		return fmt.Errorf("create memory database: %w", err)
	}
	defer db.Close()
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("secure memory database: %w", err)
	}
	if err := db.SetSchemaJSON([]byte(schemaJSON)); err != nil {
		return fmt.Errorf("set memory schema: %w", err)
	}
	if err := db.AddIndexJSON([]byte(indexJSON)); err != nil {
		return fmt.Errorf("create memory index: %w", err)
	}
	return nil
}

func waitForDatabase(path string) error {
	var last error
	for attempt := 0; attempt < 50; attempt++ {
		db, err := antflylite.OpenStatusOnly(path)
		if err == nil {
			raw, statusErr := db.StatusJSON()
			closeErr := db.Close()
			if statusErr == nil && closeErr == nil && statusHasMemoryIndex(raw) {
				return nil
			}
			if statusErr != nil {
				last = statusErr
			} else if closeErr != nil {
				last = closeErr
			} else {
				last = errors.New("memory index is not initialized")
			}
		} else {
			last = err
		}
		time.Sleep(40 * time.Millisecond)
	}
	return fmt.Errorf("memory database did not become ready: %w", last)
}

func statusHasMemoryIndex(raw []byte) bool {
	var full struct {
		Stats struct {
			Indexes []struct {
				Name string `json:"name"`
			} `json:"indexes"`
		} `json:"stats"`
	}
	if json.Unmarshal(raw, &full) != nil {
		return false
	}
	for _, index := range full.Stats.Indexes {
		if index.Name == indexName {
			return true
		}
	}
	return false
}

func openWriter(path string) (*antflylite.DB, error) {
	if err := ensureDatabase(path); err != nil {
		return nil, err
	}
	var last error
	for attempt := 0; attempt < 7; attempt++ {
		db, err := antflylite.Open(path)
		if err == nil {
			return db, nil
		}
		last = err
		time.Sleep(time.Duration(20*(1<<attempt)) * time.Millisecond)
	}
	return nil, fmt.Errorf("open memory database for writing after retries: %w", last)
}

func defaultScope(context Context) string {
	if context.UserID != "" {
		return "user"
	}
	if context.Workspace != "" {
		return "workspace"
	}
	if context.AgentID != "" {
		return "agent"
	}
	return "profile"
}

func validateScope(scope string, context Context) error {
	switch scope {
	case "profile":
		return nil
	case "agent":
		if context.AgentID != "" {
			return nil
		}
	case "user":
		if context.UserID != "" {
			return nil
		}
	case "workspace":
		if context.Workspace != "" {
			return nil
		}
	case "session":
		if context.SessionID != "" {
			return nil
		}
	default:
		return fmt.Errorf("unknown scope %q", scope)
	}
	return fmt.Errorf("scope %q requires matching context", scope)
}

func canAccess(record Record, context Context) bool {
	switch record.Scope {
	case "profile":
		return true
	case "agent":
		return context.AgentID != "" && record.AgentID == context.AgentID
	case "user":
		return context.UserID != "" && record.UserID == context.UserID
	case "workspace":
		return context.Workspace != "" && record.Workspace == context.Workspace
	case "session":
		return context.SessionID != "" && record.SessionID == context.SessionID
	default:
		return false
	}
}
