package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fgrosse/folio/internal/update"
)

// SelfUpdateCmd returns the "folio self-update" command.
func (cmd *Folio) SelfUpdateCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "self-update [version]",
		Short: "Update folio to the latest release",
		Args:  cobra.MaximumNArgs(1),
		// An update is about the binary, not about an account: like the version, it must not
		// open the database that the root command opens for every other verb.
		PersistentPreRunE:  func(*cobra.Command, []string) error { return nil },
		PersistentPostRunE: func(*cobra.Command, []string) error { return nil },
		RunE: func(c *cobra.Command, args []string) error {
			yes, err := c.Flags().GetBool("yes")
			if err != nil {
				return err
			}

			c.SilenceUsage = true // past this point, errors are runtime problems, not misuse

			var version string
			if len(args) == 1 {
				version = args[0]
			}

			return cmd.selfUpdate(c.Context(), version, yes, c.InOrStdin())
		},
	}

	c.Flags().BoolP("yes", "y", false, "update without asking")

	return c
}

// selfUpdate replaces the binary that runs with the one of the release of version, or of the
// latest release if version is empty. Unless yes says that the user agreed already, it asks them
// on stdout first and reads the answer from in.
func (cmd *Folio) selfUpdate(ctx context.Context, version string, yes bool, in io.Reader) error {
	target, err := cmd.targetVersion(ctx, version)
	if err != nil {
		return err
	}

	if target == cmd.version() {
		cmd.Printf("folio %s is up to date\n", target)
		return nil
	}

	cmd.Printf("Current version: %s\n", cmd.version())
	cmd.Printf("New version:     %s\n", target)

	if !yes && !cmd.confirm(in, "Update folio?") {
		cmd.Println("Update cancelled")
		return nil
	}

	binary, err := update.Binary(ctx, cmd.releases, target, cmd.platform)
	if err != nil {
		return err
	}

	path, err := cmd.executable()
	if err != nil {
		return fmt.Errorf("find installed binary: %w", err)
	}

	if err := update.Replace(path, binary); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}

	cmd.Printf("Updated folio to %s\n", target)

	return nil
}

// targetVersion is the version to update to, as the tag of its release: the one that was asked
// for, or the latest release if none was. A release is tagged with a "v" in front of its version,
// which is easy to leave out when typing one.
func (cmd *Folio) targetVersion(ctx context.Context, version string) (string, error) {
	if version == "" {
		return cmd.releases.Latest(ctx)
	}

	return "v" + strings.TrimPrefix(version, "v"), nil
}

// confirm asks a question on stdout and reports whether the line that in answers with is a yes.
// Anything else is a no, and so is no answer at all, as when folio is run by a script and nobody
// is there to give one: replacing the program is not what to do on a guess.
func (cmd *Folio) confirm(in io.Reader, question string) bool {
	cmd.Printf("%s [y/N] ", question)

	answer, err := bufio.NewReader(in).ReadString('\n')
	if err != nil {
		// Nothing was typed, so the line of the question is still open.
		cmd.Println()
		return false
	}

	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}

// executable returns the path of the file that this program was started from. If folio was started
// through a symbolic link, as one in a directory of the PATH that points to where it is installed,
// it is the file that the link leads to: an update is to replace the program and leave the link.
func executable() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}

	return filepath.EvalSymlinks(path)
}
