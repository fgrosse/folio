# folio

A terminal-first tracker for the stock someone holds and is still to be given by their employer,
written in Go and storing its state in a local SQLite database.

## What this is

`folio` answers what the web site of the bank or broker that holds an equity plan answers, without
logging in there: what the shares held are worth today, what the shares still to vest are worth,
and what the two add up to. It is built for stock that comes from work as RSUs, which arrive on a
vesting schedule. Stock bought privately fits the same model and matters less for now.

Its user is whoever runs it, about their own account: one person, one database on their machine,
no server and no login. `folio demo` opens it on a made-up account for anyone who wants to look
around first.

The three numbers are the bank's, and so are their definitions:

- **Current account value**: the value of the holdings that can be sold. In folio those are the
  *lots*.
- **Potential benefit value**: the value of the holdings that are unvested, or vested and pending
  release. In folio those are the *vests* of a grant that have not been released into a lot.
- **Total account value**: the sum of the two.

## LSP for Code Navigation

Use the LSP (gopls) as the primary navigation tool. Do not read files to understand types,
references, or definitions. Query LSP first. File reading is a fallback only when LSP cannot
answer.

**LSP tools and when to use them:**
- `goToDefinition` / `goToImplementation` - jump to source
- `findReferences` - see all usages across the codebase
- `workspaceSymbol` - find where something is defined
- `documentSymbol` - list all symbols in a file
- `hover` - type info without reading the file
- `incomingCalls` / `outgoingCalls` - call hierarchy

Use Grep/Glob only for text/pattern searches (comments, strings, config values) where LSP
does not help.

After writing or editing code, check LSP diagnostics before moving on. Fix any type errors
or missing imports immediately.

**Before ANY of these actions, you MUST call findReferences first:**
- Changing a function or method signature
- Renaming a type, field, or function
- Changing an interface method
- Making a field a pointer or changing its type

**Setup:** gopls is managed via mise (`mise install`). The `gopls-lsp` Claude Code plugin is
enabled in `.claude/settings.json` and connects Claude Code to the gopls language server.

## Architecture decisions

- **Core domain logic lives in `internal/portfolio`**, storage-agnostic where practical. The front
  ends are thin clients over it: `internal/cli` (the plain verbs, run from `cmd/folio`) and
  `internal/tui`.
- **The TUI is the primary interface.** Bare `folio` launches it. The plain verbs stay scriptable:
  a status bar widget shells out to `folio status --json` and must never need a TTY.
- **SQLite is the single source of truth**, opened in WAL mode and with a busy timeout. There is no
  daemon: the TUI, the CLI and any widget read and write the database file directly, at the same
  time if it comes to that.
- **folio builds without cgo.** The SQLite driver is `modernc.org/sqlite`, which is SQLite in Go,
  so that `go build` is all a release for Linux, macOS and Windows takes and nobody needs a C
  compiler to install folio. Do not bring in a dependency that needs cgo.
- **The store owns no clock.** Whatever depends on today, such as whether a vest is due, takes the
  day from its caller, so a test can pick any day.
- **Shares and prices are decimals, never floats.** `github.com/shopspring/decimal`, stored as text.
  An employer plan can release fractions of a share, and a float would not add them up exactly.
- **Dates are days, not instants.** A lot is acquired and a vest is due on a calendar day, held as a
  `time.Time` at midnight UTC so that two of them compare without a time zone getting in the way.
- **The data model is small.** A *lot* is shares that are held: a symbol, a number of shares, the
  day they were acquired and what one of them cost that day. A *grant* is an award of shares that
  vest over time, and a *vest* is one day on which some of them do. A vest stays potential until it
  is *released*, which is what turns it into a lot - with the number of shares that actually
  arrived, which is fewer than vested whenever some were withheld for tax, and at the value of a
  share on that day as the lot's cost. The vest keeps the number that vested, so both are there for
  a tax estimate to work from. A *sale* is shares of one lot sold on one day at one price. It is a
  record of its own rather than fewer shares on the lot: the lot keeps what it was acquired with,
  and what is held is that less its sales. The user says which lot a sale is from.
- **Prices come from behind an interface and are cached in the database.** `portfolio.Quoter` is
  what the domain asks of a source of prices, and `internal/yahoo` is the one there is, over an
  endpoint that needs no key and is no official API. The views show the last quote the database has
  right away and replace it once a fresh one arrives, so the TUI opens without waiting for the
  network and still works without one. Tests never touch the network.
- **A lot and a grant are typed as one line**, a spec: `12.5 PANW 2026-03-15 @380.12` and
  `Payout: 10 PANW monthly x24 from 2026-01-15`. `NewLot` and `NewGrant` have the grammar in their
  doc comments, and the CLI verbs and the TUI's dialogs both go through them.
- **Realized money is not part of the three values.** Current counts what is left of the lots, and
  what the sales brought in is stated next to the values, never added to the total, which stays
  the bank's.
- **A form for what one line cannot say.** A lot, a grant and a release are a spec in an
  `InputDialog`. A sale has notes of several lines, so it is entered in a `FormDialog`, which has a
  field for each value. Both hand what was typed to a function of the view's and know nothing of
  what it means.
