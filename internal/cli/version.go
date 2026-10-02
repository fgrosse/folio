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
			cmd.Printf("folio version %s %s/%s\n", cmd.version(), runtime.GOOS, runtime.GOARCH)
		},
	}
}

// version is the version of this binary, as "folio version" prints it. A release says which one it
// is. A binary that "go install" built was not handed a version, but the Go toolchain recorded the
// version of the module it built, which is the same thing. Anything else was built from a checkout
// and is "devel", which is what go calls a toolchain that is no release.
func (cmd *Folio) version() string {
	if cmd.BuildVersion != "" {
		return "v" + cmd.BuildVersion
	}

	info, ok := cmd.buildInfo()
	if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return "devel"
	}

	return info.Main.Version
}
