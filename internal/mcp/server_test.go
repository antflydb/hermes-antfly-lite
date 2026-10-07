package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type fakeStore struct{}

func (fakeStore) Status(context.Context) (json.RawMessage, error) {
	return json.RawMessage(`{"storage":{"format":"aflite"}}`), nil
}

func (fakeStore) Search(_ context.Context, query string, limit int) (json.RawMessage, error) {
	value, _ := json.Marshal(map[string]any{"query": query, "limit": limit})
	return value, nil
}

func (fakeStore) GetSource(_ context.Context, id string) (json.RawMessage, error) {
	value, _ := json.Marshal(map[string]any{"id": id})
	return value, nil
}

func TestInitializeAndTools(t *testing.T) {
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
	}, "\n") + "\n"
	var output bytes.Buffer
	if err := New(fakeStore{}).Serve(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d responses, want 2: %s", len(lines), output.String())
	}
	if !strings.Contains(lines[0], `"protocolVersion":"2025-06-18"`) {
		t.Fatalf("initialize response: %s", lines[0])
	}
	for _, name := range []string{"search_knowledge", "get_source", "knowledge_status"} {
		if !strings.Contains(lines[1], name) {
			t.Fatalf("tools response missing %s: %s", name, lines[1])
		}
	}
	if strings.Contains(lines[1], "backup") || strings.Contains(lines[1], "restore") {
		t.Fatalf("tools response exposed administrative operation: %s", lines[1])
	}
}

func TestSearchDefaultsAndBounds(t *testing.T) {
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search_knowledge","arguments":{"query":"managed auth"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"search_knowledge","arguments":{"query":"x","limit":7}}}`,
	}, "\n") + "\n"
	var output bytes.Buffer
	if err := New(fakeStore{}).Serve(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if !strings.Contains(lines[0], `\"limit\":6`) {
		t.Fatalf("default search response: %s", lines[0])
	}
	if !strings.Contains(lines[0], `"structuredContent":{"limit":6,"query":"managed auth"}`) {
		t.Fatalf("search response missing structured content: %s", lines[0])
	}
	if !strings.Contains(lines[1], `"isError":true`) || !strings.Contains(lines[1], "between 1 and 6") {
		t.Fatalf("bounded search response: %s", lines[1])
	}
}
