package githubdocs

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	DefaultChunkBytes = 12 * 1024
	MaxSourceBytes    = 2 * 1024 * 1024
)

var (
	repositoryPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	commitPattern     = regexp.MustCompile(`^[0-9a-f]{40}$`)
	allowedExtensions = map[string]bool{".md": true, ".mdx": true, ".txt": true, ".rst": true}
)

type Config struct {
	Root       string
	Repository string
	Commit     string
	Paths      []string
	Audience   string
	Visibility string
	State      string
	UpdatedAt  time.Time
	ChunkBytes int
}

type Record struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Text             string `json:"text"`
	SourceURL        string `json:"source_url"`
	Audience         string `json:"audience"`
	Visibility       string `json:"visibility"`
	State            string `json:"state"`
	UpdatedAt        string `json:"updated_at"`
	SourceKind       string `json:"source_kind"`
	SourceRepository string `json:"source_repository"`
	SourceCommit     string `json:"source_commit"`
	SourcePath       string `json:"source_path"`
	ChunkIndex       int    `json:"chunk_index"`
	ChunkCount       int    `json:"chunk_count"`
}

type Summary struct {
	Files  int
	Chunks int
	Bytes  int64
}

type sourceFile struct {
	absolute string
	relative string
}

func Convert(config Config) ([]Record, Summary, error) {
	if err := validateConfig(&config); err != nil {
		return nil, Summary{}, err
	}
	root, err := filepath.Abs(config.Root)
	if err != nil {
		return nil, Summary{}, fmt.Errorf("resolve repository root: %w", err)
	}
	files, err := collectFiles(root, config.Paths)
	if err != nil {
		return nil, Summary{}, err
	}
	if len(files) == 0 {
		return nil, Summary{}, fmt.Errorf("selected paths contain no supported documentation files")
	}

	records := make([]Record, 0, len(files))
	summary := Summary{Files: len(files)}
	for _, file := range files {
		info, err := os.Stat(file.absolute)
		if err != nil {
			return nil, Summary{}, fmt.Errorf("inspect %s: %w", file.relative, err)
		}
		if info.Size() > MaxSourceBytes {
			return nil, Summary{}, fmt.Errorf("source file exceeds %d bytes: %s", MaxSourceBytes, file.relative)
		}
		raw, err := os.ReadFile(file.absolute)
		if err != nil {
			return nil, Summary{}, fmt.Errorf("read %s: %w", file.relative, err)
		}
		if !utf8.Valid(raw) {
			return nil, Summary{}, fmt.Errorf("source file is not valid UTF-8: %s", file.relative)
		}
		summary.Bytes += int64(len(raw))
		chunks := chunkText(string(raw), config.ChunkBytes)
		if len(chunks) == 0 {
			continue
		}
		documentTitle := titleFor(file.relative, string(raw))
		pathHash := sha256.Sum256([]byte(config.Repository + "\x00" + file.relative))
		for index, chunk := range chunks {
			title := documentTitle
			if section := firstHeading(chunk); section != "" && section != documentTitle {
				title = documentTitle + " — " + section
			}
			title = truncateUTF8(title, 300)
			records = append(records, Record{
				ID:               fmt.Sprintf("github:%s:%s:%04d", config.Repository, hex.EncodeToString(pathHash[:8]), index+1),
				Title:            title,
				Text:             chunk,
				SourceURL:        githubBlobURL(config.Repository, config.Commit, file.relative),
				Audience:         config.Audience,
				Visibility:       config.Visibility,
				State:            config.State,
				UpdatedAt:        config.UpdatedAt.UTC().Format(time.RFC3339),
				SourceKind:       "github",
				SourceRepository: config.Repository,
				SourceCommit:     config.Commit,
				SourcePath:       file.relative,
				ChunkIndex:       index + 1,
				ChunkCount:       len(chunks),
			})
		}
	}
	if len(records) == 0 {
		return nil, Summary{}, fmt.Errorf("selected documentation files contain no text")
	}
	summary.Chunks = len(records)
	return records, summary, nil
}

