# Optimize (O) — behavior-preserving speed

**Word:** `frontier optimize`  
**Letter alias:** `O` (prefer the word in scripts)

Optimize runs **after** Learn → Guard → Slim. It applies theoretical CS judgment at the code level to find the **simplest** change that **does not alter intended behavior**.

## Pipeline order

```
  frontier scm …     # if customer has no VCS (separate command)
  frontier learn     # L — landscape
  frontier guard     # G — security
  frontier hygiene   # H — AI provenance (watermarks-remover)
  frontier slim      # S — bloat (planned)
  frontier optimize  # O — speed (this family)
```

Axioms apply throughout (F0 evidence, fail-closed ship).

## What it is

Report-driven proposals such as:

- Asymptotic waste (when an equivalent cheaper structure exists)
- Redundant work / N+1 when results stay identical
- Hot-path vs cold-path separation
- Data-structure fit under unchanged equality/API contracts
- I/O and allocation thrash when outputs stay equivalent

## What it is not

Feature work, UX changes, speculative micro-opts without a reason, full rewrites, or LLM-only vibe edits.

## Developer report (canonical in the PR)

Each finding is one review unit (**one small PR** preferred):

```text
### Opt-001 — <title>
- Location: path  function  lines
- Intended behavior: …
- Why wasteful: … (plain English + O(…) if useful)
- Suggested change: …
- Snippet before / after
- Risk / equivalence / tests
- Disposition: advise (default)
```

| Place | Role |
|-------|------|
| **PR body** | Canonical report humans read |
| Commit message | References Opt-ID |
| `.frontier/optimize/` (later) | Offline copy + ledger |
| Diff | The change |

## Small PRs

Never one mega Optimize PR. One Opt-ID (or tight cluster) per branch: `frontier/opt-<id>-…`.

## Commands (now)

| Command | What |
|---------|------|
| `frontier optimize` / `optimize report` | Build report; write `.frontier/optimize/O-*`; seal `optimize.reported` |
| `frontier optimize status` | Show latest Opt-* list |
| `frontier optimize pr-body Opt-001` | Print PR-body markdown for one finding |
| `frontier enhance optimize` | Same programmatic report (host adds residual CS in the PR) |

## Programmatic v0

Large-function **hotspots** (Go/Python/JS/TS, ~80+ lines) → Opt-* **advise** candidates.  
Not an auto-rewrite. Human or enhance fills the concrete transform in a **small PR**.

## Verification (required before merge)

See **[O_VERIFY.md](O_VERIFY.md)** — equivalence (code) + browser (Playwright) proofs.  
Standard process for every Opt-ID; stack playbooks for how to run tests.

## Status

Report writer + hotspot scanner shipped. Verification doctrine in O_VERIFY.md.  
Deeper CS detectors and `frontier optimize verify` automation come next.
