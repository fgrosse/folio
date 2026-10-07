package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fgrosse/folio/internal/update"
)

// TestSelfUpdateCmd covers moving folio to the latest release in one go: "folio self-update -y"
// says which version it has and which it found, and puts the binary of that release in the place
// of the one that runs. It has no use for the account, so it does not create the database.
func TestSelfUpdateCmd(t *testing.T) {
	cmd, dbPath := NewTestingCmd(t, "self-update", "-y")
	installed := NewTestingInstall(t, cmd, "1.1.0")
	cmd.releases = fakeReleases{
		latest:   "v1.2.0",
		releases: map[string][]byte{"v1.2.0": []byte("folio v1.2.0")},
	}

	var out bytes.Buffer
	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())

	expected := `
Current version: v1.1.0
New version:     v1.2.0
Updated folio to v1.2.0
`
	assert.Equal(t, expected[1:], out.String())
	assert.Equal(t, "folio v1.2.0", readFile(t, installed))
	assert.NoFileExists(t, dbPath, "an update must not touch the database of the account")
}

// TestSelfUpdateCmd_Confirm covers the question that an update asks before it replaces anything:
// it goes ahead on a yes, and leaves the binary alone on anything else, which includes an answer
// that never comes, as when nobody is there to give it.
func TestSelfUpdateCmd_Confirm(t *testing.T) {
	cases := map[string]struct {
		answer    string
		installed string
		outcome   string
	}{
		"y":          {answer: "y\n", installed: "folio v1.2.0", outcome: "Updated folio to v1.2.0\n"},
		"yes":        {answer: "Yes\n", installed: "folio v1.2.0", outcome: "Updated folio to v1.2.0\n"},
		"no":         {answer: "n\n", installed: "folio v1.1.0", outcome: "Update canceled\n"},
		"just enter": {answer: "\n", installed: "folio v1.1.0", outcome: "Update canceled\n"},
		"no answer":  {answer: "", installed: "folio v1.1.0", outcome: "\nUpdate canceled\n"},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			cmd, _ := NewTestingCmd(t, "self-update")
			installed := NewTestingInstall(t, cmd, "1.1.0")
			cmd.releases = fakeReleases{
				latest:   "v1.2.0",
				releases: map[string][]byte{"v1.2.0": []byte("folio v1.2.0")},
			}

			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetIn(strings.NewReader(c.answer))

			require.NoError(t, cmd.Execute())

			expected := `
Current version: v1.1.0
New version:     v1.2.0
Update folio? [y/N] `
			assert.Equal(t, expected[1:]+c.outcome, out.String())
			assert.Equal(t, c.installed, readFile(t, installed))
		})
	}
}

// TestSelfUpdateCmd_UpToDate covers a folio that is the latest release already: the update says
// so and is done, without a question to answer and without downloading what is installed.
func TestSelfUpdateCmd_UpToDate(t *testing.T) {
	cmd, _ := NewTestingCmd(t, "self-update")
	installed := NewTestingInstall(t, cmd, "1.2.0")
	// The release has no files, so that a download of it would fail.
	cmd.releases = fakeReleases{latest: "v1.2.0"}

	var out bytes.Buffer
	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())

	assert.Equal(t, "folio v1.2.0 is up to date\n", out.String())
	assert.Equal(t, "folio v1.2.0", readFile(t, installed))
}

// TestSelfUpdateCmd_Version covers moving folio to a release that is named, which need not be the
// latest and may be older than the one installed: the way back from a release that broke something.
// The version is the tag of the release, and is taken without its "v" as well.
func TestSelfUpdateCmd_Version(t *testing.T) {
	for _, version := range []string{"v1.0.0", "1.0.0"} {
		t.Run(version, func(t *testing.T) {
			cmd, _ := NewTestingCmd(t, "self-update", "-y", version)
			installed := NewTestingInstall(t, cmd, "1.1.0")
			cmd.releases = fakeReleases{
				latest: "v1.2.0",
				releases: map[string][]byte{
					"v1.0.0": []byte("folio v1.0.0"),
					"v1.2.0": []byte("folio v1.2.0"),
				},
			}

			var out bytes.Buffer
			cmd.SetOut(&out)

			require.NoError(t, cmd.Execute())

			expected := `
Current version: v1.1.0
New version:     v1.0.0
Updated folio to v1.0.0
`
			assert.Equal(t, expected[1:], out.String())
			assert.Equal(t, "folio v1.0.0", readFile(t, installed))
		})
	}
}

