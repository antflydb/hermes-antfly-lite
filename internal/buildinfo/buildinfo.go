package buildinfo

import "fmt"

var (
	Version    = "dev"
	Commit     = "unknown"
	AntflyLite = "unknown"
	Target     = "unknown"
)

func String() string {
	return fmt.Sprintf("version=%s commit=%s antfly_lite=%s target=%s", Version, Commit, AntflyLite, Target)
}
