package portfolio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewAccount_Current covers the first of the account's three values: what the shares that are
// held would sell for, which is every lot's shares at the latest price of its stock, to the cent.
func TestNewAccount_Current(t *testing.T) {
	lots := []Lot{
		{Symbol: "PANW", Shares: shares("6"), Acquired: day("2026-01-15")},
		{Symbol: "PANW", Shares: shares("2.5"), Acquired: day("2026-02-15")},
		{Symbol: "AAPL", Shares: shares("3"), Acquired: day("2025-11-02")},
	}
	quotes := map[string]Quote{
		"PANW": {Symbol: "PANW", Price: shares("396.25")},
		"AAPL": {Symbol: "AAPL", Price: shares("330.32")},
	}

	account := NewAccount(lots, nil, quotes)

	// 8.5 × 396.25 + 3 × 330.32
	assert.Equal(t, shares("4359.09"), account.Current)
}
