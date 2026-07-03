# Go-team history audit — `next` (163 unpushed commits) — 2026-07-03

**Range audited:** `origin/next..next` (163 commits sitting on top of the open PR #1 base).
**Notebook:** `aaf1a665-6ba6-44ab-a7a5-f779fdb503cf` ("mcp v1 history audit (2026-07-03)"),
four windowed sources W1/W2a/W2b/W3.
**Outcome:** rewrite executed → 162 commits, verified green, not pushed.

## Context

The 163 commits are the v1.0 hardening campaign, built on top of PR #1's 166-commit
initial-import base (`origin/next`, GitHub `d0d6e163`). PR #1 (`next → main`, opened
2025-08-27) is a mis-titled ("develop and testing tools … vim integration") 288K-line
whole-repo import, untouched ~10 months. It was left alone; the hardening line was
audited on its own.

## Panel findings, after triage against `git log` (hard rule)

| Panel claim | Triage verdict |
|---|---|
| Drop `5e9a8d18c NEEDSUTIL` | **FABRICATED** — not in the 163 (it is in PR #1's older 166). Bled in from the prompt. |
| `4f595c983 tmp:` adds orphan `tmp/go.mod` | **CONFIRMED** — in-range, sentinel live at HEAD. |
| 3.3MB `analyze-deduplication` binary in history | **CONFIRMED, worse** — added `a5207b2cd`, deleted `da7db67b5`, *both inside the 163* → blob baked into this line's history. |
| Subjects >50 chars (`d7455fb5b` etc.) | Confirmed by measurement; soft guideline, left as-is. |
| `conformance-server:` / `server:` non-standard prefixes | **Bikeshed** — those are the real package dirs; legitimate scoping. |
| `a2499b9cb` core test after `422b2a894` tool → reorder | True but trivial; every commit builds, not a bisect issue. Not acted on. |
| Type-dup `protocol/` removed late | Code critique, out of scope for a history audit. |
| "No Conventional-Commits style present" | **WRONG** — 34 CC-style subjects present, concentrated in W1 (24/34). This was the real consistency issue. |

## Action taken (curation)

Local-only history, so rewritten before any push. `git filter-repo --refs origin/next..next`
(base's 166 commits untouched):

1. **Purged** the `analyze-deduplication` blob from history (strip path across add+delete).
2. **Dropped** `4f595c983 tmp:` (became empty once `tmp/go.mod` stripped) → 163 → 162 commits.
3. **Reworded** the 10 genuine CC-*type* violations (`feat:`/`refactor:`/`chore:`/`test:` that
   had a real package home) to package-scoped Go style; **kept** the 24 legitimate `docs:`/`ci:`
   commits (valid Go-repo scopes — over-rewriting good history is worse).
4. **Migrated** all 130 git notes onto the rewritten hashes via filter-repo's commit-map.

## Verification

- Final tree differs from original by exactly `tmp/go.mod` removal; base SHA intact.
- `go build ./...`, `go vet .` clean; `go test -race -count=1 .` green (~26s).
- Original preserved on `backup/pre-history-curate-20260703T195500Z`.

## Not done (user-gated)

- **Not pushed.** `next` is local; force-push required (history rewritten).
- **PR #1 fate** deferred — left open/untouched per decision.
- v1.0.0 tag deferred.
