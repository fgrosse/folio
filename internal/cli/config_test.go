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
// that is set, and says that there is none in an account that has none.
func TestConfigCmd_Get(t *testing.T) {
	cmd, dbPath := NewTestingCmd(t, "config", "tax-rate")
	var out bytes.Buffer
	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "No tax rate is set.\n", out.String())

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