func validateConfig(config *Config) error {
	if config.Root == "" || !repositoryPattern.MatchString(config.Repository) || !commitPattern.MatchString(config.Commit) {
		return fmt.Errorf("root, owner/repository, and a lowercase 40-character commit SHA are required")
	}
	if len(config.Paths) == 0 {
		return fmt.Errorf("at least one repository-relative path is required")
	}
	if config.UpdatedAt.IsZero() {
		return fmt.Errorf("updated_at is required")
	}
	if config.UpdatedAt.After(time.Now().UTC().Add(5 * time.Minute)) {
		return fmt.Errorf("updated_at cannot be in the future")
	}
	if config.ChunkBytes == 0 {
		config.ChunkBytes = DefaultChunkBytes
	}
	if config.ChunkBytes < 1024 || config.ChunkBytes > 16*1024 {
		return fmt.Errorf("chunk bytes must be between 1024 and 16384")
	}
	validAudience := map[string]bool{"shared": true, "support": true, "research": true, "sales": true, "marketing": true, "hr": true}
	validVisibility := map[string]bool{"public": true, "internal": true, "restricted": true}
	validState := map[string]bool{"approved": true, "draft": true, "expired": true, "superseded": true}
	if !validAudience[config.Audience] || !validVisibility[config.Visibility] || !validState[config.State] {
		return fmt.Errorf("invalid audience, visibility, or state")
	}
	return nil
}

func collectFiles(root string, paths []string) ([]sourceFile, error) {
	seen := map[string]sourceFile{}
	for _, requested := range paths {
		clean := filepath.Clean(filepath.FromSlash(requested))
		if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("path must be repository-relative and cannot escape the root: %q", requested)
		}
		selected := filepath.Join(root, clean)
		rel, err := filepath.Rel(root, selected)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("path escapes repository root: %q", requested)
		}
		if err := filepath.WalkDir(selected, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() {
				if path != selected && (entry.Name() == ".git" || entry.Name() == "node_modules" || entry.Name() == "vendor") {
					return filepath.SkipDir
				}
				return nil
			}
			if !allowedExtensions[strings.ToLower(filepath.Ext(entry.Name()))] {
				return nil
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			relative = filepath.ToSlash(relative)
			seen[relative] = sourceFile{absolute: path, relative: relative}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("walk %q: %w", requested, err)
		}
	}
	files := make([]sourceFile, 0, len(seen))
	for _, file := range seen {
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].relative < files[j].relative })
	return files, nil
}

func chunkText(raw string, maxBytes int) []string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var chunks []string
	var current strings.Builder
	flush := func() {
		if text := strings.TrimSpace(current.String()); text != "" {
			chunks = append(chunks, text)
		}
		current.Reset()
	}
	for _, line := range strings.Split(raw, "\n") {
		lineWithNewline := line + "\n"
		if current.Len() > 0 && current.Len()+len(lineWithNewline) > maxBytes {
			flush()
		}
		for len(lineWithNewline) > maxBytes {
			cut := validUTF8Cut(lineWithNewline, maxBytes)
			current.WriteString(lineWithNewline[:cut])
			flush()
			lineWithNewline = lineWithNewline[cut:]
		}
		current.WriteString(lineWithNewline)
	}
	flush()
	return chunks
}

func validUTF8Cut(value string, limit int) int {
	if len(value) <= limit {
		return len(value)
	}
	cut := limit
	for cut > 0 && !utf8.RuneStart(value[cut]) {
		cut--
	}
	if cut == 0 {
		_, size := utf8.DecodeRuneInString(value)
		return size
	}
	return cut
}

func titleFor(path, raw string) string {
	if heading := firstHeading(raw); heading != "" {
		return heading
	}
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	base = strings.ReplaceAll(base, "-", " ")
	base = strings.ReplaceAll(base, "_", " ")
	return strings.TrimSpace(base)
}

func firstHeading(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			heading := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			if heading != "" {
				return heading
			}
		}
	}
	return ""
}

func truncateUTF8(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return strings.TrimSpace(value[:validUTF8Cut(value, limit)])
}

func githubBlobURL(repository, commit, path string) string {
	segments := strings.Split(filepath.ToSlash(path), "/")
	for index, segment := range segments {
		segments[index] = url.PathEscape(segment)
	}
	return "https://github.com/" + repository + "/blob/" + commit + "/" + strings.Join(segments, "/")
}
