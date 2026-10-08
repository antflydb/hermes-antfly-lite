package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/antflydb/hermes-antfly-lite/internal/buildinfo"
	"github.com/antflydb/hermes-antfly-lite/internal/githubdocs"
)

type pathFlags []string

func (values *pathFlags) String() string { return fmt.Sprint([]string(*values)) }
func (values *pathFlags) Set(value string) error {
	*values = append(*values, value)
	return nil
}

func main() {
	var paths pathFlags
	repoDir := flag.String("repo-dir", "", "local checkout root")
	repository := flag.String("repo", "", "GitHub owner/repository")
	commit := flag.String("commit", "", "lowercase 40-character source commit SHA")
	updatedAt := flag.String("updated-at", "", "source commit time in RFC3339")
	output := flag.String("output", "", "new governed JSONL output path")
	audience := flag.String("audience", "support", "record audience")
	visibility := flag.String("visibility", "public", "record visibility")
	state := flag.String("state", "approved", "record lifecycle state")
	chunkBytes := flag.Int("chunk-bytes", githubdocs.DefaultChunkBytes, "maximum UTF-8 bytes per record")
	showVersion := flag.Bool("version", false, "print build identity and exit")
	flag.Var(&paths, "path", "repository-relative file or directory; repeatable")
	flag.Parse()
	if *showVersion {
		fmt.Println(buildinfo.String())
		return
	}
	if *repoDir == "" || *repository == "" || *commit == "" || *updatedAt == "" || *output == "" || len(paths) == 0 {
		fmt.Fprintln(os.Stderr, "error: --repo-dir, --repo, --commit, --updated-at, --output, and at least one --path are required")
		os.Exit(2)
	}
	parsedUpdatedAt, err := time.Parse(time.RFC3339, *updatedAt)
	if err != nil {
		fail("parse --updated-at", err)
	}
	records, summary, err := githubdocs.Convert(githubdocs.Config{
		Root: *repoDir, Repository: *repository, Commit: *commit, Paths: paths,
		Audience: *audience, Visibility: *visibility, State: *state,
		UpdatedAt: parsedUpdatedAt, ChunkBytes: *chunkBytes,
	})
	if err != nil {
		fail("convert GitHub documentation", err)
	}
	file, err := os.OpenFile(*output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		fail("create JSONL output", err)
	}
	writer := bufio.NewWriter(file)
	encoder := json.NewEncoder(writer)
	for _, record := range records {
		if err := encoder.Encode(record); err != nil {
			file.Close()
			fail("write JSONL output", err)
		}
	}
	if err := writer.Flush(); err != nil {
		file.Close()
		fail("flush JSONL output", err)
	}
	if err := file.Close(); err != nil {
		fail("close JSONL output", err)
	}
	fmt.Printf("created=%s repository=%s commit=%s files=%d chunks=%d source_bytes=%d\n",
		*output, *repository, *commit, summary.Files, summary.Chunks, summary.Bytes)
}

func fail(operation string, err error) {
	fmt.Fprintf(os.Stderr, "error: %s: %v\n", operation, err)
	os.Exit(1)
}
