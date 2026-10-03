package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fgrosse/folio/internal/portfolio"
)

// TestConfigCmd_Set covers setting a value of the configuration, as git config does: "folio config
// tax-rate" with a percentage stores the rate that vests are taxed at for the whole account.
func TestConfigCmd_Set(t *testing.T) {
	cmd, dbPath := NewTestingCmd(t, "config", "tax-rate", "44.3%")
	require.NoError(t, cmd.Execute())

	store, err := portfolio.NewStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	rate, err := store.TaxRate()
	require.NoError(t, err)
	require.True(t, rate.Valid)
	assert.Equal(t, "44.3", rate.Decimal.String())
}

// TestConfigCmd_Get covers asking for a value: "folio config tax-rate" without one prints the rate
// that is set. A key that is not set prints nothing and exits 1, as git config does, so that a
// script can tell it apart from a value by the exit code alone.
func TestConfigCmd_Get(t *testing.T) {
	cmd, dbPath := NewTestingCmd(t, "config", "tax-rate")
	var out, stderr bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&stderr)

	assert.Equal(t, 1, cmd.Main())
	assert.Empty(t, out.String())
	assert.Empty(t, stderr.String())

	set := New()
	set.SetArgs([]string{"--db", dbPath, "config", "tax-rate", "44.3"})
	require.NoError(t, set.Execute())

	out.Reset()
	cmd = New()
	cmd.SetArgs([]string{"--db", dbPath, "config", "tax-rate"})
	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "44.3%\n", out.String())
}

// TestConfigCmd_UnknownKey covers a key that the configuration does not have, whose value can be
// neither set nor read: folio config says which keys there are.
func TestConfigCmd_UnknownKey(t *testing.T) {
	cmd, _ := NewTestingCmd(t, "config", "tax", "44.3%")
	assert.EqualError(t, cmd.Execute(), `"tax" is no key of the configuration: use tax-rate`)

	cmd, _ = NewTestingCmd(t, "config", "tax")
	assert.EqualError(t, cmd.Execute(), `"tax" is no key of the configuration: use tax-rate`)
}

// TestConfigCmd_List covers "folio config" without a key, which prints the whole configuration as
// YAML: every key that is set, by its name, with the value it would print on its own. An account
// that has none set has an empty configuration.
func TestConfigCmd_List(t *testing.T) {
	cmd, dbPath := NewTestingCmd(t, "config")
	var out bytes.Buffer
	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "{}\n", out.String())

	set := New()
	set.SetArgs([]string{"--db", dbPath, "config", "tax-rate", "44.3"})
	require.NoError(t, set.Execute())

	out.Reset()
	cmd = New()
	cmd.SetArgs([]string{"--db", dbPath, "config"})
	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "tax-rate: 44.3%\n", out.String())
}

// TestConfigCmd_ListJSON covers "folio config --output=json", which prints the whole configuration
// as a JSON object for a script to read, with the same keys and values as the YAML. As in kubectl,
// -o is short for --output.
func TestConfigCmd_ListJSON(t *testing.T) {
	cmd, dbPath := NewTestingCmd(t, "config", "--output=json")
	var out bytes.Buffer
	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "{}\n", out.String())

	set := New()
	set.SetArgs([]string{"--db", dbPath, "config", "tax-rate", "44.3"})
	require.NoError(t, set.Execute())

	out.Reset()
	cmd = New()
	cmd.SetArgs([]string{"--db", dbPath, "config", "-o", "json"})
	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())
	assert.Equal(t, `{"tax-rate":"44.3%"}`+"\n", out.String())
}

// TestConfigCmd_OutputWithKey covers --output with a key, which is refused rather than ignored: it
// is the whole configuration that is printed in a format, and a single value prints as itself.
func TestConfigCmd_OutputWithKey(t *testing.T) {
	cmd, _ := NewTestingCmd(t, "config", "--output=json", "tax-rate")
	assert.EqualError(t, cmd.Execute(), "--output is for the whole configuration: leave out the key")
}
