# Agents imported into frontier-ship

Wakalabs agent inventory, brought in so agents working through frontier-ship
can load them with `frontier agents show <name>`. Provenance recorded per set.

Use through the CLI:

```powershell
frontier agents                     # list (name + one-line purpose)
frontier agents show monitor-agent  # print the agent definition
```

## waka-agents/ — the local agent team (no API keys)

Provenance: `D:\wakalabs\waka-agents\`. Pattern: cheap always-on PowerShell
watcher → condition met → `waka-bot -p --yolo "<prompt>"` → exits. No polling,
no cloud. The same doctrine the Frontier monitor watcher follows.

| Agent | Script | Schedule | AI cost |
|---|---|---|---|
| `bloat-killer` | `agents/bloat-killer.ps1` | Manual / on-demand | None (pure PowerShell) |
| `monitor-agent` | `agents/monitor-agent.ps1` | Every 30 min | Only when thresholds breached |
| `docker-agent` | `agents/docker-agent/watch.ps1` | Always-on at logon | Only on container crash |
| `security-agent` | `agents/security-agent/watch.ps1` | Always-on at logon | Only on WAF anomaly |

Also imported: `config.ps1` (shared thresholds + logging), `install.ps1`
(scheduled-task registration), the workflow scripts (`audit-branches`,
`bootstrap-repo`, `create-prs`, `pr-workflow`, `repair-main`, `invoke-local`,
…), and the `prompts/` templates (`local-system.md` is the authoritative
habitat briefing injected into every local agent turn).

## github/ — VS Code / GitHub delegation profiles

Provenance: `D:\wakalabs\.github\`.

| Profile | Purpose |
|---|---|
| `git-agent` | Git/GitHub specialist: Frontier flow, PRs, history repair, project management |
| `waka-local` | bridge that delegates implementation to the local waka-bot |
| `waka-local-routing` | instruction: route Git/GitHub/Docker work through Waka Local |

## Relationship to the monitor

`frontier monitor` verifies the machine-checkable subset of the rules these
agents are told to obey: feature branches (D2), plan → apply → push with fresh
seals (D1), no `FRONTIER_SOFT=1` (D3), no blocked ship (D4), and denied
attempts surfaced as incidents (D5).
