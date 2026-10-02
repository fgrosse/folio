// Command folio tracks the stock you hold and the stock that is still to vest.
package main

import (
	"fmt"
	"os"

	"github.com/fgrosse/folio/internal/cli"
)

func main() {
	if err := cli.New().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
