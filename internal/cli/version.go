package cli

import (
	"runtime"

	"github.com/spf13/cobra"
)

// VersionCmd returns the "folio version" command.
func (cmd *Folio) VersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version of folio",
		Long: `
Print the version of folio and the platform it was built for, such as
"folio version v1.0.0 linux/amd64".
`,
		Args: cobra.NoArgs,
		// The version is a fact about the binary, not about an account: it must not open the
		// database that the root command opens for every other verb, let alone create it.
		PersistentPreRunE:  func(*cobra.Command, []string) error { return nil },
		PersistentPostRunE: func(*cobra.Command, []string) error { return nil },
		Run: func(*cobra.Command, []string) {
			cmd.Printf("folio version v%s %s/%s\n", cmd.BuildVersion, runtime.GOOS, runtime.GOARCH)
		},
	}
}
