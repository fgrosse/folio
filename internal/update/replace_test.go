package update

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReplace covers putting a new binary in the place of the installed one: the file at the path
// is the new program afterwards, it can still be run by whoever could run the old one, and nothing
// else is left in its directory.
func TestReplace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "folio")
	require.NoError(t, os.WriteFile(path, []byte("the old folio"), 0o750)) //nolint:gosec // the permissions are what the test is about

	require.NoError(t, Replace(path, []byte("the new folio")))

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "the new folio", string(content))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o750), info.Mode().Perm())

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "the update must leave nothing next to the binary")
}
