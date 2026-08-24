# How Guard (G) is implemented (security)

CLI: **`frontier guard`** (letter alias **`G`**). Formal definition-set in axioms may still be written `V` (vulnerability set); the command is Guard.

## Shape

```
  english/SECURITY_POLICY_OWASP.md     what humans mean
           │
  haskell/src/Frontier/OWASP.hs        proof shelf (rule ids)
           │
  internal/owasp + internal/vscan      programmatic checks (no tokens)
           │
  frontier guard | gate | push         control_point = changeset
           │
  frontier enhance guard               lean brief → host model (residual only)
           │
  ledger                               F0 evidence
```

## Programmatic first (token thrift)

| Layer | Tool | Tokens? |
|-------|------|---------|
| Built-in OWASP v0 | `internal/owasp` regex ScanTree | no |
| Secret surfaces | path names only (`.env`, `.pem`, credentials…) | no |
| Adapters | Checkov / Gitleaks / Trivy if on PATH; semgrep planned | no |
| Scope + inventory | git diff vs main, lang/manifest counts | no |
| Enhance brief | `.frontier/enhance/V-*.md` capped (~12 KiB) | handoff only |

Secret **surfaces** are Guard (security), not Learn.

Gate/plan still **hard-block only** on built-in OWASP High/Critical. Adapter + enhance findings default to **advise** until promoted into English→Haskell→Go definitions.

## Commands

| Command | What it does |
|---------|----------------|
| `frontier guard` / `G` / `exam` | OWASP exam + secret surfaces; seal `exam.owasp` |
| `frontier guard list` | Built-in + adapters |
| `frontier guard checkov` | Checkov adapter if on PATH |
| `frontier guard gitleaks` | Gitleaks adapter if on PATH |
| `frontier guard trivy` | Trivy fs adapter if on PATH |
| `frontier enhance guard` | Programmatic pack + lean brief |
| `frontier enhance seal PATH` | Host result → `enhance.completed` (advise) |
| `frontier gate` | Exam + push rules + seal pass/fail |

## Dispositions

| Disposition | When |
|-------------|------|
| **block** | High/Critical under built-in OWASP (blocks gate/push) |
| **advise** | Non-blocking findings; secret surfaces; enhance results |
| **record** | Clean under current Guard |

## Stick with Guard + Learn

Slim (`frontier slim` / `S`) is planned. Runtime (`frontier runtime` / `R`) is post-ship, not a push gate. Learn (`frontier learn` / `L`) first.
