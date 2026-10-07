package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/antflydb/antfly/go/pkg/antflylite"
)

func main() {
	dbPath := flag.String("db", "", "existing .aflite database")
	mode := flag.String("mode", "readonly", "open mode: writer or readonly")
	hold := flag.Duration("hold", 0, "time to keep the handle open")
	flag.Parse()

	var (
		db  *antflylite.DB
		err error
	)
	switch *mode {
	case "writer":
		db, err = antflylite.Open(*dbPath)
	case "readonly":
		db, err = antflylite.OpenReadonly(*dbPath)
	default:
		fmt.Fprintf(os.Stderr, "error: unsupported mode %q\n", *mode)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "open_error=%v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	if _, err := db.Status(); err != nil {
		fmt.Fprintf(os.Stderr, "status_error=%v\n", err)
		os.Exit(1)
	}
	fmt.Printf("opened=%s\n", *mode)
	if *hold > 0 {
		time.Sleep(*hold)
	}
}
