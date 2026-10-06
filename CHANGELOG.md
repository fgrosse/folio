# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
- `folio config` to get and set the configuration of the account, such as the tax rate
- The Vesting view shows what each vest is worth after tax
- `c` opens the configuration in the TUI, to set and unset its values without leaving it
- `folio config --unset` takes a value of the configuration back
- The Holdings view shows the gain of each lot since it was acquired, in percent
- `folio config gains-tax-rate` sets the rate the gain of a sale is taxed at, and the Holdings view shows the tax that selling each lot would cost
- `folio config show-net-summary true` shows the potential and the current value after tax in the header of the TUI, each marked net
- The Holdings view leaves out the gain and the tax in a terminal too narrow for them
- `enter` in the Holdings view shows the details of the selected lot next to the table, with what it has gained since it was acquired

## [v1.0.0] - 2026-10-02
- Initial release
- Track the shares you hold as lots, with the day they were acquired and what they cost
- Track grants of stock that vest over time, on a monthly, quarterly or yearly schedule or one listed vest by vest
- Release a vest into a lot, with the shares that arrived and what they were worth that day
- Record sales of a lot, with their proceeds, their gain and notes
- Show the current, potential and total value of the account at prices from Yahoo Finance
- A TUI with a view each for holdings, vesting, grants and sales
- The `lot`, `grant`, `release` and `status` commands, and `status --json` for a status bar widget
- `folio demo` to try folio with a made-up account
- `folio version` to print the version of folio

[Unreleased]: https://github.com/fgrosse/folio/compare/v1.0.0...HEAD
[v1.0.0]: https://github.com/fgrosse/folio/releases/tag/v1.0.0