- **Every view shows the same account.** They all receive the one `PortfolioLoadedMsg`, whichever
  of them asked for the load, and each renders the same header with the three values. The Holdings
  view is the one that keeps the quotes fresh: it fetches them on start and every five minutes.
- **Tax is an estimate at two rates.** A vest is taxed as income at a rate that depends on the rest
  of the year's income, which folio does not know, so the user sets one rate in percent for the
  whole account (`folio config tax-rate`) and folio takes it off each vest. What is held was taxed
  when it vested, and a sale of it is taxed on the gain since, at a rate of its own
  (`folio config gains-tax-rate`). Each lot is taxed on its own gain, as the Holdings view shows it:
  a lot that lost is not taxed and takes nothing off the gain of another, and a lot without a cost
  has no gain that is known. The three values stay the bank's, before tax, unless the user asks
  for one after tax: `folio config potential net` for the potential value after the tax on the
  vests, and `folio config current net` for the current value after the tax on the gains. The
  TUI's header marks a value that is after tax `(net)`, and the total is the two values it shows,
  so that the values on screen add up. The rates are kept in the `settings` table, which holds
  what applies to the whole account, one value by name, so that it is in the database like
  everything else.
- **Everything is in USD for now**, the currency the stock trades in. Showing another currency is
  in `TODO.md`.
- **The demo is the one verb with a database of its own.** Every other verb shares the database of
  the real account, which the root command opens and creates before the verb runs. `folio demo`
  overrides those hooks, makes up an account in a temporary database, opens the TUI on it and
  removes it afterwards, so that trying folio leaves nothing behind and the real account alone.
- **The TUI is started through a hook**, `Folio.openTUI`, which a test replaces: the real one needs
  a terminal, and the tests of a verb that opens it look at the store it was handed instead.

## How we build this

Claude writes this project, tests and implementation both, and writes it test-first. The cycle is
red, green, refactor, and it is small:

1. **Red.** Write one failing test for the next small unit of behavior, and run it to see it fail
   for the reason it should - a test that fails to compile for want of the function it tests counts,
   a typo in the test does not.
2. **Green.** Write just enough to make it pass, and run the tests of the whole module.
3. **Refactor.** With the tests green, tidy up what the step left behind, in the code and in the
   tests, and run them again.
4. **Commit.** One commit for the cycle, the test together with the code that makes it pass.

Then the next cycle. One test at a time: not a batch of tests written ahead of the code, and no
behavior without a test that asked for it. A table-driven test is one test as long as its cases are
the same behavior, such as the ways a lot can be invalid; a case that needs new code of its own is
a cycle of its own.

Scaffolding and boilerplate (project layout, tool wiring, config files) are exempt, and so is
refactoring that keeps the existing tests green. Those are committed on their own.

Commits follow the Go convention: `<package>: <what changed>` in the imperative, with the package
path without `internal/`, and a body that says what was done and why.

## Tooling

- **mise** manages the Go version (`mise.toml`) and is the task runner - add `[tasks.*]` entries
  instead of a Makefile, but only when asked to; don't add tasks speculatively.
- **Go style**: follow the `go-dev` conventions (Google Go Style Guide + local additions for error
  wrapping, logging, testing, concurrency). Use `go install ./...` over `go build ./...`.
- **Comments say why.** A doc comment explains what a thing is for and why it is the way it is, in
  full sentences, rather than restating its name.
- **Charm v2 throughout.** Import paths are `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`,
  `charm.land/lipgloss/v2`, *not* `github.com/charmbracelet/...`, which is the v1 line.
- **Bubble Tea messages are exported**: every `...Msg` type in `internal/tui` gets an exported name,
  even when nothing outside the package sends or receives it. Their fields can stay unexported.
- **Logging**: not wired up. If a daemon or HTTP surface ever appears, use `go.uber.org/zap`.

## Testing

- **Libraries**: `github.com/stretchr/testify` (`require` for fatal assertions, `assert` for
  non-fatal). No `t.Parallel()` by default.
- **The store is tested against a real SQLite database** that lives in memory (`NewTestingStore`),
  so the tests run the same SQL as the program.
- **A computed decimal is compared as text**, `account.Total().String()`, not with `assert.Equal` on
  the decimal: two decimals of the same value are not always the same bytes, and those are what
  testify compares. A decimal that was only parsed, stored and read back is safe to compare.
- **TUI tests come in layers**, each catching what the one below cannot: pure formatting of a row,
  message and state transitions through `Update` with the `runCmd` helper, and golden-file frames.
- **Goldens**: regenerate with `go test ./internal/tui/ -update`, then *read* the result before
  committing: `-update` blesses whatever renders that day.
- **Screenshot the running TUI with `tmux capture-pane`.** ANSI-stripping a captured frame lies
  about spacing: Bubble Tea emits cursor-forward escapes instead of runs of spaces.

## What's next

Plans live in `TODO.md`, not here. Check it at the start of a session and keep it current: delete
an item once it is done, and add an idea as soon as it comes up rather than leaving it in chat
history. It holds only work that is still to do. A decision goes into this file instead, and a
gotcha goes into a comment at the code it affects.
