# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
- `folio config` to get and set the configuration of the account, the way `git config` does,
  and to print all of it as YAML or JSON, picked with `--output` (`-o`) as in kubectl
- The `tax-rate` key of the configuration: the rate that vests are taxed at
- The Vesting view shows what each vest is worth after tax at that rate

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
