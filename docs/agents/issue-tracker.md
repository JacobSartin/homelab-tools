# Issue tracker: GitHub

Issues and specs live in GitHub Issues for `JacobSartin/homelab-tools`.
Run the `gh` CLI inside this checkout; use `--repo JacobSartin/homelab-tools`
when explicitly selecting the repository.

## Conventions

- Create: `gh issue create --title "..." --body-file <file>`.
  Use a UTF-8 file for multiline bodies.
- Read: `gh issue view <number> --json number,title,body,labels,comments`.
- List: `gh issue list --state open --json number,title,body,labels,comments`.
  Apply appropriate label and state filters.
- Comment: `gh issue comment <number> --body-file <file>`.
- Apply or remove labels: `gh issue edit <number> --add-label "..."`
  or `--remove-label "..."`.
- Close: `gh issue close <number> --comment "..."`.

When a skill says "publish to the issue tracker", create a GitHub issue.
When it says "fetch the relevant ticket", read the issue and its comments.

## Pull requests as a triage surface

**PRs as a request surface: no.**

GitHub shares issue and PR numbers. Resolve ambiguous references with
`gh pr view <number>` and fall back to `gh issue view <number>`.

## Wayfinding operations

- Keep the map in one issue labelled `wayfinder:map`.
- Link child tickets as GitHub sub-issues. If unavailable, list children
  in the map and put `Part of #<map>` at the top of each child.
- Use `wayfinder:research`, `wayfinder:prototype`, `wayfinder:grilling`,
  or `wayfinder:task` for child ticket types.
- Record blockers with native GitHub issue dependencies. If unavailable,
  use a `Blocked by: #<number>` line.
- Select the first open, unassigned child in map order with no open blockers.
- Claim with `gh issue edit <number> --add-assignee @me`.
- Resolve by commenting with the answer, closing the child, and adding
  a context pointer to the map's Decisions-so-far.
