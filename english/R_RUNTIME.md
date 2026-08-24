# Runtime (R) — post-ship probe and bounded chaos

**Word:** `frontier runtime`  
**Aliases:** `probe`, `chaos`, letter **`R`**

Guard examines the **changeset** before GitHub. Runtime examines **what is actually running** (loopback / habitat network), on a budget, with a sealed allowlist. It is not a second push gate.

## Map from common “defense in depth” write-ups

Those write-ups assume Kubernetes + a public cloud account. Frontier does not. Same *jobs*, different tools, same control points we already named.

| Job in the write-up | Their tools | Frontier control point | What we run |
|---------------------|-------------|------------------------|-------------|
| Pre-deploy gate | Checkov, Trivy, Gitleaks in CI, block Critical | **changeset** — Guard | Built-in OWASP (already blocks High/Critical). Checkov / Gitleaks / Trivy **adapters** when on PATH. Advise until a rule is promoted into English→Haskell→Go. CI should run the same `frontier guard` + named adapters, not a parallel zoo. |
| Post-deploy validation | Prowler, “AI scans” off-peak | **runtime** | `frontier runtime scan` against the **allowlist only** (health/config). No account-wide cloud auditor unless that cloud is later a declared target. |
| Chaos / detection / auto-remediate | Chaos Mesh, Falco, GuardDuty, Lambda, K8s operators | **engagement** (subset of Runtime) | `frontier runtime chaos` — dry-run default, inject only with an explicit env flag, short TTL, ledgered. Detection is compose/container logs + the scan, not a cloud SIEM we do not operate. Auto-remediate is **out of v1** (too much blast radius). |
| Feedback into IaC policy | Custom Checkov policies | **Grow V** (F4) | Chaos or scan findings become an English policy line, then a Haskell rule, then a Go/Checkov check. Do not silently mutate Checkov YAML from a model. |

Do **not** put Trivy image rebuilds, DAST, or chaos on `git push`. Push stays cheap: OWASP + (optional) adapter advise.

## Token share (AI in containers)

Runtime **reports** an intended token share vs what boxed jobs have consumed. Enforcement is **off** unless `FRONTIER_RUNTIME_TOKEN_CAP=1`.

| Knob | Default | Meaning |
|------|---------|---------|
| `token_pct` in allowlist / `FRONTIER_RUNTIME_TOKEN_PCT` | `5` | Intended share, for reporting |
| `FRONTIER_RUNTIME_TOKEN_CAP` | unset (off) | Set `1` later to refuse jobs over the share |
| `FRONTIER_RUNTIME_ALLOWLIST` | `D:\frontier\runtime\allowlist.json` | Only these URLs / docker networks |
| `FRONTIER_RUNTIME_CHAOS` | unset | Must be `1` to inject; otherwise dry-run |

`frontier runtime budget` prints configured / consumed / remaining percentages and seals `runtime.tokens`. Consumed stays 0 until a runner records spend. Allowlist missing still refuses **scan/chaos**; it does not refuse because of tokens.

## Allowlist (fail closed)

Example (`D:\frontier\runtime\allowlist.json`):

```json
{
  "token_pct": 5,
  "targets": [
    {"name": "local-health", "kind": "http", "url": "http://127.0.0.1:8765/health"}
  ],
  "chaos": {
    "enabled": false,
    "max_duration_s": 60
  }
}
```

Rules:

- `http` URLs must be `127.0.0.1` or `localhost` in v1.
- `docker-net` names are recorded only; chaos inject is still dry-run unless `FRONTIER_RUNTIME_CHAOS=1` **and** `chaos.enabled` is true.
- No `0.0.0.0`, no public hostnames, no cloud account IDs in v1.

## Commands

| Command | Effect |
|---------|--------|
| `frontier runtime` / `status` | Show allowlist path, budget %, chaos flag |
| `frontier runtime scan` | HTTP GET allowlisted loopback URLs; optional Trivy if asked later |
| `frontier runtime chaos` | Dry-run the inject plan; inject only with the env flag |
| `frontier runtime budget` | Report configured vs consumed token % |

## What we are not copying

- Chaos Mesh (needs Kubernetes). Habitat isolation is Docker networks + loopback binds.
- Prowler / GuardDuty / Azure Defender (cloud-account products). Add only if that account is a declared engagement.
- Lambda/operator auto-remediation. Humans remediate; V grows; Guard blocks the next commit.
- Blocking `git push` on adapter Critical in v1. Promote into `V` first (same as Checkov).

## Evidence

Ledger: `runtime.status`, `runtime.scan`, `runtime.budget`, `runtime.chaos_dry`, `runtime.chaos_denied`, `runtime.chaos_injected`.
