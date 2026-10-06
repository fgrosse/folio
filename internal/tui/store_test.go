package tui

import (
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"

	"github.com/fgrosse/folio/internal/portfolio"
)

// MockStore is a Store whose every answer a test sets up in advance.
type MockStore struct {
	mock.Mock
}

func (m *MockStore) Lots() ([]portfolio.Lot, error) {
	result := m.Called()
	if x := result.Get(0); x != nil {
		return x.([]portfolio.Lot), result.Error(1)
	}
	return nil, result.Error(1)
}

func (m *MockStore) Grants() ([]portfolio.Grant, error) {
	result := m.Called()
	if x := result.Get(0); x != nil {
		return x.([]portfolio.Grant), result.Error(1)
	}
	return nil, result.Error(1)
}

func (m *MockStore) Quotes() (map[string]portfolio.Quote, error) {
	result := m.Called()
	if x := result.Get(0); x != nil {
		return x.(map[string]portfolio.Quote), result.Error(1)
	}
	return nil, result.Error(1)
}

func (m *MockStore) GetConfig(key string) (string, error) {
	result := m.Called(key)
	return result.String(0), result.Error(1)
}

func (m *MockStore) SetConfig(key, value string) error {
	return m.Called(key, value).Error(0)
}

func (m *MockStore) UnsetConfig(key string) error {
	return m.Called(key).Error(0)
}

func (m *MockStore) SaveLot(lot portfolio.Lot) error {
	return m.Called(lot).Error(0)
}

func (m *MockStore) DeleteLot(id int) error {
	return m.Called(id).Error(0)
}

func (m *MockStore) SaveQuote(quote portfolio.Quote) error {
	return m.Called(quote).Error(0)
}

func (m *MockStore) ReleaseVest(id int, shares, cost decimal.Decimal) error {
	return m.Called(id, shares, cost).Error(0)
}

func (m *MockStore) SaveGrant(grant portfolio.Grant) error {
	return m.Called(grant).Error(0)
}

func (m *MockStore) DeleteGrant(id int) error {
	return m.Called(id).Error(0)
}

func (m *MockStore) Sales() ([]portfolio.Sale, error) {
	result := m.Called()
	if x := result.Get(0); x != nil {
		return x.([]portfolio.Sale), result.Error(1)
	}
	return nil, result.Error(1)
}

func (m *MockStore) SaveSale(sale portfolio.Sale) error {
	return m.Called(sale).Error(0)
}

func (m *MockStore) DeleteSale(id int) error {
	return m.Called(id).Error(0)
}

// testPortfolio is the account most tests of the views look at: PANW stock in two lots of 8.5
// shares in all, and a grant with one vest released into the first of them and two still to come,
// which are taxed at 44.3%. The gain of a sale is taxed at 26.4%.
func testPortfolio() Portfolio {
	return Portfolio{
		Lots: []portfolio.Lot{
			{ID: 1, Symbol: "PANW", Shares: dec("6"), Acquired: day("2026-01-15"), Cost: dec("380.12"), Grant: "Payout"},
			{ID: 2, Symbol: "PANW", Shares: dec("2.5"), Acquired: day("2026-02-15")},
		},
		Grants: []portfolio.Grant{
			{
				ID:     1,
				Name:   "Payout",
				Symbol: "PANW",
				Vests: []portfolio.Vest{
					{ID: 1, Date: day("2026-01-15"), Shares: dec("10"), Released: true},
					{ID: 2, Date: day("2026-11-15"), Shares: dec("10")},
					{ID: 3, Date: day("2026-12-15"), Shares: dec("10")},
				},
			},
		},
		Quotes: map[string]portfolio.Quote{
			"PANW": {Symbol: "PANW", Price: dec("396.25"), PreviousClose: dec("397.31"), Currency: "USD"},
		},
		TaxRate:        decimal.NewNullDecimal(dec("44.3")),
		GainsTaxRate:   decimal.NewNullDecimal(dec("26.4")),
		PotentialBasis: portfolio.Gross,
		CurrentBasis:   portfolio.Gross,
	}
}

// returns sets store up to answer with p whenever the portfolio is loaded.
func (m *MockStore) returns(p Portfolio) {
	m.returnsAccount(p)
	m.returnsRate(portfolio.TaxRateKey, p.TaxRate)
	m.returnsRate(portfolio.GainsTaxRateKey, p.GainsTaxRate)
	m.On("GetConfig", portfolio.PotentialBasisKey).Return(string(p.PotentialBasis), nil)
	m.On("GetConfig", portfolio.CurrentBasisKey).Return(string(p.CurrentBasis), nil)
}

// returnsRate sets store up to answer with rate for the key of a tax rate, written as folio config
// stores it, or with that the key is not set if rate is not valid.
func (m *MockStore) returnsRate(key string, rate decimal.NullDecimal) {
	if !rate.Valid {
		m.On("GetConfig", key).Return("", portfolio.ErrNotSet)
		return
	}

	m.On("GetConfig", key).Return(rate.Decimal.String()+"%", nil)
}

// returnsAccount sets store up to answer with the lots, grants, quotes and sales of p, and leaves
// the configuration to the test.
func (m *MockStore) returnsAccount(p Portfolio) {
	m.On("Lots").Return(p.Lots, nil)
	m.On("Grants").Return(p.Grants, nil)
	m.On("Quotes").Return(p.Quotes, nil)
	m.On("Sales").Return(p.Sales, nil)
}
