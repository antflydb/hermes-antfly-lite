package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/antflydb/antfly/go/pkg/antflylite"
	"github.com/antflydb/hermes-antfly/internal/buildinfo"
	"github.com/antflydb/hermes-antfly/internal/lite"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println(buildinfo.String())
		return
	}
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "doctor":
		doctor(os.Args[2:])
	case "check":
		check(os.Args[2:])
	case "backup":
		backup(os.Args[2:])
	case "restore":
		restore(os.Args[2:])
	default:
		usage()
	}
}

type doctorReport struct {
	Ready     bool            `json:"ready"`
	Build     string          `json:"build"`
	ABI       doctorABI       `json:"abi"`
	Database  doctorFile      `json:"database"`
	Backup    doctorBackup    `json:"backup"`
	Retrieval doctorRetrieval `json:"retrieval"`
	Status    json.RawMessage `json:"status"`
	Warnings  []string        `json:"warnings"`
}

type doctorABI struct {
	Expected uint32 `json:"expected"`
	Loaded   uint32 `json:"loaded"`
	Valid    bool   `json:"valid"`
}

type doctorFile struct {
	Path        string `json:"path"`
	Size        int64  `json:"size"`
	Permissions string `json:"permissions"`
	Integrity   bool   `json:"integrity"`
	Records     uint64 `json:"records"`
}

type doctorBackup struct {
	Path        string `json:"path"`
	Size        int64  `json:"size"`
	Permissions string `json:"permissions"`
	Fresh       bool   `json:"fresh"`
}

type doctorRetrieval struct {
	Mode      string `json:"mode"`
	Query     string `json:"query"`
	TotalHits int    `json:"total_hits"`
	Healthy   bool   `json:"healthy"`
}

func doctor(args []string) {
	flags := flag.NewFlagSet("doctor", flag.ExitOnError)
	dbPath := flags.String("db", "", "existing .aflite database")
	backupPath := flags.String("backup", "", "portable .afb backup to verify")
	query := flags.String("query", "knowledge", "broad full-text health query")
	flags.Parse(args)
	requireExtension("--db", *dbPath, ".aflite")
	requireExtension("--backup", *backupPath, ".afb")
	if *query == "" {
		fail("validate arguments", errors.New("--query must not be empty"))
	}

	if err := antflylite.ValidateABI(); err != nil {
		fail("validate Antfly Lite ABI", err)
	}
	dbInfo, err := os.Stat(*dbPath)
	if err != nil {
		fail("inspect database", err)
	}
	checkReport, err := antflylite.CheckFile(*dbPath)
	if err != nil {
		fail("check database", err)
	}
	backupInfo, err := os.Stat(*backupPath)
	if err != nil {
		fail("inspect backup", err)
	}

	warnings := []string{}
	databasePrivate := dbInfo.Mode().Perm()&0o077 == 0
	backupPrivate := backupInfo.Mode().Perm()&0o077 == 0
	if !databasePrivate {
		warnings = append(warnings, "database is accessible to group or other users")
	}
	if !backupPrivate {
		warnings = append(warnings, "backup is accessible to group or other users")
	}
	backupFresh := !backupInfo.ModTime().Before(dbInfo.ModTime().Add(-time.Second))
	if !backupFresh {
		warnings = append(warnings, "backup is older than the live database")
	}

	store, err := lite.OpenReadonly(*dbPath)
	if err != nil {
		fail("open database read-only", err)
	}
	defer store.Close()
	status, err := store.Status(context.Background())
	if err != nil {
		fail("read database status", err)
	}
	search, err := store.Search(context.Background(), *query, 1)
	if err != nil {
		fail("run retrieval health query", err)
	}
	var searchSummary struct {
		TotalHits int `json:"total_hits"`
	}
	if err := json.Unmarshal(search, &searchSummary); err != nil {
		fail("decode retrieval health result", err)
	}
	if searchSummary.TotalHits == 0 {
		warnings = append(warnings, "health query returned no evidence")
	}

	abiLoaded := antflylite.ABIVersion()
	report := doctorReport{
		Ready: checkReport.Valid && backupFresh && searchSummary.TotalHits > 0 && databasePrivate && backupPrivate,
		Build: buildinfo.String(),
		ABI: doctorABI{
			Expected: antflylite.SupportedABIVersion,
			Loaded:   abiLoaded,
			Valid:    abiLoaded == antflylite.SupportedABIVersion,
		},
		Database: doctorFile{
			Path:        *dbPath,
			Size:        dbInfo.Size(),
			Permissions: dbInfo.Mode().Perm().String(),
			Integrity:   checkReport.Valid,
			Records:     checkReport.RecordCount,
		},
		Backup: doctorBackup{
			Path:        *backupPath,
			Size:        backupInfo.Size(),
			Permissions: backupInfo.Mode().Perm().String(),
			Fresh:       backupFresh,
		},
		Retrieval: doctorRetrieval{
			Mode:      "full_text",
			Query:     *query,
			TotalHits: searchSummary.TotalHits,
			Healthy:   searchSummary.TotalHits > 0,
		},
		Status:   status,
		Warnings: warnings,
	}
	writeJSON(report)
	if !report.Ready {
		os.Exit(1)
	}
}

