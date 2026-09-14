---
name: test-as-you-go
description: >
  Run the project's tests in the same turn as code is produced. Use when
  implementing, editing, refactoring, or generating application code; when
  the user wants tests as you go, TDD, a vim-like produce-and-verify
  loop, a full suite of tests, a smoke test, smoke tests, or to structure a test; or when they run
  /test-as-you-go.
when-to-use: >
  producing code, writing or editing source files, implementing a feature,
  refactoring, TDD, run tests as you go, test as you produce, full suite of
  tests, smoke test, smoke tests, structure a test, at least we have some testing done, /test-as-you-go
argument-hint: "[on|off|test command]"
---

# Test as you go

Tests run in the same turn as the change. Do not batch a pile of edits and
only then discover they fail.

## Loop

1. Make one coherent change (one behavior, not the whole feature).
2. Run the smallest test command that covers it. Paste the log if it fails.
3. On failure: fix, rerun, then continue. Do not start the next behavior.
4. Before claiming done, run the project's full test command once.

## Command

Resolve in this order:

1. First non-comment line of `.grok/test-on-stop` in the workspace root.
2. Auto-detect: `package.json` `scripts.test` (`pnpm test` / `yarn test` /
   `npm test` from the lockfile), `cargo test`, `go test ./...`,
   `pytest -v --tb=short`, `make test`.
3. If none of those exist, say so and skip — do not invent a runner.

Prefer a targeted invocation when the change has an obvious test
(`pytest path -k name`, `cargo test name`, `go test ./pkg -run Name`).

## `/test-as-you-go`

- No args or `on`: write `.grok/test-on-stop` with the resolved command.
  That arms the Stop hook (rerun on turn end after this session wrote
  files; block finish on failure). Confirm the path and command.
- `off`: delete `.grok/test-on-stop`.
- Anything else: write those args as the command.

Do not create the file in the user's home directory.

In-session tests do not replace the Frontier pre-push pipeline
(`frontier plan` / `frontier apply`) before GitHub.
