package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fgrosse/folio/internal/portfolio"
)

// TestTaxRateCmd covers setting the rate that vests are taxed at: "folio tax-rate" takes it as a
// percentage and stores it for the whole account.
func TestTaxRateCmd(t *testing.T) {
	cmd, dbPath := NewTestingCmd(t, "tax-rate", "44.3%")
	require.NoError(t, cmd.Execute())

	store, err := portfolio.NewStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	rate, err := store.TaxRate()
	require.NoError(t, err)
	require.True(t, rate.Valid)
	assert.Equal(t, "44.3", rate.Decimal.String())
}

// TestTaxRateCmd_Print covers asking for the rate: "folio tax-rate" without one prints the rate
// that is set, and says that there is none in an account that has none.
func TestTaxRateCmd_Print(t *testing.T) {
	cmd, dbPath := NewTestingCmd(t, "tax-rate")
	var out bytes.Buffer
	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "No tax rate is set.\n", out.String())

	set := New()
	set.SetArgs([]string{"--db", dbPath, "tax-rate", "44.3"})
	require.NoError(t, set.Execute())

	out.Reset()
	cmd = New()
	cmd.SetArgs([]string{"--db", dbPath, "tax-rate"})
	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "44.3%\n", out.String())
}
