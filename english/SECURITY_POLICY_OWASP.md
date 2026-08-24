# Security as policy — OWASP Top 10 (first version)

This is the **human** policy document.  
It becomes **Haskell** definitions in `V` (vuln set).  
At `git frontier gate` / push, Go runs those checks against the change-set.

`V` starts from these named rules. Empty matches ⇒ clean under current `V`.

---

## Scope

Applies to the **union of lines in the proposed change** (`C`), at maximum / at least once (F4).

## OWASP Top 10 (2021) → Frontier checks (v0)

| ID | OWASP | Policy (simple English) | Check id |
|----|-------|-------------------------|----------|
| A01 | Broken Access Control | Do not commit hardcoded admin bypasses or `if user == "admin"` style auth skips in app code | `owasp.a01.hardcoded_auth_bypass` |
| A02 | Cryptographic Failures | Do not commit private keys, `BEGIN RSA PRIVATE KEY`, or cleartext password assignments | `owasp.a02.secret_material` |
| A03 | Injection | Do not build SQL/shell with raw string concat of request input (`"SELECT ..." +`, `os.system(`, `eval(`) | `owasp.a03.injection_sink` |
| A04 | Insecure Design | Do not disable security flags in-repo (`VERIFY_NONE`, `InsecureSkipVerify`, `csrf=False`) | `owasp.a04.insecure_flag` |
| A05 | Security Misconfiguration | Do not commit `.env` with secrets, or `debug=True` / `APP_DEBUG=true` defaults for prod paths | `owasp.a05.misconfig` |
| A06 | Vulnerable Components | Do not pin obviously wild `latest` for critical base images without digest in Dockerfiles we own | `owasp.a06.unpinned_latest` |
| A07 | Auth Failures | Do not hardcode session secrets / JWT secrets / `password = "` literals | `owasp.a07.hardcoded_credential` |
| A08 | Software/Data Integrity | Do not curl \| bash installers in scripts we add | `owasp.a08.curl_bash` |
| A09 | Logging/Monitoring Failures | Do not log raw passwords or `Authorization:` headers | `owasp.a09.sensitive_log` |
| A10 | SSRF | Do not fetch user-controlled URLs with open redirects to metadata IPs without allowlist comments | `owasp.a10.open_url_fetch` |

v0 is **one fast regex per Top 10 class** (2021), not a “Top 100.” There is no OWASP Top 100; depth lives in ASVS / CWE and in **adapters** (Checkov, Gitleaks, Trivy) that run after the regex pass.

Severity in Go today: **Critical** A02; **High** A01/A03/A07/A08; **Medium** A04/A05/A06/A09/A10.  
Untriaged **built-in** High/Critical under `V` blocks plan/apply (F4). Adapter Critical is **advise** until promoted into this chain.

## Translation chain

```
  english/SECURITY_POLICY_OWASP.md
            │
            v
  haskell/src/Frontier/OWASP.hs     (named rules in V)
            │
            v
  Go internal/owasp (pattern scan on C)
            │
            v
  git frontier gate  →  ledger: exam.owasp + gate.passed|failed
```
