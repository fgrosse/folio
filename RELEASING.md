# Release Process

This document describes the steps to cut a new release of `folio`.

## Versioning

This project follows [Semantic Versioning](https://semver.org/):
- **Patch** (`v1.x.Y`): bug fixes only
- **Minor** (`v1.Y.0`): new features, backwards-compatible
- **Major** (`vX.0.0`): breaking changes

## Steps

### 1. Update `CHANGELOG.md`

- Rename the `## [Unreleased]` section to `## [vX.Y.Z] - YYYY-MM-DD`
- Add a new `## [Unreleased]` section at the top with the content `_Nothing yet_`
- Update the comparison links at the bottom of the file:
  ```
  [Unreleased]: https://github.com/fgrosse/folio/compare/vX.Y.Z...HEAD
  [vX.Y.Z]: https://github.com/fgrosse/folio/compare/vA.B.C...vX.Y.Z
  ```
  The first release has no version before it to compare with, so its link is
  `[v1.0.0]: https://github.com/fgrosse/folio/releases/tag/v1.0.0`.

### 2. Commit and push to main

```bash
git add CHANGELOG.md
git commit -m "Release vX.Y.Z"
git push origin main
```

### 3. Create and push a signed tag

```bash
git tag -s vX.Y.Z -m "Release vX.Y.Z"
git verify-tag vX.Y.Z
git push origin vX.Y.Z
```

> **Important:** Do not forget the `-s` flag. Tags are not signed by default (`tag.gpgSign` is not set),
> so a plain `git tag vX.Y.Z` silently creates an unsigned lightweight tag. `git verify-tag` must report
> a good signature before the tag is pushed.

> **Important:** The tag must point to the commit that includes the updated `CHANGELOG.md`.
> If you need to move the tag (e.g. you pushed it too early), run:
> ```bash
> git tag -f -s vX.Y.Z -m "Release vX.Y.Z"
> git push --force origin vX.Y.Z
> ```
> Only do this before running goreleaser, as force-pushing a tag after a GitHub Release is published will cause issues.

### 4. Run goreleaser

goreleaser is one of the tools in `mise.toml` that `mise install` installs. To try the build
without publishing anything, run `goreleaser release --snapshot --clean` and look at what it
leaves in `dist/`.

Extract the release notes for the current version from `CHANGELOG.md` and pass them to goreleaser:

```bash
VERSION=vX.Y.Z
awk "/^## \[$VERSION\]/{found=1; next} /^## \[v/{if(found) exit} found" CHANGELOG.md > /tmp/release-notes.md
GITHUB_TOKEN=$(gh auth token) goreleaser release --clean --release-notes=/tmp/release-notes.md
```

This will:
- Build binaries for Linux, macOS, and Windows
- Create tarballs and a `checksums.txt`
- Publish a GitHub Release with the built artifacts

> **Important:** `--release-notes` does **not** currently reach the published
> release. Our `.goreleaser.yaml` sets `changelog.disable: true`, which skips the
> pipeline stage that applies the flag, so goreleaser publishes the release with an
> **empty body**. This is silent, there is no warning.

### 5. Set the release notes

Because of the above, attach the notes explicitly after goreleaser has published:

```bash
gh release edit "$VERSION" --notes-file=/tmp/release-notes.md
```

### 6. Verify the release

```bash
gh release view "$VERSION" --json tagName,isDraft,body --jq '.tagName, .isDraft, .body'
gh release view "$VERSION" --json assets --jq '.assets[].name'
```

Check that the body is not empty, that the release is not a draft, and that all
nine assets are present (eight archives plus `checksums.txt`).

## Checklist

- [ ] `CHANGELOG.md` updated (unreleased → version + date, new empty unreleased section, links)
- [ ] Changes committed and pushed to `main`
- [ ] Signed tag created with `git tag -s`, checked with `git verify-tag` and pushed (pointing to the commit that updates the changelog)
- [ ] `goreleaser release --clean --release-notes=/tmp/release-notes.md` run successfully
- [ ] Release notes attached with `gh release edit` (goreleaser leaves the body empty)
- [ ] Release verified: non-empty body, not a draft, all nine assets uploaded
