package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFolioMain covers what folio exits with: 0 once a verb has done what it was asked, and 1 when it
// could not, with the reason on stderr where a script that reads stdout does not take it for output.
func TestFolioMain(t *testing.T) {
	cmd, _ := NewTestingCmd(t, "config", "tax-rate", "44.3%")
	assert.Equal(t, 0, cmd.Main())

	cmd, _ = NewTestingCmd(t, "config", "tax-rate", "lots")
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	assert.Equal(t, 1, cmd.Main())
	assert.Empty(t, stdout.String())
	assert.Equal(t, "Error: \"lots\" is not a tax rate such as 44.3%\n", stderr.String())
}
