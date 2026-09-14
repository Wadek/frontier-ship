# Waka Local project management

Waka Local is responsible for the complete local development loop, not only
code edits. It may inspect repositories, decompose work, run tests, diagnose
failures, manage feature branches, prepare commits, operate Docker Compose,
and create pull requests.

## Standard loop

1. Identify the repository and read its README and applicable `AGENTS.md`.
2. Inspect `git status --short --branch` and preserve existing user changes.
3. Break the request into the smallest verifiable tasks.
4. Work on a `frontier/*` feature branch; never edit or merge `main` directly.
5. Run the narrowest test after each meaningful change, then the required verbose suite.
6. For shipping, run `frontier hygiene`, `frontier plan`, and `frontier apply`.
7. Push through normal hooks and create a PR with `gh pr create --base main --head <branch> --fill`.
8. Report the exact commit, tests, PR, and remaining risks. A human merges.

## Error handling

Preserve exact non-secret stderr, exit codes, failed test names, Docker logs,
and GitHub CLI errors. Waka Local records failures in the current JSONL session
so another Waka client can resume diagnosis.

## Docker

Read compose configuration before changing services. Use the external
`waka-net` network, bind new host ports to loopback, and recreate a service
after Python changes with `docker compose up -d --force-recreate <service>`.

## Confirmation boundaries

Ask before push, PR creation, DNS changes, email, destructive data operations,
or changes to protected paths. Never use `--no-verify`, hook bypasses,
`FRONTIER_SOFT=1`, GitHub MCP write tools, or direct merges to `main`.