// TestSelfUpdateCmd_NotARelease covers the builds that no release made, which are not to be
// replaced with one: a folio that "go install" built is updated by "go install", and one built from
// a checkout by building it again. The update leaves both alone and says how they are updated.
func TestSelfUpdateCmd_NotARelease(t *testing.T) {
	cases := map[string]struct {
		args  []string
		info  *debug.BuildInfo
		error string
	}{
		"installed with go install": {
			info:  &debug.BuildInfo{Main: debug.Module{Version: "v1.1.0"}},
			error: "this folio was installed with \"go install\", so update it that way: go install github.com/fgrosse/folio/cmd/folio@latest",
		},
		"installed with go install, to a version": {
			args:  []string{"1.2.0"},
			info:  &debug.BuildInfo{Main: debug.Module{Version: "v1.1.0"}},
			error: "this folio was installed with \"go install\", so update it that way: go install github.com/fgrosse/folio/cmd/folio@v1.2.0",
		},
		"built from a checkout": {
			info:  &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}},
			error: "this folio was built from its source and not released, so update the source and build it again",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			cmd, _ := NewTestingCmd(t, append([]string{"self-update", "-y"}, c.args...)...)
			installed := NewTestingInstall(t, cmd, "1.1.0")
			cmd.BuildVersion = "" // which only the build of a release sets
			cmd.buildInfo = func() (*debug.BuildInfo, bool) { return c.info, true }
			cmd.releases = fakeReleases{
				latest:   "v1.2.0",
				releases: map[string][]byte{"v1.2.0": []byte("folio v1.2.0")},
			}

			require.EqualError(t, cmd.Execute(), c.error)

			assert.Equal(t, "folio v1.1.0", readFile(t, installed))
		})
	}
}

// TestSelfUpdateCmd_Windows covers the platform where the update does not work yet: Windows does
// not let the file of a program that runs be replaced, and its releases are zip files. The update
// says so before it asks or downloads anything, and points to where the releases are.
func TestSelfUpdateCmd_Windows(t *testing.T) {
	cmd, _ := NewTestingCmd(t, "self-update", "-y")
	installed := NewTestingInstall(t, cmd, "1.1.0")
	cmd.platform = update.Platform{OS: "windows", Arch: "amd64"}
	cmd.releases = fakeReleases{latest: "v1.2.0"}

	var out bytes.Buffer
	cmd.SetOut(&out)

	require.EqualError(t, cmd.Execute(), "folio cannot update itself on Windows yet, download the release from https://github.com/fgrosse/folio/releases")

	assert.Empty(t, out.String())
	assert.Equal(t, "folio v1.1.0", readFile(t, installed))
}

// NewTestingInstall makes cmd the folio of a release of version that is installed as a file in a
// temporary directory, on Linux, and returns the path of that file. Its content is no program but
// says which version it is, as "folio v1.1.0", for a test to see whether it was replaced.
func NewTestingInstall(t *testing.T, cmd *Folio, version string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "folio")
	require.NoError(t, os.WriteFile(path, []byte("folio v"+version), 0o755)) //nolint:gosec // a program is a file that others may run

	cmd.BuildVersion = version
	cmd.executable = func() (string, error) { return path, nil }
	cmd.platform = update.Platform{OS: "linux", Arch: "amd64"}

	return path
}

// fakeReleases is an update.Source that serves releases from memory, for Linux on amd64.
type fakeReleases struct {
	latest   string            // the version of the latest release
	releases map[string][]byte // the folio binary of each release, by its version
}

func (r fakeReleases) Latest(context.Context) (string, error) {
	return r.latest, nil
}

func (r fakeReleases) Asset(_ context.Context, version, name string) ([]byte, error) {
	binary, ok := r.releases[version]
	if !ok {
		return nil, fmt.Errorf("no %s of release %s: %w", name, version, fs.ErrNotExist)
	}

	archiveName := "folio-" + version + "-linux-amd64.tar.gz"
	archive := archiveOf(binary)

	switch name {
	case archiveName:
		return archive, nil
	case "checksums.txt":
		sum := sha256.Sum256(archive)
		return []byte(hex.EncodeToString(sum[:]) + "  " + archiveName + "\n"), nil
	default:
		return nil, fmt.Errorf("no %s of release %s: %w", name, version, fs.ErrNotExist)
	}
}

// archiveOf returns the archive of a release, a gzipped tarball, with binary in it as folio.
func archiveOf(binary []byte) []byte {
	var archive bytes.Buffer
	zipped := gzip.NewWriter(&archive)
	tarball := tar.NewWriter(zipped)

	header := &tar.Header{Name: "folio", Mode: 0o755, Size: int64(len(binary))}
	if err := tarball.WriteHeader(header); err != nil {
		panic(err)
	}
	if _, err := tarball.Write(binary); err != nil {
		panic(err)
	}
	if err := tarball.Close(); err != nil {
		panic(err)
	}
	if err := zipped.Close(); err != nil {
		panic(err)
	}

	return archive.Bytes()
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	require.NoError(t, err)

	return string(content)
}
