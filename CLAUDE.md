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

Find types, definitions, references and callers with the LSP (gopls) rather than by reading
files, and keep Grep for text such as comments, strings and config values. After an edit, check
the LSP diagnostics and fix them before moving on. Call `findReferences` before changing a
signature or an interface method, renaming something, or changing the type of a field.

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
  what the domain asks of a source of prices, and `internal/yahoo` is the one there is. The views
  show the last quote the database has and replace it once a fresh one arrives, so the TUI opens
  without waiting for the network and works without one. Tests never touch the network.
- **A lot and a grant are typed as one line**, a spec such as `12.5 PANW 2026-03-15 @380.12`, which
  `NewLot` and `NewGrant` parse. The CLI verbs and the TUI's dialogs both go through them, so the
  grammar is the same everywhere.
- **Realized money is not part of the three values.** What the sales brought in is stated next to
  them, never added to the total, which stays the bank's.
- **Tax is an estimate at rates the user sets.** folio cannot know what a vest is taxed at as
  income, or a gain, so the account has a rate for each (`folio config tax-rate` and
  `gains-tax-rate`). Every lot and every sale is taxed on its own gain, and a loss takes nothing off
  another (#31). The three values stay the bank's, before tax, unless the user asks for them after
  tax with `show-net-summary`. What applies to the whole account is kept in the `settings` table.
- **Everything is in USD for now**, the currency the stock trades in (#32).

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

Plans live in the issues of the GitHub repository, `fgrosse/folio`, not in a file here. An open
issue is work that is still to do, and closing it is what marks it done: a pull request says
`Fixes #<number>` so that merging it does.

- **An idea that comes up is offered, not filed.** When work on one thing turns up another, such as
  a gap, a follow-up or something left out on purpose, say so and offer to create an issue for it.
  Create it only once the user agrees. Neither keep a list in a file nor leave it in chat history.
- **Keep the issues few.** Search the open and the closed ones first (`gh issue list --state all
  --search`), and add a comment to the one that covers the idea already rather than opening a
  second. One issue is one improvement that can be built and merged by itself. Do not open one for
  what the pull request at hand can settle, for a wish without a need behind it, or for a question:
  ask that.
- **An issue says what is missing and why it matters**, in a few sentences, then what to build,
  and the code it touches if that is known. It gets one size label, and `needs-discussion` on top
  as long as its design is open, which keeps it from being picked up:
  - `size:S`: one or two red-green cycles, in one package or one view.
  - `size:M`: several cycles across the domain and a front end, maybe a migration.
  - `size:L`: a new concept, with new data, a new source or a design that is still open.
- **The next issue to work on** is the oldest open one with a size label and without
  `needs-discussion`. The `next-issue` skill picks it, builds it in a worktree and opens the pull
  request.

## Where knowledge goes

This file is read in full at the start of every session, so it holds only what applies across
packages and cannot be read off the code: a rule such as "no cgo" or "no floats for money", or the
vocabulary of the domain. What one type, function or view does, and why, goes into its doc comment,
and a gotcha into a comment at the code it affects. A feature does not get an entry here for being
new.
