# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
- `folio self-update` updates a folio that was installed from a release to the latest one, or to a version that is given, after checking the download against the checksums of the release. `-y` skips the question it asks first. Not on Windows yet.

## [v1.1.0] - 2026-10-06
This release is about tax. folio now estimates what the account is worth after it: what is left of
each vest once it is taxed as income, and what selling a lot today would cost in tax on its gain.
The rates are yours to set, with the new `folio config` or without leaving the TUI.

- `folio config` gets and sets the configuration of the account, prints all of it as YAML or with `-o json`, and takes a value back with `--unset`
- `c` opens the configuration in the TUI, to set and unset its values without leaving it
- `folio config tax-rate` sets the rate vests are taxed at, and the Vesting view shows what each vest is worth after tax
- `folio config gains-tax-rate` sets the rate the gain of a sale is taxed at, and the details of a lot show the tax that selling it would cost
- `folio config show-net-summary true` shows the potential and the current value after tax in the header of the TUI, with a badge that says whether the values are `[GROSS]` or `[NET]`
- The Holdings view shows the gain of each lot since it was acquired, in percent
- `enter` in the Holdings view shows the details of the selected lot next to the table, with what it has gained since it was acquired
- The Holdings view leaves out the gain in a terminal too narrow for it, and the header cuts the prices short where it has no room
- `folio demo` opens on an account with both tax rates set

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

[Unreleased]: https://github.com/fgrosse/folio/compare/v1.1.0...HEAD
[v1.1.0]: https://github.com/fgrosse/folio/compare/v1.0.0...v1.1.0
[v1.0.0]: https://github.com/fgrosse/folio/releases/tag/v1.0.0
