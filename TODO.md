# TODO

Work still to do on `folio`. Delete an item once it is done rather than ticking it off; this is not
a changelog. Anything that stays true once the work is done belongs elsewhere: decisions in
`CLAUDE.md`, gotchas in a comment at the code they affect.

## Next up

Roughly in the order worth doing.

- **Lots**: the shares that are held, in the store and as the Holdings view.
- **Quotes**: the price of a symbol, fetched from behind an interface and cached in the database.
- **Grants and their vests**: an award of shares and the days on which they vest, as the Vesting
  view, with releasing a vest into a lot.
- **The account values**: current, potential and total, in the header of every view and from
  `folio status`, with `--json` for a status bar widget.

## Later

- **Show values in EUR**, toggled with a key in the TUI and a flag on `folio status`. It needs an
  exchange rate, fetched and cached like a quote.
- **Tax estimates**: what is left of a vest after tax, from a rate that is configured, and the
  potential value after tax next to the one before.
- **Stock bought privately** as a first-class thing: a lot can be entered by hand already, but it
  has no cost, so there is no gain or loss to show.
- **Sales**: selling shares takes them out of a lot, and what they sold for is worth keeping.
- **The account value over time**, as a chart or a table by month.
- **The Omarchy bar widget of `~/src/stock-ticker`** could show the account value from
  `folio status --json` next to the price.
