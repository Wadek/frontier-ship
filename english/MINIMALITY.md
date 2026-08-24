# Minimality (why less code wins)

## The Linus problem (without bothering Linus)

Call **“Linus”** any person with:

- high computer-science skill, and  
- low patience for noise.

Their attention is scarce.  
If every vibe-coded repo dumps thousands of lines on that attention, the commons dies.

So Frontier treats **volume** as a first-class hazard — same family as secrets and ungated push.

```
  All people × all vibe code  →  infinite noise
  Least code that still proves the result  →  finite, teachable, reviewable
```

**Least code that produces the sealed result is probably best.**  
Fewer lines ⇒ fewer tokens ⇒ fewer places for false “security.”

---

## Principle M1 — Minimum sufficient artifact

A change is better when it is smaller **and** still satisfies F0–F4.

We do **not** maximize languages, files, frameworks, or comments.  
We maximize **clarity per byte**.

Operational (for gates / reviews):

| Check | Meaning |
|-------|---------|
| LOC budget | Prefer diffs under a stated max (default soft: 400 lines net; hard later) |
| File budget | Prefer fewer new files |
| Dup budget | Do not re-implement what Haskell already states |
| Token budget | AI authors should stop when the gate can pass — not when the model is bored |

These numbers are **policy knobs**, not moral absolutes. Start soft (`FRONTIER_SOFT`), then tighten.

---

## Principle M2 — Any language at the edge; one language at the core

We were wrong to say “contributors may only write Go/Haskell.”

**Contribute in whatever language you can.**  
**Admit into Frontier only after reverse-engineering into the core.**

```
  Your PR (any language, any style)
           │
           ▼
  Reverse-engineer  →  base math / logic claim
           │
           ▼
  Write / update Haskell (pure proof form)
           │
           ▼
  English note (simple) + Go runtime only if something must run
           │
           ▼
  Admit (F5)
```

So:

| Layer | Language rule |
|-------|----------------|
| Edge contribution | **Open** (Python, Rust, C, scripts, …) |
| Human meaning | **English** (simple) |
| Universal compute / proof | **Haskell** (for now) |
| Local runner / git / CLI | **Go** (for now) |

Haskell is not a clubhouse. It is the **proof shelf**.  
Go is not holy. It is the **tool that talks to git today**.

---

## Principle M3 — Every push is reverse-engineered

For *this* project, a push is incomplete until:

1. **Claim** — one English sentence: what became true  
2. **Proof surface** — Haskell types/functions (or a clear “no new law; runtime only”)  
3. **Witness** — how to *see* it (command output, ledger rows, test)  
4. **Delta size** — LOC called out; justify if over budget  

If the change cannot be reduced to a small claim + Haskell, it is probably vibe bloat. Refuse or shrink.

---

## Seeing the test (visibility)

Security you cannot see does not teach.

Minimum visible witnesses:

```text
git frontier explain
git frontier status
git frontier gate
git frontier ledger
git frontier demo      # short theater: ladder + last seals
```

No dashboard required. The terminal is enough.  
If a change adds UI, it must still shrink or clarify these witnesses — not bury them.
