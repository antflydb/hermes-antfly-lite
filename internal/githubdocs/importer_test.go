package githubdocs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestConvertProducesStableGovernedChunks(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o700); err != nil {
		t.Fatal(err)
	}
	text := "# Antfly Lite\n\nLocal database.\n\n## Search\n\n" + strings.Repeat("retrieval ", 300)
	if err := os.WriteFile(filepath.Join(root, "docs", "lite.mdx"), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	config := Config{
		Root: root, Repository: "antflydb/antfly", Commit: strings.Repeat("a", 40), Paths: []string{"docs"},
		Audience: "support", Visibility: "public", State: "approved",
		UpdatedAt: time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC), ChunkBytes: 1024,
	}
	records, summary, err := Convert(config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Files != 1 || len(records) < 2 || summary.Chunks != len(records) {
		t.Fatalf("unexpected summary: %+v records=%d", summary, len(records))
	}
	for index, record := range records {
		if len(record.Text) > 1024 || record.ChunkIndex != index+1 || record.ChunkCount != len(records) {
			t.Fatalf("invalid chunk metadata: %+v", record)
		}
		if record.SourceURL != "https://github.com/antflydb/antfly/blob/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/docs/lite.mdx" {
			t.Fatalf("unexpected source URL: %s", record.SourceURL)
		}
	}
	second, _, err := Convert(config)
	if err != nil || second[0].ID != records[0].ID {
		t.Fatalf("record IDs are not stable: %v", err)
	}
}

func TestConvertRejectsEscapingPath(t *testing.T) {
	_, _, err := Convert(Config{
		Root: t.TempDir(), Repository: "antflydb/antfly", Commit: strings.Repeat("b", 40),
		Paths: []string{"../secret"}, Audience: "support", Visibility: "public", State: "approved",
		UpdatedAt: time.Now().UTC().Add(-time.Hour),
	})
	if err == nil || !strings.Contains(err.Error(), "cannot escape") {
		t.Fatalf("expected path escape error, got %v", err)
	}
}
