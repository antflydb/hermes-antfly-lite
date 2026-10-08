package corpus

import (
	"path/filepath"
	"testing"
)

func validManifest() Manifest {
	return Manifest{
		SchemaVersion: SchemaVersion,
		CorpusID:      "antfly-docs",
		Audience:      "support",
		Visibility:    "public",
		HealthQuery:   "Antfly Lite",
		Source: Source{
			Kind: "github", Repository: "antflydb/antfly", Commit: "0123456789012345678901234567890123456789",
			Paths: []string{"docs"},
		},
		Evaluation: Evaluation{RetrievalSuite: "evals/antfly-github-docs.jsonl"},
		Counts:     Counts{Files: 10, Chunks: 15},
	}
}

func TestManifestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "knowledge.manifest.json")
	if err := WriteNew(path, validManifest()); err != nil {
		t.Fatal(err)
	}
	manifest, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Source.Commit != "0123456789012345678901234567890123456789" || manifest.Counts.Chunks != 15 {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}
}

func TestManifestRejectsEscapingSuite(t *testing.T) {
	manifest := validManifest()
	manifest.Evaluation.RetrievalSuite = "../secret.jsonl"
	if err := manifest.Validate(); err == nil {
		t.Fatal("expected escaping suite to be rejected")
	}
}
