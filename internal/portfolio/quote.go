package portfolio

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

// A Quote is what one share of a stock was worth at some point in time.
type Quote struct {
	Symbol string
	Price  decimal.Decimal

	// PreviousClose is the price at the end of the trading day before, which is what the change of
	// the day is measured against.
	PreviousClose decimal.Decimal

	// Currency is the ISO code of the currency both prices are in, such as USD.
	Currency string

	// At is when the price was read.
	At time.Time
}

// A Quoter looks up what a share of a stock is worth right now, such as a client of a service that
// serves quotes.
type Quoter interface {
	Quote(ctx context.Context, symbol string) (Quote, error)
}

// FetchQuotes asks quoter for the quote of each of symbols and returns them by symbol. A symbol
// that has no quote does not cost the others theirs: the quotes that could be read come back
// together with an error that joins the ones of those that could not.
func FetchQuotes(ctx context.Context, quoter Quoter, symbols []string) (map[string]Quote, error) {
	quotes := make(map[string]Quote, len(symbols))

	var errs []error
	for _, symbol := range symbols {
		quote, err := quoter.Quote(ctx, symbol)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		quotes[symbol] = quote
	}

	return quotes, errors.Join(errs...)
}
