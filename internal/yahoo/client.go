// Package yahoo reads stock quotes from Yahoo Finance.
package yahoo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/shopspring/decimal"

	"github.com/fgrosse/folio/internal/portfolio"
)

// defaultBaseURL is where Yahoo serves the chart endpoint, which needs no API key. It is not an
// official API: it can change or turn requests away without notice.
const defaultBaseURL = "https://query1.finance.yahoo.com"

// userAgent is what the client introduces itself as. Yahoo turns away a request that does not look
// like it comes from a browser.
const userAgent = "Mozilla/5.0"

// A Client reads quotes from Yahoo Finance.
type Client struct {
	http    *http.Client
	baseURL string
}

// New returns a Client that gives up on a request after ten seconds.
func New() *Client {
	return &Client{
		http:    &http.Client{Timeout: 10 * time.Second},
		baseURL: defaultBaseURL,
	}
}

// Quote reads the latest quote of symbol. Its time is that of the trade the price is from, not of
// the request, so a quote read while the market is closed says how old its price is.
func (c *Client) Quote(ctx context.Context, symbol string) (portfolio.Quote, error) {
	endpoint := c.baseURL + "/v8/finance/chart/" + url.PathEscape(symbol) + "?interval=1d&range=1d"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return portfolio.Quote{}, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return portfolio.Quote{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	var body struct {
		Chart struct {
			Result []struct {
				Meta struct {
					Symbol        string          `json:"symbol"`
					Currency      string          `json:"currency"`
					Price         decimal.Decimal `json:"regularMarketPrice"`
					PreviousClose decimal.Decimal `json:"chartPreviousClose"`
					Time          int64           `json:"regularMarketTime"`
				} `json:"meta"`
			} `json:"result"`
		} `json:"chart"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return portfolio.Quote{}, fmt.Errorf("decode response: %w", err)
	}

	meta := body.Chart.Result[0].Meta

	return portfolio.Quote{
		Symbol:        meta.Symbol,
		Price:         meta.Price,
		PreviousClose: meta.PreviousClose,
		Currency:      meta.Currency,
		At:            time.Unix(meta.Time, 0).UTC(),
	}, nil
}
