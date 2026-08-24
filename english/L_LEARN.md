# Learn (L) — Landscape

**Learn** is a policy family alongside **Guard (G)** and **Slim (S)**.

| Word | Letter | Job |
|------|--------|-----|
| `frontier learn` | L | Ingest + classify before change |
| `frontier guard` | G | Security (OWASP, secret surfaces, adapters) |
| `frontier slim` | S | Vibe-bloat reduction (planned) |

## Purpose

Ingest and classify a **single project** before changing it. Learn is the first phase of Slim and of any habitat/architecture review.

```
  learn classify  →  map (kind, compose, langs, topology)
  guard           →  security exam + secret path names
  slim            →  bloat using Learn + (later) coverage   [planned]
```

## Commands

| Command | What |
|---------|------|
| `frontier learn` / `frontier learn classify` | Classify cwd; write `.frontier/learn/L-*`; seal `learn.classified` |
| `frontier learn classify PATH` | Classify that project root |
| `frontier learn status` / `frontier L status` | Recent `learn.*` ledger rows |

## Kinds (v0)

`app_compose` · `network_edge` · `media_stack` · `tooling` · `data_volume` · `skills_pack` · `compose_project` · `app_repo` · `unknown`

## Rules

- Read-only: Learn does not move/delete files.
- **Security belongs in Guard** — secret surfaces (`.env`, keys, …) are not part of Learn.
- Run **per project**, not on an umbrella folder of many apps.
