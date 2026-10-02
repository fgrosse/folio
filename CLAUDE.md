# folio

A terminal-first tracker for the stock Friedrich holds and is still to be given, written in Go and
storing its state in a local SQLite database.

## What this is

`folio` answers what the account at the bank's web site answers, without logging in there: what the
shares held are worth today, what the shares still to vest are worth, and what the two add up to.
It is built for stock that comes from work as RSUs, which arrive on a vesting schedule. Stock bought
privately fits the same model and matters less for now.

It is the sibling of `tick` (`~/src/tick`), the time tracker, and mirrors its structure, its code
style and its look: the same packages, the same frame around a table, the same tab bar. When in
doubt about how something should be written or look, see how tick does it.

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
- **SQLite is the single source of truth**, opened in WAL mode where it matters. There is no
  daemon: the CLI and any widget read and write the database file directly.
- **The store owns no clock.** Whatever depends on today, such as whether a vest is due, takes the
  day from its caller, so a test can pick any day.
- **Shares and prices are decimals, never floats.** `github.com/shopspring/decimal`, stored as text.
  An employer plan can release fractions of a share, and a float would not add them up exactly.
- **Dates are days, not instants.** A lot is acquired and a vest is due on a calendar day, held as a
  `time.Time` at midnight UTC so that two of them compare without a time zone getting in the way.
- **The data model is small.** A *lot* is shares that are held: a symbol, a number of shares and
  the day they were acquired. A *grant* is an award of shares that vest over time, and a *vest* is
  one day on which some of them do. A vest stays potential until it is *released*, which is what
  turns it into a lot - with the number of shares that actually arrived, which is fewer than vested
  whenever some were withheld for tax.
- **Prices come from behind an interface and are cached in the database.** The views show the last
  quote the database has right away and replace it once a fresh one arrives, so the TUI opens
  without waiting for the network and still works without one. Tests never touch the network.
- **Everything is in USD for now**, the currency the stock trades in. Showing EUR is in `TODO.md`.

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
behavior without a test that asked for it. A table-driven test grows a case per cycle.

Scaffolding and boilerplate (project layout, tool wiring, config files) are exempt, and so is
refactoring that keeps the existing tests green. Those are committed on their own.

Commits follow the Go convention of tick's history: `<package>: <what changed>` in the imperative,
with the package path without `internal/`, and a body that says what was done and why.

## Tooling

- **mise** manages the Go version (`mise.toml`) and is the task runner - add `[tasks.*]` entries
  instead of a Makefile, but only when asked to; don't add tasks speculatively.
- **Go style**: follow the `go-dev` conventions (Google Go Style Guide + local additions for error
  wrapping, logging, testing, concurrency). Use `go install ./...` over `go build ./...`.
- **Comments say why.** As in tick, a doc comment explains what a thing is for and why it is the
  way it is, in full sentences, rather than restating its name.
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
