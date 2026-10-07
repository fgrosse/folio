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
		"no":         {answer: "n\n", installed: "folio v1.1.0", outcome: "Update cancelled\n"},
		"just enter": {answer: "\n", installed: "folio v1.1.0", outcome: "Update cancelled\n"},
		"no answer":  {answer: "", installed: "folio v1.1.0", outcome: "\nUpdate cancelled\n"},
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

// NewTestingInstall makes cmd the folio of a release of version that is installed as a file in a
// temporary directory, on Linux, and returns the path of that file. Its content is no program but
// says which version it is, as "folio v1.1.0", for a test to see whether it was replaced.
func NewTestingInstall(t *testing.T, cmd *Folio, version string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "folio")
	require.NoError(t, os.WriteFile(path, []byte("folio v"+version), 0o755))

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
