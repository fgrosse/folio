// Package update moves an installed folio to another release of it: it fetches the binary of a
// release from wherever the releases are published and puts it in the place of the one that runs.
package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

// A Source is where the releases of folio are published. It is an interface for the reason
// portfolio.Quoter is one: the one there is talks to GitHub, and a test must not.
type Source interface {
	// Latest returns the version of the newest release, with its "v", such as "v1.2.0".
	Latest(ctx context.Context) (string, error)

	// Asset returns one of the files of the release of version, by its name.
	Asset(ctx context.Context, version, name string) ([]byte, error)
}

// A Platform is what a binary is built for, in the words of GOOS and GOARCH. A release has an
// archive for each one.
type Platform struct {
	OS   string
	Arch string
}

// binaryName is what the folio binary is called inside the archive of a release.
const binaryName = "folio"

// checksumsName is the file of a release that lists the SHA-256 of each of its archives.
const checksumsName = "checksums.txt"

// Binary fetches the folio binary that the release of version has for a platform.
func Binary(ctx context.Context, source Source, version string, platform Platform) ([]byte, error) {
	// The name is the one that the archives section of .goreleaser.yaml gives an archive. It
	// has the version in it, which is why an update has to know the version before it can fetch
	// anything.
	name := fmt.Sprintf("folio-%s-%s-%s.tar.gz", version, platform.OS, platform.Arch)

	archive, err := source.Asset(ctx, version, name)
	if err != nil {
		return nil, fmt.Errorf("download release: %w", err)
	}

	checksums, err := source.Asset(ctx, version, checksumsName)
	if err != nil {
		return nil, fmt.Errorf("download checksums: %w", err)
	}

	if err := verify(name, archive, checksums); err != nil {
		return nil, fmt.Errorf("verify %s: %w", name, err)
	}

	binary, err := extract(archive)
	if err != nil {
		return nil, fmt.Errorf("unpack %s: %w", name, err)
	}

	return binary, nil
}

// verify checks that an archive is the one that was released under its name: that its SHA-256 is
// the one that the checksums of the release state for it. They come from the same place as the
// archive, so this catches a download that went wrong or was cut short, and a file that was put in
// the place of an archive later on. It is no defense against a release that was forged as a whole.
func verify(name string, archive, checksums []byte) error {
	sum := sha256.Sum256(archive)
	actual := hex.EncodeToString(sum[:])

	// A line of the file is what sha256sum prints: the checksum and the name of its file.
	for line := range strings.Lines(string(checksums)) {
		fields := strings.Fields(line)
		if len(fields) != 2 || fields[1] != name {
			continue
		}

		if fields[0] != actual {
			return fmt.Errorf("its checksum is %s, and the release states %s", actual, fields[0])
		}

		return nil
	}

	return fmt.Errorf("%s has no checksum of it", checksumsName)
}

// extract returns the folio binary out of the gzipped tarball that is the archive of a release.
func extract(archive []byte) ([]byte, error) {
	unzipped, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}

	tarball := tar.NewReader(unzipped)
	for {
		header, err := tarball.Next()
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("it has no %s in it", binaryName)
		}
		if err != nil {
			return nil, err
		}

		if header.Name == binaryName {
			return io.ReadAll(tarball)
		}
	}
}
