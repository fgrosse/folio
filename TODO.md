# TODO

Work still to do on `folio`. Delete an item once it is done rather than ticking it off; this is not
a changelog. Anything that stays true once the work is done belongs elsewhere: decisions in
`CLAUDE.md`, gotchas in a comment at the code they affect.

## Next up

Roughly in the order worth doing.

- **Edit a grant and its vests.** A lot can be edited, but a grant and a single vest can only be
  deleted and entered again. The vest matters most: a plan that vests differently than it was
  entered, a share more here or a day later there, should be corrected on the row in the Vesting
  view.
- **Check an edited lot against its vest.** A lot that a vest was released into can be edited to
  hold more shares than vested, or another symbol than its grant's, which a release would refuse.
- **Catch up on vests that are pending.** A grant entered with a first vest in the past leaves a
  row to release for every vest since, one dialog each. Releasing all pending vests of a grant in
  one go, with the shares that arrived as a percentage, would make that a single step.
- **Finish the sales.** A sale cannot be edited, only deleted and entered again. It has no fee, so
  its proceeds are before costs, and it is in USD like everything else, while the money that
  arrives may be in another currency. There is no verb to record a sale from the command line. A sale that spans
  several lots is entered once per lot, and folio could spread it over the lots oldest first, the
  way the tax office counts.
- **Record the whole of a release.** A release confirmation states more than folio keeps: the
  shares that were sold to cover tax, which folio can work out, and the price they sold at, which
  it cannot. That sale is a taxable event of its own, with a gain or loss against the value of the
  shares on the day of the vest. With sales in place, that one could be a sale like any other, of
  shares that were never held. The confirmation itself, a PDF, could be attached to the release and
  kept in the database, so that every number has its source next to it.
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

- **Show values in another currency**, such as EUR, toggled with a key in the TUI and a flag on `folio status`. It needs an
  exchange rate, fetched and cached like a quote.
- **Tax estimates**: the Vesting view shows each vest after tax at the rate of the account, and the
  header shows the potential value after tax with `folio config potential net`. Still to come:
  the potential value after tax in `folio status`, a key in the TUI that sets the rate, which only
  the command line does now, and one that switches the header between gross and net.
  For what is held, the gain of a lot since its cost, which is recorded, and the tax on it if it
  was sold. Costs are in USD, and a tax return outside the US wants them in its own currency at
  the rate of the day, so this needs the exchange rate of a past day.
- **The gain of a lot** in the Holdings view, now that it has a cost: per lot and in the header.
- **The account value over time**, as a chart or a table by month.
- **A status bar widget**, such as a module for Waybar, that shows the account value from
  `folio status --json`.
- **A vesting schedule with a cliff** that is not a whole year, and intervals other than monthly,
  quarterly and yearly, if a grant ever needs them.
