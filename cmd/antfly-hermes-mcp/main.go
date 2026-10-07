package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/antflydb/hermes-antfly-lite/internal/buildinfo"
	"github.com/antflydb/hermes-antfly-lite/internal/lite"
	"github.com/antflydb/hermes-antfly-lite/internal/mcp"
)

func main() {
	dbPath := flag.String("db", "", "path to an existing Antfly Lite .aflite knowledge database")
	showVersion := flag.Bool("version", false, "print build identity and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(buildinfo.String())
		return
	}
	if *dbPath == "" {
		fmt.Fprintln(os.Stderr, "error: --db is required")
		os.Exit(2)
	}

	store, err := lite.OpenReadonly(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	if err := mcp.New(store).Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "error: serve MCP: %v\n", err)
		os.Exit(1)
	}
}
