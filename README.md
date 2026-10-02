# folio

A terminal-first tracker for the stock you hold and the stock that is still to vest. It answers
what the account at your broker's web site answers, without logging in there:

- **Current**: what the shares you hold would sell for.
- **Potential**: what the shares that are still to vest, or vested and not released yet, are worth.
- **Total**: the two added up.

It is built for stock that comes from work as RSUs, on a vesting schedule. Everything is kept in a
local SQLite database, and prices come from Yahoo Finance.

```
  8.5 PANW                                                                       Total: $11,293.13
  PANW $396.25 ▼ 0.3%                                      Current $3,368.13 · Potential $7,925.00
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Acquired    Symbol    From                        Shares        Cost       Price           Value │
│──────────────────────────────────────────────────────────────────────────────────────────────────│
│ 2026-01-15  PANW      Payout                           6     $380.12     $396.25       $2,377.50 │
│ 2026-02-15  PANW                                     2.5           -     $396.25         $990.63 │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
↑/k up • ↓/j down • q quit
a add • d delete
1 Holdings • 2 Vesting • 3 Grants
```

## Install

Requires Go and a C compiler (for SQLite).

```bash
git clone <this repo> ~/src/folio
cd ~/src/folio
go install ./...
```

The database lives at `$XDG_DATA_HOME/folio/folio.db` (`~/.local/share/folio/folio.db`). Use `--db`
or `FOLIO_DB` for another one.

## How it works

An account is made of four things:

- A **lot** is shares that you hold: a number of shares of one stock, the day they arrived and
  what one of them cost that day. Lots are what the current value counts.
- A **grant** is an award of shares that vest over time, such as a grant of RSUs.
- A **vest** is one day on which shares of a grant vest. Vests are what the potential value counts,
  until each is **released** into a lot - with the number of shares that actually arrived, which
  is fewer than vested when some were withheld for tax, and what a share was worth that day.

- A **sale** is shares of one lot that were sold on one day, at one price. The lot keeps what it
  was acquired with, and what is left of it is that less its sales. What the sales brought in is
  the money that was realized, which is stated apart from the three values.

## The TUI

A bare `folio` opens it. The header of every view has the three values, and the prices they were
worked out from. `1`-`4` or `tab` switch views, `q` quits.

| View | Shows | Keys |
|---|---|---|
| Holdings | Every lot that has shares left, and what they are worth | `a` add a lot, `e` edit, `s` sell shares of it, `d` delete |
| Vesting | Every vest that has not been released, in the order of their days | `r` release a vest that is due |
| Grants | Every grant, with the shares still to come | `a` add a grant, `d` delete |
| Sales | Every sale, with what it brought in and gained | `d` delete |

A lot and a grant are typed as one line:

```
12.5 PANW 2026-03-15 @380.12                               a lot: shares, day and cost of a share
Payout: 10 PANW monthly x24 from 2026-01-15                10 shares each month, 24 times
RSU 2025: 400 PANW quarterly 10/20/30/40 from 2026-02-20   400 shares over four years
```

In a lot, the day defaults to today and the cost may be left out. Releasing a vest asks for the
shares that arrived and their cost the same way: `6 @380.12`.

Selling opens a form with a field each for the shares, the price, the day and notes of several
lines. `tab` moves between the fields, `enter` saves, and in the notes, where `enter` starts a new
line, `ctrl+s` does. The note of a sale shows above the Sales table while the sale is selected.

In a grant, the interval is `monthly`, `quarterly` or `yearly`, and the day after `from` is that
of the first vest. With percentages, the shares are those of the whole grant, and each percentage
is the part of them that vests in one year, spread evenly over the vests of that year in whole
shares.

Prices are fetched when the TUI starts and every five minutes after that. It opens with the last
prices it saw, so it works without a network too.

## The command line

```bash
folio lot 12.5 PANW 2026-03-15 @380.12
folio grant "Payout: 10 PANW monthly x24 from 2026-01-15"
folio grant --vests schedule.txt "Payout: PANW"    # the vests listed, "<YYYY-MM-DD> <shares>" a line
folio release 2026-01-15 6 @380.12                 # 6 shares arrived, worth $380.12 each that day
folio status
folio status --json
```

`folio status` prints the three values, and what was realized once something was sold. With
`--json` it prints an object for a status bar widget:

```json
{"current":"3368.13","potential":"7925.00","total":"11293.13","realized":"0.00","realized_gain":"0.00","currency":"USD","unpriced":[]}
```

## Development

```bash
mise install          # Go, gopls and golangci-lint
mise run test
mise run lint
mise run git:hooks    # run both before every push
```

See `CLAUDE.md` for how the project is built and why it is the way it is, and `TODO.md` for what is
planned.

## Notes

Quotes come from Yahoo Finance's chart endpoint, which needs no API key but is not an official API:
it may change or turn requests away without notice, and prices can be delayed. Everything is in USD
for now.
