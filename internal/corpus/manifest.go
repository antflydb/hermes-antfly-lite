package corpus

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const SchemaVersion = 1

var (
	repositoryPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	commitPattern     = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

type Manifest struct {
	SchemaVersion int        `json:"schema_version"`
	CorpusID      string     `json:"corpus_id"`
	Audience      string     `json:"audience"`
	Visibility    string     `json:"visibility"`
	HealthQuery   string     `json:"health_query"`
	Source        Source     `json:"source"`
	Evaluation    Evaluation `json:"evaluation"`
	Counts        Counts     `json:"counts"`
}

type Source struct {
	Kind       string   `json:"kind"`
	Repository string   `json:"repository,omitempty"`
	Commit     string   `json:"commit,omitempty"`
	Paths      []string `json:"paths,omitempty"`
}

type Evaluation struct {
	RetrievalSuite    string `json:"retrieval_suite"`
	ConversationSuite string `json:"conversation_suite,omitempty"`
}

type Counts struct {
	Files  int `json:"files,omitempty"`
	Chunks int `json:"chunks"`
}

func PathForDatabase(databasePath string) string {
	extension := filepath.Ext(databasePath)
	return strings.TrimSuffix(databasePath, extension) + ".manifest.json"
}

func Read(path string) (Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode corpus manifest: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Manifest{}, errors.New("decode corpus manifest: trailing JSON content")
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func (manifest Manifest) Validate() error {
	if manifest.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported corpus manifest schema_version %d", manifest.SchemaVersion)
	}
	if manifest.CorpusID == "" || manifest.Audience == "" || manifest.Visibility == "" || manifest.HealthQuery == "" {
		return errors.New("corpus_id, audience, visibility, and health_query are required")
	}
	validAudience := map[string]bool{"shared": true, "support": true, "research": true, "sales": true, "marketing": true, "hr": true}
	validVisibility := map[string]bool{"public": true, "internal": true, "restricted": true}
	if !validAudience[manifest.Audience] || !validVisibility[manifest.Visibility] {
		return errors.New("invalid corpus audience or visibility")
	}
	if manifest.Source.Kind == "" || manifest.Counts.Chunks < 1 {
		return errors.New("source.kind and a positive counts.chunks are required")
	}
	if manifest.Source.Kind == "github" {
		if !repositoryPattern.MatchString(manifest.Source.Repository) || !commitPattern.MatchString(manifest.Source.Commit) || len(manifest.Source.Paths) == 0 {
			return errors.New("GitHub corpus manifests require repository, commit, and paths")
		}
	}
	for _, sourcePath := range manifest.Source.Paths {
		clean := filepath.Clean(filepath.FromSlash(sourcePath))
		if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return fmt.Errorf("source path must be repository-relative: %q", sourcePath)
		}
	}
	if manifest.Evaluation.RetrievalSuite == "" {
		return errors.New("evaluation.retrieval_suite is required")
	}
	for _, suite := range []string{manifest.Evaluation.RetrievalSuite, manifest.Evaluation.ConversationSuite} {
		if suite == "" {
			continue
		}
		clean := filepath.Clean(filepath.FromSlash(suite))
		if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return fmt.Errorf("evaluation suite must be plugin-relative: %q", suite)
		}
	}
	return nil
}

func WriteNew(path string, manifest Manifest) error {
	if err := manifest.Validate(); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode corpus manifest: %w", err)
	}
	raw = append(raw, '\n')
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(raw); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}
