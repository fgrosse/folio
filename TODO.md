# TODO

Work still to do on `folio`. Delete an item once it is done rather than ticking it off; this is not
a changelog. Anything that stays true once the work is done belongs elsewhere: decisions in
`CLAUDE.md`, gotchas in a comment at the code they affect.

## Next up

Roughly in the order worth doing.

- **Edit what was entered.** A lot, a grant and a single vest can only be deleted and entered
  again. The vest matters most: a plan that vests differently than the spec laid it out, a share
  more here or a day later there, should be corrected on the row in the Vesting view.
- **Catch up on vests that are pending.** A grant entered with a first vest in the past leaves a
  row to release for every vest since, one dialog each. Releasing all pending vests of a grant in
  one go, with the shares that arrived as a percentage, would make that a single step.
- **Say how old the prices are.** The header shows the prices but not when they are from, which
  matters at a weekend and without a network. The time of the quote is stored already.
- **A key that fetches the quotes now**, rather than waiting for the five minutes to pass.
- **`folio status` should say when stock has no price.** `Account.Unpriced` names it, and the JSON
  output lists it, but the plain output shows values that leave it out without a word.
- **The released vests** have no place in the TUI once they are lots. A key in the Vesting view
  that shows them, dimmed, would make it the whole schedule of a grant.
- **How the account moved today**: the change of the total since the previous close, next to it
  in the header.

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
- **A vesting schedule with a cliff** that is not a whole year, and intervals other than monthly,
  quarterly and yearly, if a grant ever needs them.
