// Command folio tracks the stock you hold and the stock that is still to vest.
package main

import (
	"fmt"
	"os"

	"github.com/fgrosse/folio/internal/cli"
)

// version is the version of the release this binary belongs to. The release build sets it through
// the linker, and it stays empty in every other build.
var version string

func main() {
	cmd := cli.New()
	cmd.BuildVersion = version

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
