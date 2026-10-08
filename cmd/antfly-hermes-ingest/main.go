package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/antflydb/antfly/go/pkg/antflylite"
	"github.com/antflydb/hermes-antfly-lite/internal/buildinfo"
	"github.com/antflydb/hermes-antfly-lite/internal/corpus"
	"github.com/antflydb/hermes-antfly-lite/internal/policy"
)

const schemaJSON = `{"version":1,"default_type":"knowledge","document_schemas":{"knowledge":{"schema":{"type":"object","required":["id","title","text","source_url","audience","visibility","state","updated_at"],"additionalProperties":true}}}}`
const fullTextIndexJSON = `{"name":"knowledge_text","kind":"full_text","config_json":"{\"fields\":[\"title\",\"text\"]}"}`

func main() {
	dbPath := flag.String("db", "", "new .aflite database path")
	inputPath := flag.String("input", "", "JSONL source file; mutually exclusive with --restore")
	restorePath := flag.String("restore", "", "portable .afb backup to restore; mutually exclusive with --input")
	backupPath := flag.String("backup", "", "optional portable .afb backup path")
	manifestInput := flag.String("manifest", "", "optional validated corpus manifest to install beside the database")
	audience := flag.String("audience", "support", "target audience: support, research, sales, marketing, or hr")
	maxVisibility := flag.String("max-visibility", "internal", "maximum visibility: public, internal, or restricted")
	showVersion := flag.Bool("version", false, "print build identity and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(buildinfo.String())
		return
	}
	if *dbPath == "" || (*inputPath == "") == (*restorePath == "") {
		fmt.Fprintln(os.Stderr, "error: --db and exactly one of --input or --restore are required")
		os.Exit(2)
	}
	if filepath.Ext(*dbPath) != ".aflite" {
		fmt.Fprintln(os.Stderr, "error: --db must end in .aflite")
		os.Exit(2)
	}
	if _, err := os.Stat(*dbPath); err == nil {
		fmt.Fprintf(os.Stderr, "error: database already exists: %s\n", *dbPath)
		os.Exit(1)
	} else if !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "error: inspect database path: %v\n", err)
		os.Exit(1)
	}
	manifestOutput := corpus.PathForDatabase(*dbPath)
	manifestResult := ""
	var corpusManifest corpus.Manifest
	if *manifestInput != "" {
		manifestResult = manifestOutput
		var err error
		corpusManifest, err = corpus.Read(*manifestInput)
		if err != nil {
			fail("read corpus manifest", err)
		}
		if _, err := os.Stat(manifestOutput); err == nil {
			fail("install corpus manifest", fmt.Errorf("target already exists: %s", manifestOutput))
		} else if !errors.Is(err, os.ErrNotExist) {
			fail("inspect corpus manifest target", err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(*dbPath), 0o700); err != nil {
		fmt.Fprintf(os.Stderr, "error: create database directory: %v\n", err)
		os.Exit(1)
	}
	if *restorePath != "" {
		if filepath.Ext(*restorePath) != ".afb" {
			fmt.Fprintln(os.Stderr, "error: --restore must end in .afb")
			os.Exit(2)
		}
		if *backupPath != "" {
			fmt.Fprintln(os.Stderr, "error: --backup cannot be combined with --restore")
			os.Exit(2)
		}
		if err := antflylite.RestoreBackupFile(*dbPath, *restorePath, false); err != nil {
			fail("restore database from portable backup (verify format/version compatibility)", err)
		}
		if err := os.Chmod(*dbPath, 0o600); err != nil {
			fail("secure restored database permissions", err)
		}
		check, err := antflylite.CheckFile(*dbPath)
		if err != nil {
			fail("check restored database", err)
		}
		if !check.Valid {
			fail("check restored database", errors.New("integrity report is not valid"))
		}
		if *manifestInput != "" {
			if err := corpus.WriteNew(manifestOutput, corpusManifest); err != nil {
				fail("install corpus manifest", err)
			}
		}
		fmt.Printf("restored=%s source=%s valid=%t records=%d\n", *dbPath, *restorePath, check.Valid, check.RecordCount)
		return
	}

	ingestPolicy := policy.IngestPolicy{Audience: *audience, MaxVisibility: *maxVisibility, Now: time.Now().UTC()}
	if err := ingestPolicy.Validate(); err != nil {
		fail("validate ingestion policy", err)
	}
	if *manifestInput != "" && (corpusManifest.Audience != *audience || corpusManifest.Visibility != *maxVisibility) {
		fail("validate corpus manifest", errors.New("manifest audience and visibility must match ingestion policy"))
	}
	writes, skipped, err := loadDocuments(*inputPath, ingestPolicy)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: load input: %v\n", err)
		os.Exit(1)
	}
	if len(writes) == 0 {
		fmt.Fprintln(os.Stderr, "error: input contains no approved documents")
		os.Exit(1)
	}
	if *manifestInput != "" && corpusManifest.Counts.Chunks != len(writes) {
		fail("validate corpus manifest", fmt.Errorf("manifest declares %d chunks but ingestion approved %d", corpusManifest.Counts.Chunks, len(writes)))
	}

	db, err := antflylite.Create(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: create database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := os.Chmod(*dbPath, 0o600); err != nil {
		fail("secure database permissions", err)
	}

	if err := db.SetSchemaJSON([]byte(schemaJSON)); err != nil {
		fail("set schema", err)
	}
	if err := db.AddIndexJSON([]byte(fullTextIndexJSON)); err != nil {
		fail("create full-text index", err)
	}
	if err := db.Batch(writes, uint64(time.Now().UnixNano())); err != nil {
		fail("write documents", err)
	}
	if _, err := db.RunUntilIdleStatus(); err != nil {
		fail("drain index maintenance", err)
	}
	check, err := db.Check()
	if err != nil {
		fail("check database", err)
	}
	if !check.Valid {
		fail("check database", errors.New("integrity report is not valid"))
	}
	if *backupPath != "" {
		if filepath.Ext(*backupPath) != ".afb" {
			fail("write backup", errors.New("--backup must end in .afb"))
		}
		if err := db.BackupToFile(*backupPath); err != nil {
			fail("write backup", err)
		}
		if err := os.Chmod(*backupPath, 0o600); err != nil {
			fail("secure backup permissions", err)
		}
	}
	if *manifestInput != "" {
		if err := corpus.WriteNew(manifestOutput, corpusManifest); err != nil {
			fail("install corpus manifest", err)
		}
	}

	skippedTotal := 0
	for _, count := range skipped {
		skippedTotal += count
	}
	skippedJSON, _ := json.Marshal(skipped)
	fmt.Printf("created=%s approved=%d skipped=%d skip_reasons=%s audience=%s max_visibility=%s valid=%t backup=%s manifest=%s\n",
		*dbPath, len(writes), skippedTotal, skippedJSON, *audience, *maxVisibility, check.Valid, *backupPath, manifestResult)
}

func loadDocuments(path string, ingestPolicy policy.IngestPolicy) ([]antflylite.WriteIntent, map[string]int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	var writes []antflylite.WriteIntent
	skipped := map[string]int{}
	seen := map[string]bool{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 8<<20)
	for line := 1; scanner.Scan(); line++ {
		raw := append([]byte(nil), scanner.Bytes()...)
		if len(bytes.TrimSpace(raw)) == 0 {
			continue
		}
		decision, err := ingestPolicy.Evaluate(raw)
		if err != nil {
			return nil, skipped, fmt.Errorf("line %d: %w", line, err)
		}
		if seen[decision.Document.ID] {
			return nil, skipped, fmt.Errorf("line %d: duplicate id %q", line, decision.Document.ID)
		}
		seen[decision.Document.ID] = true
		if !decision.Include {
			skipped[decision.Reason]++
			continue
		}
		writes = append(writes, antflylite.WriteIntent{Key: decision.Document.ID, Value: decision.Raw})
	}
	if err := scanner.Err(); err != nil {
		return nil, skipped, err
	}
	return writes, skipped, nil
}

func fail(operation string, err error) {
	fmt.Fprintf(os.Stderr, "error: %s: %v\n", operation, err)
	os.Exit(1)
}
