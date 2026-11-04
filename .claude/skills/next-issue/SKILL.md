---
name: next-issue
description: Pick the next open issue of folio from GitHub, build it in a worktree and open a pull request for it. Use when asked to work on the next issue, the next ticket or the next item, or on a given issue number.
---

# Work on the next issue

One run is one issue and one pull request. `CLAUDE.md` says how folio is built, and all of it
applies here.

## 1. Pick

If an issue number was given, that is the issue. Otherwise it is the first line of:

```sh
gh issue list --state open --limit 1 \
  --search 'label:"size:S","size:M","size:L" -label:needs-discussion no:assignee -linked:pr sort:created-asc'
```

That is the oldest open issue that has a size, whose design is settled, and that nobody has taken.
If the list is empty, say so and stop: an issue without a size label or with `needs-discussion` is
not ready, and picking one anyway is not the way out.

Read it with its comments (`gh issue view <number> --comments`), then take it, so that a session
that runs next to this one picks another:

```sh
gh issue edit <number> --add-assignee @me
```

## 2. Check that it can be built

Look at the code the issue is about before writing any. Hand the issue back rather than guess if

- it is done already, or the code has moved on and it no longer applies,
- it leaves a choice open that changes what the user sees or what the database holds, or
- it is larger than its label says and wants splitting.

To hand it back, comment on the issue with what was found and what has to be decided, add the
`needs-discussion` label, remove the assignee (`--remove-assignee @me`), report and stop. Do not go
on to the next issue: the user asked for one.

## 3. Build

Enter a worktree with `EnterWorktree`, named after the issue, such as `issue-21-fetch-quotes`.
Build test-first, one commit to a red-green-refactor cycle. Keep to the issue: what turns up next
to it is offered to the user as an issue of its own at the end, not built.

Before the pull request, run `mise run test` and `mise run lint`. For a change to the TUI, read the
goldens that changed, and look at the running TUI with `tmux capture-pane`. Update `README.md`,
`CHANGELOG.md` and `CLAUDE.md` where the change is one they describe.

## 4. Open the pull request

Push the branch and open a pull request against `main` whose body says what was built and why, how
it was checked, and ends with `Fixes #<number>`, which closes the issue on merge. Do not merge it.

Report the pull request, anything the user has to decide, and the follow-ups that came up, each as
an offer to create an issue.
