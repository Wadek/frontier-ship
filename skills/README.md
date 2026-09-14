# Skills imported into frontier-ship

Wakalabs skill inventory, brought in so agents working through frontier-ship
can load them with `frontier skills show <name>`. Provenance is recorded per
set; the live copies on the habitat remain authoritative.

Use through the CLI:

```powershell
frontier skills                # list (name + one-line purpose)
frontier skills show dogfood   # print the skill body
```

## grok/ — the habitat skill pack (9 + 2 rules)

Provenance: `D:\wakalabs\grok skills\` (identical bytes also at
`C:\Users\waka\.grok\skills\`; 7 of 9 at `ai_learning\waka-cli\skills\`).
Rules: `C:\Users\waka\.grok\rules\`.

| Skill | Load when |
|---|---|
| `ai-learning` | persist/load lessons at `D:\wakalabs\ai_learning` |
| `cloudflare` | user wants their own WAF, not Cloudflare |
| `dogfood` | exercise the change as a user, then push (never skip Frontier plan/apply) |
| `inbox-watcher` | reuse the existing mailbox monitor + PowerShell hook; never poll |
| `local-first` | prefer installed local Ollama models; minimize cloud tokens |
| `long-running-background-tasks` | required before starting/watching any background job |
| `regular-git` | private repo, feature branch, confirm push; does not replace Frontier |
| `test-as-you-go` | tests same turn; resolve command via `.grok/test-on-stop` |
| `waka-cli` | build/operate the local coding CLI (`waka-cli` only) |
| `rules/local-build-pipeline.md` | the Frontier sequence + forbidden paths (Grok rule) |
| `rules/wakalabs-aware.md` | generated habitat snapshot (GPU, model, do-not-move list) |

## wakagym/ — gym.wakalabs.net coaching skills (4)

Provenance: `D:\wakalabs\opengym\.grok\skills\`.

| Skill | Purpose |
|---|---|
| `wakagym-400m` | CLUB 400 campaign doctrine (after the mile) |
| `wakagym-coach` | coaching doctrine for gym.wakalabs.net |
| `wakagym-hints` | hint phrasing for athletes |
| `wakagym-pros` | adapted pro sessions (+ `references/athletes.md`) |

## watermarks/ — hygiene service clients (2)

Provenance: `D:\wakalabs\watermarks-remover\skills\`. Thin clients over the
`127.0.0.1:8765` hygiene service; mirrors Frontier family H.

| Skill | Purpose |
|---|---|
| `clean-user-facing-text` | strip AI marks from user-facing text (+ `scripts/`) |
| `remove-ai-marks` | provenance-mark classes, vendors, removal matrix |

## github/ — VS Code delegation profiles (1)

Provenance: `D:\wakalabs\.github\skills\`.

| Skill | Purpose |
|---|---|
| `waka-local` | delegate coding/debug work to the local waka-bot bridge |
