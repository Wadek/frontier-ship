# Permanent local pre-GitHub build pipeline

Every session, every repo, every push. Do not invent a shorter path.

**Spec:** `D:\wakalabs\frontier\PIPELINE.md`  
**Tool:** Frontier Ship (`frontier` / `git frontier`) at `D:\frontier\bin`  
**Source:** `C:\Users\waka\src\frontier-ship`

## Required sequence before GitHub

1. Feature branch (never commit or push `main` / `master`).
2. Clean commit.
3. `frontier hygiene` inspects the changeset for AI provenance (watermarks-remover at `http://127.0.0.1:8765`). Advise by default.
4. `frontier plan` must exit 0 (OWASP Guard + push rules).
5. `frontier apply` must exit 0 (seals `gate.passed`).
6. `git push` — global hooks re-run plan/apply. Never `--no-verify`.
7. Open a PR into `main`. Human merges. Agents do not merge to main.

`FRONTIER_SOFT=1` is forbidden for real ship.

## Forbidden

- `git push --no-verify`
- Overriding `core.hooksPath`
- Calling `C:\Program Files\Git\cmd\git.exe push` to skip the shim/hooks
- GitHub MCP `push_files`, `create_or_update_file`, or `merge_pull_request`
- Pushing or committing on `main`/`master`

## Tests

When the repo has a stack playbook or test suite, run it **verbose** locally (`pytest -v`, etc.) and paste the log. Doctrine: `english/O_VERIFY.md`. Tests are not a substitute for `frontier plan`/`apply`.
