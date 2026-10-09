package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/antflydb/hermes-antfly/internal/buildinfo"
	"github.com/antflydb/hermes-antfly/internal/memory"
)

type response struct {
	OK     bool   `json:"ok"`
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func main() {
	dbPath := flag.String("db", "", "profile-scoped Antfly Lite memory database")
	showVersion := flag.Bool("version", false, "print build identity and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(buildinfo.String())
		return
	}
	if *dbPath == "" || filepath.Ext(*dbPath) != ".aflite" {
		fmt.Fprintln(os.Stderr, "error: --db with an .aflite path is required")
		os.Exit(2)
	}
	decoder := json.NewDecoder(io.LimitReader(os.Stdin, 1<<20))
	decoder.DisallowUnknownFields()
	var request memory.Request
	if err := decoder.Decode(&request); err != nil {
		write(response{OK: false, Error: "invalid request: " + err.Error()})
		os.Exit(1)
	}
	result, err := memory.Handle(*dbPath, request)
	if err != nil {
		write(response{OK: false, Error: err.Error()})
		os.Exit(1)
	}
	write(response{OK: true, Result: result})
}

func write(value response) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintln(os.Stderr, "error: encode response:", err)
		os.Exit(1)
	}
}
