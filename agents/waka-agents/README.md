# waka-agents

Local agent team for Wade's habitat. **No API keys. No cloud. Runs on the local 7B model via waka-bot.**

## Agents

| Agent | Script | Schedule | AI cost |
|---|---|---|---|
| **bloat-killer** | `agents/bloat-killer.ps1` | Manual / on-demand | None (pure PowerShell) |
| **monitor-agent** | `agents/monitor-agent.ps1` | Every 30 min | Only when thresholds breached |
| **docker-agent** | `agents/docker-agent/watch.ps1` | Always-on at logon | Only on container crash |
| **security-agent** | `agents/security-agent/watch.ps1` | Always-on at logon | Only on WAF anomaly |

## Quick start

### Use from VS Code Chat

Select **Waka Local** in the Chat agent picker. It delegates implementation and
debugging tasks to the local Ollama-backed `waka-bot` through
`invoke-local.ps1`, using the selected workspace path as its working directory.
It does not use a cloud API. The custom agent profile lives at
`.github/agents/waka-local.agent.md` and its reusable skill at
`.github/skills/waka-local/SKILL.md`.

The bridge resumes the latest local session for the target working directory.
Tool failures, failed tests and commands, and Ollama errors are captured in the
session JSONL and become context for the next Waka Local turn, regardless of
which client launched it.

```powershell
# Install scheduled tasks (once):
powershell -ExecutionPolicy Bypass -File D:\wakalabs\waka-agents\install.ps1

# Run bloat-killer immediately:
powershell -ExecutionPolicy Bypass -File D:\wakalabs\waka-agents\agents\bloat-killer.ps1

# Run bloat-killer as admin (to also permanently disable services):
Start-Process powershell -Verb RunAs -ArgumentList '-ExecutionPolicy Bypass -File D:\wakalabs\waka-agents\agents\bloat-killer.ps1'

# Start the always-on watchers now (without logging off):
Start-ScheduledTask -TaskName WakaDockerAgent
Start-ScheduledTask -TaskName WakaSecurityAgent

# Trigger monitor-agent manually:
powershell -ExecutionPolicy Bypass -File D:\wakalabs\waka-agents\agents\monitor-agent.ps1

# View logs:
Get-Content D:\wakalabs\waka-agents\logs\monitor-$(Get-Date -Format 'yyyyMMdd').log
```

## Bloat list (confirmed killed)

- `iCloudHome`, `ApplePhotoStreams`, `iCloudCKKS` — iCloud suite
- `gamingservices`, `gamingservicesnet`, `GameInputRedistService`, `GameInputSvc` — Xbox/Gaming Services
- `AsusUpdateCheck` — ASUS update nag
- `OneApp.IGCC.WinService` — Intel Graphics Command Center (NVIDIA machine)
- `jhi_service` — Intel Dynamic App Loader
- `DiagTrack` — Windows telemetry
- `DoSvc` — Delivery Optimization (P2P Windows Update)
- NVIDIA GFE telemetry tasks (4x crash report, self-update nag)
- Ubisoft Connect background update task

Tailscale is intentionally not managed until its use on this machine is confirmed.

## Architecture pattern

```
Cheap always-on watcher (PowerShell loop, no AI)
        │
        │ condition met (crash / anomaly / threshold)
        ↓
  waka-bot -p --yolo "<prompt>" --model waka-coder
        │
        │ 7B model uses tools to diagnose + act
        ↓
  exits. Logs result. No polling.
```

This follows the habitat's "no poll" doctrine from `ai_learning/README.md §0`.

## File layout

```
waka-agents/
  config.ps1                     shared bloat lists, thresholds, logging helper
  install.ps1                    registers Scheduled Tasks
  README.md                      this file
  agents/
    bloat-killer.ps1             kill + disable known bloat
    monitor-agent.ps1            30-min health snapshot → waka-bot
    docker-agent/
      watch.ps1                  60s container health loop (no AI)
      fix.ps1                    one-shot fix via waka-bot
    security-agent/
      watch.ps1                  WAF log tailer (no AI)
      report.ps1                 one-shot threat analysis via waka-bot
  prompts/
    monitor.txt                  prompt template for monitor-agent
    docker-fix.txt               prompt template for docker fix
    security-report.txt          prompt template for security report
  logs/                          all agent run logs (YYYY-MM-DD per agent)
```

## Disarming an agent

```powershell
# Pause monitor:
Disable-ScheduledTask -TaskName WakaMonitorAgent

# Stop docker watcher:
Stop-ScheduledTask -TaskName WakaDockerAgent

# Re-enable:
Enable-ScheduledTask -TaskName WakaMonitorAgent
Start-ScheduledTask -TaskName WakaDockerAgent
```

## Constraints

- **No API keys.** All AI is local Ollama (`waka-coder` / `qwen3-coder-7b` on RTX 2080 8GB).
- **No polling.** Watchers are cheap loops; the 7B runs only when needed.
- **YOLO.** Agents use `waka-bot --yolo`. The prompts encode the same guardrails as `train.md`.
- **Protected paths never touched:** `D:\immich-photos`, `D:\immich_backups`, `~\.grok\auth.json`.
