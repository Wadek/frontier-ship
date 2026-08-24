# Scoring Frontier against security standards

Frontier is a **push gate** with a supplied vuln set `V`, not a full AppSec program.  
To “pass with a perfect score,” we must say **which score** — and measure against **ground truth**, not vibes.

---

## 1. Standards people usually mean

| Standard | What it scores | Perfect score means | Fit for Frontier today |
|----------|----------------|---------------------|-------------------------|
| **OWASP Top 10** | Category coverage / awareness | Every Top 10 *class* has policy in `V` and at least one real check | We have v0 category mapping; depth is shallow |
| **OWASP ASVS** | App verification controls (L1/L2/L3) | All applicable controls PASS for a target level | Process + tests; not only regex |
| **OWASP Benchmark** / **Juliet** | SAST tool quality | High **recall** + high **precision** on labeled cases | Best scientific compare for our scanner |
| **CWE Top 25** | Weakness coverage | Each relevant CWE detectable or explicitly out-of-scope | Map rules → CWE ids |
| **CVSS** | Severity of a *finding* | Not a product score — grades one vuln | Use for triage, not “Frontier grade” |
| **CIS / cloud benchmarks** | Host/cloud config | N/A to app push gate unless we add infra `V` | Later |
| **Lab guidebooks** (DVNA, Juice Shop) | Intentional vulns listed in docs | Detect **all intentional** issues the lab claims (or clear gate only when fixed) | Ideal **1st customer** scorecard |

**Important:** A perfect ASVS L2 score on a real product is a **program**.  
A perfect score for Frontier-as-tool is closer to: **on a labeled suite, gate decisions match ground truth.**

---

## 2. What we should optimize (Frontier’s score)

Define one primary metric for the push gate:

### Gate decision score (primary)

For each labeled case `C` with ground truth `G`:

| Ground truth | Frontier gate | Result |
|--------------|---------------|--------|
| Has blocking issue under agreed policy | `gate.failed` | **True Positive block** |
| Has blocking issue | `gate.passed` | **False Negative** (miss — bad) |
| Clean under policy | `gate.passed` | **True Negative** |
| Clean | `gate.failed` | **False Positive** (noise — bad) |

```
  Recall    = TP / (TP + FN)     # catch real bad
  Precision = TP / (TP + FP)     # don’t cry wolf
  F1        = harmonic mean
```

**Perfect score for Frontier** = Recall = 1 and Precision = 1 **on the agreed suite and policy level**.

Secondary (customer apps):

- **Lab completion:** % of intentional DVNA/Juice Shop issues either (a) detected by `V`, or (b) fixed so Clean(C,V) and still correct vs guidebook  
- **ASVS coverage:** % of L1 controls with an automated or documented check (grows over time)

---

## 3. How to run the comparison (process)

```
  A. Choose suite
     - Unit: synthetic files per OWASP rule (we control labels)
     - Lab: DVNA guidebook vulns (1st customer)
     - Hard: OWASP Benchmark / Juliet (later)

  B. Freeze policy version
     - english/SECURITY_POLICY_OWASP.md @ commit
     - haskell Frontier.OWASP @ commit
     - Go scanner @ commit
     - Call this V_rev

  C. For each case
     - checkout fixture
     - git frontier gate (SOFT=0)
     - record: findings[], gate ok/fail, axiom log, ledger digests

  D. Score
     - compare to labels → confusion matrix → F1
     - publish english/scores/V_rev.md (simple English)

  E. Improve
     - FN → strengthen Haskell rule + Go check (and English line)
     - FP → narrow pattern / skip vendor (minimality)
     - Re-run until F1 = 1.0 on that suite
```

This is **security-as-code**: policy → proof → runner → **measured** against labels.

---

## 4. How this helps “CEO vibe code”

| Step | Use of standards |
|------|------------------|
| Day 0 | Gate uses current `V` (OWASP Top 10 v0) — may be incomplete |
| Day 1 | Baseline scorecard: what we catch vs what lab/docs say exists |
| Week 1 | Fix **detected** High/Critical OR grow `V` for agreed misses |
| “Perfect” for ship | On **this product’s** agreed checklist (e.g. ASVS L1 subset + no High under V), gate.passed with F1=1 on that checklist’s fixtures |

You do **not** claim “perfect AppSec.”  
You claim: **perfect agreement with the written policy and labeled tests we committed to.**

---

## 5. Improving the lab apps vs improving Frontier

| Improve Frontier (`V` + scanner) | Improve customer/lab code |
|----------------------------------|---------------------------|
| Higher recall/precision on suite | Make Clean(C,V) true with minimal patches |
| Better CWE/ASVS mapping | Delete vibe features that fail policy |
| Skip vendor noise | Add tests that encode ASVS controls |

Both can move toward “perfect”:

1. Frontier F1=1 on synthetic + DVNA labels  
2. Customer tree gate.passed after minimal hardening  
3. Document remaining out-of-scope (deep business logic) honestly

---

## 6. Concrete next build (when we execute)

1. **`testdata/owasp/`** — one tiny file per rule (positive + negative) → automated Go test = living score  
2. **`english/scores/`** — human scorecard after each `V_rev`  
3. **DVNA label list** — map guidebook items → rule ids or `out_of_scope`  
4. Optional later: OWASP Benchmark runner job in CI (heavy; not alpha)

Primary definition of done for “perfect”:

```
  go test ./internal/owasp/...     # all labeled cases green
  DVNA scorecard                   # TP/FN/FP table → F1 = 1.0 for in-scope items
  git frontier gate on hardened tree → passed
```

---

## 7. One-sentence answer

**Run Frontier gate on labeled suites (synthetic OWASP cases + DVNA guidebook), score precision/recall against that ground truth, grow `V` until F1=1 on in-scope cases, then harden the customer app until `gate.passed` — that is “perfect score” under a written policy, not omniscient security.**
