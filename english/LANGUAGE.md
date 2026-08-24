# Languages of Frontier

## For humans

Everything a person must understand is **simple English** under `english/`.

## For the universe (proof)

The shared compute form is **Haskell** (for now): pure functions, types, tests.  
That is where laws and gates live as *truth*, not as framework fashion.

## For the machine you run today

**Go** builds the local `git` shim and `frontier` CLI. Boring on purpose.

## For contributors

**Any language is welcome at the edge.**  
Pushing into Frontier still requires reverse-engineering:

```
  any-language patch
        → math/logic claim
        → Haskell proof surface
        → English note
        → Go only if runtime must change
```

See [MINIMALITY.md](MINIMALITY.md).

```
  English  →  what we mean
  Haskell  →  what is true
  Go       →  what runs (today)
  *        →  what you may draft in — then reduce
```

We limit **volume**, not **human entry languages**.

---

## Why not C for the runtime?

Git’s engine is C. That does **not** mean Frontier’s policy layer should be C.

| | **Go** (keep) | **C** |
|--|----------------|--------|
| Job | `git` shim, `frontier` CLI, ledger I/O | Upstream `git/git`, kernels, tiny hooks |
| Safety | Memory-safe by default | Easy to violate F1 via bugs |
| Size of *our* code | Small for JSON/CLI/strings | Same features ⇒ more lines, more review |
| Proof story | Runtime only; laws in Haskell | C is not a better proof language than Haskell |
| Reviewer tax | Lower | Higher (Linus-patience people hate vibe-C) |

**Rule of thumb**

- **Haskell** — what is true  
- **Go** — what runs beside git today  
- **C** — only when we must touch git’s own code or OS guts later  

Rewriting the shim in C would add risk and volume without making the axioms clearer.  
If we ever patch upstream git, that patch may be C — and must still reverse-engineer to English + Haskell.