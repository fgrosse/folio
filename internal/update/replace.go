package update

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Replace puts binary in the place of the program at path, which stays as it was unless all of
// that worked.
//
// The new program is written to a file next to the old one and renamed over it. A rename within a
// directory is atomic, so there is a whole folio at path at every moment, the old one or the new,
// even if the update is interrupted halfway. It is next to the old one, rather than in the
// directory for temporary files, because a rename does not reach across file systems. The program
// that runs is not disturbed: it keeps the file it was started from until it exits.
func Replace(path string, binary []byte) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	file, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-update-*")
	if err != nil {
		return err
	}

	err = errors.Join(write(file, binary, info.Mode().Perm()), file.Close())
	if err == nil {
		err = os.Rename(file.Name(), path)
	}
	if err != nil {
		return errors.Join(err, os.Remove(file.Name()))
	}

	return nil
}

// write makes file the program that binary is, for mode to say who may run it, and returns once
// that is on the disk.
func write(file *os.File, binary []byte, mode os.FileMode) error {
	if _, err := file.Write(binary); err != nil {
		return fmt.Errorf("write new binary: %w", err)
	}

	// A temporary file is private to its owner. The new program is for whoever the old one was.
	if err := file.Chmod(mode); err != nil {
		return fmt.Errorf("set permissions: %w", err)
	}

	// The rename can reach the disk before the content does. Without the sync, a crash right
	// after the update could leave an empty file where folio was.
	if err := file.Sync(); err != nil {
		return fmt.Errorf("write new binary: %w", err)
	}

	return nil
}
