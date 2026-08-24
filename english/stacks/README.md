# Stack playbooks

**Contextual, repeatable recipes** for Learn → Guard → Slim → Optimize on a given technology shape.

The **standard** process lives in [`../O_VERIFY.md`](../O_VERIFY.md) (equivalence + browser + visibility + GitHub Actions).  
Each playbook answers: *how do we run that standard on **this** stack?*

## When to add a playbook

After the first successful Opt cycle on a new stack, write or extend a playbook so the next similar repo does not re-learn from scratch.

## Index

| Playbook | Matches (Learn hints) |
|----------|------------------------|
| [python-web-static.md](python-web-static.md) | `app_compose`, Python HTTP + static HTML/JS |
| [habitat-edge.md](habitat-edge.md) | cloudflared + a private Docker network + public hostnames |
| [go-http.md](go-http.md) | Go `net/http` / chi / echo APIs *(stub — fill on first Go Opt)* |
| [node-express.md](node-express.md) | Node Express/Fastify + SPA *(stub — fill on first Node Opt)* |

## Playbook template (copy for a new stack)

```markdown
# Stack: <name>

## Detect
- File/layout signals Learn should notice
- Example remotes

## Equivalence (Layer A)
- Commands (verbose)
- What must stay equal

## Browser (Layer B)
- Tool (default Playwright for web)
- Smoke flows
- Env vars (never commit secrets)

## GitHub Actions
- Workflow path / job name
- Same commands as local

## Pitfalls learned
- Fixture name clashes, path quirks, etc.

## Optimize notes
- Typical Opt smells for this stack
```

## Relation to Learn

`frontier learn` classifies **kind**. Later, Learn (or a human) picks the playbook id and stores it under `.frontier/optimize/stack.md` pointing here — so Optimize verify stays contextual.
