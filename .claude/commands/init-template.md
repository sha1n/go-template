---
description: Initialize this repo from go-template — interview, rename, tailor docs, commit, and remove the init scaffolding.
allowed-tools: Bash, Read, Write, Edit, AskUserQuestion
---

You are initializing a brand-new project created from the `go-template` template.
Drive the whole process; the user should only answer a few questions.

## 1. Detect defaults (do not assume — show what you found)

- Owner/repo: run `git remote get-url origin` and parse `github.com/<owner>/<repo>(.git)`
  or `git@github.com:<owner>/<repo>(.git)`. If there is no remote, leave blank.
- Go version: run `go version` and extract `X.Y` (e.g. `1.22`).

## 2. Interview (use AskUserQuestion)

Confirm or collect, pre-filled with the detected values:
- **owner**, **repo**, **go-version** (all required).
- **project purpose** (one sentence) and a **short description** (2–4 sentences),
  used to write the README and CLAUDE.md.
- **CI OS coverage** — which operating systems CI should run `go test ./...` on.
  Offer these options (default = the first):
  - `Linux + Windows` (the template default)
  - `Linux only`
  - `Linux + macOS + Windows`
  - `Linux + macOS`

  Note when asking: GitHub's macOS runners are billed at a higher rate than Linux.

Do not proceed until owner, repo, and go-version are non-empty.

## 3. Preview, then apply the mechanical rename

- Preview: `go run ./internal/bootstrap --owner <owner> --repo <repo> --go-version <ver> --dry-run`
  and show the user the list of files that will change.
- Apply: rerun the same command **without** `--dry-run`.

## 4. Apply the CI OS choice

Edit the `os:` matrix in `.github/workflows/go.yml` to match the chosen coverage:
- `Linux + Windows` → `[ubuntu-latest, windows-latest]` (no change needed)
- `Linux only` → `[ubuntu-latest]`
- `Linux + macOS + Windows` → `[ubuntu-latest, macos-latest, windows-latest]`
- `Linux + macOS` → `[ubuntu-latest, macos-latest]`

Then reconcile the comment above the `os:` line: if macOS was **included**, remove
the "macOS is intentionally excluded … Do NOT add macOS" comment (it now
contradicts the config); otherwise leave it. Keep the `make` pipeline and Coveralls
steps gated to `if: matrix.os == 'ubuntu-latest'` regardless of the choice.

## 5. Generate tailored docs

- **README.md** — replace the template's self-describing README with one about the
  user's project: title = repo, the one-sentence purpose, the description, and a
  short "Build & Test" section derived from the `Makefile` targets (`make`,
  `make test`, `make build`).
- **CLAUDE.md** — create concise project guidance: what the project is, key build/
  test commands (`make`, `make ci-checks`, `make test`), and a note that CI runs
  `go test ./...` on the OSes chosen in step 2. Make it self-contained — do not
  reference `AGENTS.md`, which is removed in step 7.

## 6. Verify the build

- Run `make` (or `go build ./...` if `make` is unavailable). Report failures to the
  user but continue — environment issues should not block initialization.

## 7. Remove the init scaffolding

Delete, in the working tree:
- `internal/bootstrap/` (the engine + tests)
- `.claude/commands/init-template.md` (this command)
- the `init:` target block in the `Makefile`
- `AGENTS.md` — it documents the *template's* policy (CI choices, self-deleting
  scaffolding) and is tuned to this template, not the target repo; the tailored
  `CLAUDE.md` you generated in step 5 supersedes it for the new project.

Leave everything else (all template features and the CI workflow) intact.

## 8. Commit

Stage everything and create a single commit:
`Initialize <repo> from go-template`
Do not add `Co-Authored-By` trailers.

## Notes

- The rename engine is cross-platform and deterministic; you only supply flags.
- Never re-run the engine after step 7 — the scaffolding is gone and the template
  tokens are already replaced.
