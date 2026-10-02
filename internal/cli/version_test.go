package cli

import (
	"bytes"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVersionCmd covers asking folio which release it is: "folio version" prints the version it was
// built as and the platform it was built for, on one line, the way "go version" does. It has no
// use for the account, so it does not create the database.
func TestVersionCmd(t *testing.T) {
	cmd, dbPath := NewTestingCmd(t, "version")
	cmd.BuildVersion = "1.0.0"

	var out bytes.Buffer
	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())

	assert.Equal(t, "folio version v1.0.0 "+runtime.GOOS+"/"+runtime.GOARCH+"\n", out.String())
	assert.NoFileExists(t, dbPath, "the version must not touch the database of the account")
}
