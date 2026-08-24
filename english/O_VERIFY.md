# Optimize verification — prove behavior is unchanged

Optimize (O) only ships when we can show the **end result is the same** for the system and for the **end user**.

This is the standard process to bake into Frontier Ship. Each **code stack** adds a short playbook (how to run tests in that ecosystem).

---

## Standard process (every Opt-ID)

```text
1. Pick ONE open Opt-* issue (small PR)
2. Capture baseline
   a. Equivalence / code tests (API, pure functions, golden outputs)
   b. Browser smoke (critical user paths)
3. Implement the Opt change (simplest behavior-preserving transform)
4. Re-run the same tests — must stay green / same assertions
5. PR body = Opt report section + "Verification" (commands + **paste/verbose log**)
6. Merge; close the issue
```

### Visibility (required)

Customers (and agents) must **see** the proof, not only hear “tests passed.”

| Where | What |
|-------|------|
| **Local / agent terminal** | Always run with **verbose** output, e.g. `pytest -v --tb=short`. Paste the summary (and failures) into the chat/PR. |
| **GitHub (when remote is GitHub)** | Add a **Actions** workflow that runs the same suite on every PR/push. The check must be visible on the PR. |

Do **not** treat a silent `pytest -q` with exit 0 as sufficient communication. Quiet mode is fine for CI logs after the first green run, but humans reviewing Opt work should get a visible list of passed tests.

### Two proof layers

| Layer | Question | Typical tools |
|-------|----------|----------------|
| **A. Equivalence (code)** | Same outputs / contracts for the same inputs? | unit/integration, API clients, golden files |
| **B. Browser (user)** | Same UX for the human path? | **Playwright** (default for web UIs) |

Layer A does **not** replace Layer B. Fast APIs can still break clicks, focus, tokens, empty states.

### What “same” means

- **API / data:** same status codes, JSON fields, ordering guarantees we document, file schema (whitespace-only JSON diffs may be OK if stated).
- **User:** same visible flows (list, add, edit, toggle, delete, search, auth token still works).
- **Not required:** identical timings, log lines, or internal structure.

### Frontier Ship hooks (future)

| Step | Command / artifact (planned) |
|------|------------------------------|
| Record baseline | `frontier optimize verify baseline` |
| After change | `frontier optimize verify` |
| Stack playbook | `.frontier/optimize/stack.md` or `english/stacks/<id>.md` |
| PR checklist | Auto-append Verification section to `optimize pr-body` |

Until those exist: follow this doc manually and paste results into the PR.

---

## Stack playbooks (contextual + repeatable)

Full index and template: **[`stacks/README.md`](stacks/README.md)**.

| Stack | Playbook |
|-------|----------|
| Python + static UI | [`stacks/python-web-static.md`](stacks/python-web-static.md) |
| Go HTTP | [`stacks/go-http.md`](stacks/go-http.md) (stub) |
| Node Express + SPA | [`stacks/node-express.md`](stacks/node-express.md) (stub) |

### Local loop for one Opt (any stack)

```text
# baseline (verbose — show named results in terminal)
<playbook Layer A + B commands>
# implement Opt-00N
<same commands again>
frontier guard && frontier plan && frontier apply
# PR = Opt section + Verification paste + CI green (if GitHub)
```

---

## Relation to Learn / Guard / Slim

- **Learn:** know the project kind (web? API-only?).
- **Guard:** security still clean after Opt.
- **Slim:** do not confuse deletion of unused code with Optimize.
- **Optimize:** speed with **A + B** proofs.
