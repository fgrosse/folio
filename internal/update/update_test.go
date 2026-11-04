package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBinary covers fetching the folio of a release: the archive of the platform is picked by its
// name, checked against the checksums of the release, and the binary is taken out of it, past the
// files that the archive carries next to it.
func TestBinary(t *testing.T) {
	archive := newArchive(t, map[string]string{
		"README.md": "# folio",
		"folio":     "the new folio",
	})
	source := fakeSource{
		"v1.2.0/folio-v1.2.0-linux-amd64.tar.gz": archive,
		"v1.2.0/folio-v1.2.0-darwin-arm64.tar.gz": newArchive(t, map[string]string{
			"folio": "the new folio for another platform",
		}),
		"v1.2.0/checksums.txt": []byte(
			"0000000000000000000000000000000000000000000000000000000000000000  folio-v1.2.0-darwin-arm64.tar.gz\n" +
				checksum(archive) + "  folio-v1.2.0-linux-amd64.tar.gz\n",
		),
	}

	binary, err := Binary(t.Context(), source, "v1.2.0", Platform{OS: "linux", Arch: "amd64"})
	require.NoError(t, err)

	assert.Equal(t, "the new folio", string(binary))
}

// TestBinary_Unverified covers the archives that the checksums of the release do not vouch for. An
// update puts what it downloaded in the place of the program that runs, so it takes nothing that
// it could not check: not an archive that differs from the one that was released, and not one
// that the release has no checksum of.
func TestBinary_Unverified(t *testing.T) {
	archive := newArchive(t, map[string]string{"folio": "the new folio"})
	archiveName := "folio-v1.2.0-linux-amd64.tar.gz"
	released := checksum([]byte("the archive that was released"))

	cases := map[string]struct {
		checksums string // what checksums.txt says, and no such file if it is empty
		error     string
	}{
		"another archive than was released": {
			checksums: released + "  " + archiveName + "\n",
			error:     "verify " + archiveName + ": its checksum is " + checksum(archive) + ", and the release states " + released,
		},
		"an archive without a checksum": {
			checksums: checksum(archive) + "  folio-v1.2.0-darwin-arm64.tar.gz\n",
			error:     "verify " + archiveName + ": checksums.txt has no checksum of it",
		},
		"a release without checksums": {
			error: "download checksums: checksums.txt of v1.2.0: file does not exist",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			source := fakeSource{"v1.2.0/" + archiveName: archive}
			if c.checksums != "" {
				source["v1.2.0/checksums.txt"] = []byte(c.checksums)
			}

			binary, err := Binary(t.Context(), source, "v1.2.0", Platform{OS: "linux", Arch: "amd64"})

			require.EqualError(t, err, c.error)
			assert.Nil(t, binary)
		})
	}
}

// fakeSource is a Source that serves the assets it holds, each under the version of its release
// and its name, as "v1.2.0/checksums.txt".
type fakeSource map[string][]byte

func (s fakeSource) Latest(context.Context) (string, error) {
	return "", fmt.Errorf("latest release: %w", fs.ErrNotExist)
}

func (s fakeSource) Asset(_ context.Context, version, name string) ([]byte, error) {
	asset, ok := s[version+"/"+name]
	if !ok {
		return nil, fmt.Errorf("%s of %s: %w", name, version, fs.ErrNotExist)
	}

	return asset, nil
}

// newArchive returns a release archive, a gzipped tarball, that holds files by their names.
func newArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var archive bytes.Buffer
	zipped := gzip.NewWriter(&archive)
	tarball := tar.NewWriter(zipped)

	for name, content := range files {
		header := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(content))}
		require.NoError(t, tarball.WriteHeader(header))
		_, err := tarball.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, tarball.Close())
	require.NoError(t, zipped.Close())

	return archive.Bytes()
}

// checksum is the SHA-256 of data the way checksums.txt writes it.
func checksum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
