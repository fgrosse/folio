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
	assert.Equal(t, "4359.09", account.Current.String())
}

// TestNewAccount_Potential covers the second value: what the shares still to come are worth, which
// is every vest that has not been released at the latest price of its grant's stock. That takes in
// a vest whose day has passed but which is still pending release, as the bank counts it. A released
// vest is a lot by now, and counts there.
func TestNewAccount_Potential(t *testing.T) {
	grants := []Grant{
		{
			Name:   "Payout",
			Symbol: "PANW",
			Vests: []Vest{
				{Date: day("2026-01-15"), Shares: shares("10"), Released: true},
				{Date: day("2026-02-15"), Shares: shares("10")},
				{Date: day("2026-03-15"), Shares: shares("10")},
			},
		},
		{
			Name:   "Old plan",
			Symbol: "AAPL",
			Vests:  []Vest{{Date: day("2027-01-01"), Shares: shares("1.5")}},
		},
	}
	quotes := map[string]Quote{
		"PANW": {Symbol: "PANW", Price: shares("396.25")},
		"AAPL": {Symbol: "AAPL", Price: shares("330.32")},
	}

	account := NewAccount(nil, grants, quotes)

	// 20 × 396.25 + 1.5 × 330.32
	assert.Equal(t, "8420.48", account.Potential.String())
	assert.Equal(t, "0", account.Current.String())
}

// TestAccount_Total covers the third value, which is the other two added up.
func TestAccount_Total(t *testing.T) {
	account := Account{Current: shares("4359.09"), Potential: shares("8420.48")}

	assert.Equal(t, "12779.57", account.Total().String())
}

// TestNewAccount_Unpriced covers stock there is no quote of, such as right after a lot of a new
// symbol was entered or when a symbol was mistyped. It is worth nothing in the values, since there
// is nothing to value it at, and the account names the symbol so that a view can say the values are
// incomplete rather than pass them off as the whole account.
func TestNewAccount_Unpriced(t *testing.T) {
	lots := []Lot{
		{Symbol: "PANW", Shares: shares("2"), Acquired: day("2026-01-15")},
		{Symbol: "SAP.DE", Shares: shares("5"), Acquired: day("2026-01-15")},
	}
	grants := []Grant{
		{Name: "Plan", Symbol: "MSFT", Vests: []Vest{{Date: day("2027-01-01"), Shares: shares("1")}}},
		{Name: "More", Symbol: "MSFT", Vests: []Vest{{Date: day("2027-01-01"), Shares: shares("1")}}},
	}
	quotes := map[string]Quote{
		"PANW": {Symbol: "PANW", Price: shares("396.25")},
	}

	account := NewAccount(lots, grants, quotes)

	assert.Equal(t, "792.5", account.Current.String())
	assert.Equal(t, "0", account.Potential.String())
	assert.Equal(t, []string{"MSFT", "SAP.DE"}, account.Unpriced)
}

// TestSymbols covers which stock an account needs quotes of: that of every lot and every grant, each
// symbol once and in alphabetical order, so that the same account always asks for the same list.
func TestSymbols(t *testing.T) {
	lots := []Lot{
		{Symbol: "PANW", Shares: shares("2")},
		{Symbol: "AAPL", Shares: shares("5")},
		{Symbol: "PANW", Shares: shares("1")},
	}
	grants := []Grant{
		{Name: "Plan", Symbol: "MSFT"},
		{Name: "Payout", Symbol: "PANW"},
	}

	assert.Equal(t, []string{"AAPL", "MSFT", "PANW"}, Symbols(lots, grants))
	assert.Empty(t, Symbols(nil, nil))
}