func check(args []string) {
	flags := flag.NewFlagSet("check", flag.ExitOnError)
	dbPath := flags.String("db", "", "existing .aflite database")
	flags.Parse(args)
	requireExtension("--db", *dbPath, ".aflite")
	report, err := antflylite.CheckFile(*dbPath)
	if err != nil {
		fail("check database", err)
	}
	writeJSON(report)
	if !report.Valid {
		os.Exit(1)
	}
}

func backup(args []string) {
	flags := flag.NewFlagSet("backup", flag.ExitOnError)
	dbPath := flags.String("db", "", "existing .aflite database")
	outPath := flags.String("out", "", "new portable .afb backup")
	flags.Parse(args)
	requireExtension("--db", *dbPath, ".aflite")
	requireExtension("--out", *outPath, ".afb")
	if _, err := os.Stat(*outPath); err == nil {
		fail("write backup", fmt.Errorf("target already exists: %s", *outPath))
	} else if !errors.Is(err, os.ErrNotExist) {
		fail("inspect backup target", err)
	}
	db, err := antflylite.OpenReadonly(*dbPath)
	if err != nil {
		fail("open database read-only", err)
	}
	defer db.Close()
	if err := db.BackupToFile(*outPath); err != nil {
		fail("write backup", err)
	}
	if err := os.Chmod(*outPath, 0o600); err != nil {
		fail("secure backup permissions", err)
	}
	fmt.Printf("backup=%s\n", *outPath)
}

func restore(args []string) {
	flags := flag.NewFlagSet("restore", flag.ExitOnError)
	backupPath := flags.String("backup", "", "portable .afb backup")
	dbPath := flags.String("db", "", "new .aflite database")
	flags.Parse(args)
	requireExtension("--backup", *backupPath, ".afb")
	requireExtension("--db", *dbPath, ".aflite")
	if _, err := os.Stat(*dbPath); err == nil {
		fail("restore database", fmt.Errorf("target already exists: %s", *dbPath))
	} else if !errors.Is(err, os.ErrNotExist) {
		fail("inspect restore target", err)
	}
	if err := os.MkdirAll(filepath.Dir(*dbPath), 0o700); err != nil {
		fail("create restore directory", err)
	}
	if err := antflylite.RestoreBackupFile(*dbPath, *backupPath, false); err != nil {
		fail("restore database", err)
	}
	if err := os.Chmod(*dbPath, 0o600); err != nil {
		fail("secure restored database permissions", err)
	}
	report, err := antflylite.CheckFile(*dbPath)
	if err != nil {
		fail("check restored database", err)
	}
	if !report.Valid {
		fail("check restored database", errors.New("integrity report is not valid"))
	}
	fmt.Printf("restored=%s valid=%t records=%d\n", *dbPath, report.Valid, report.RecordCount)
}

func requireExtension(name, path, extension string) {
	if path == "" || filepath.Ext(path) != extension {
		fail("validate arguments", fmt.Errorf("%s must name a %s file", name, extension))
	}
}

func writeJSON(value any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		fail("encode result", err)
	}
}

func fail(operation string, err error) {
	fmt.Fprintf(os.Stderr, "error: %s: %v\n", operation, err)
	os.Exit(1)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: antfly-hermes-maintain <doctor|check|backup|restore> [options]")
	os.Exit(2)
}
