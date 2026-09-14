<#
.SYNOPSIS
    install.ps1 — registers waka-agents as Windows Scheduled Tasks.
    Run once. Re-running is idempotent (unregisters + re-registers).

.NOTES
    - monitor-agent: every 30 min, current user scope (no elevation needed)
    - docker-agent watcher: at logon, current user scope, restarts on failure
    - security-agent watcher: at logon, current user scope, restarts on failure
    - bloat-killer: run manually or add here — requires elevation for service disable
#>

$ErrorActionPreference = "Stop"
. "$PSScriptRoot\config.ps1"

$PS = "powershell.exe"
$ARGS_PREFIX = "-NoProfile -ExecutionPolicy Bypass -File"
$USER = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name

function Register-WakaTask {
    param(
        [string]$Name,
        [string]$Script,
        [object]$Trigger,
        [string]$Description
    )
    # Remove existing if any
    Unregister-ScheduledTask -TaskName $Name -Confirm:$false -ErrorAction SilentlyContinue

    $action  = New-ScheduledTaskAction -Execute $PS -Argument "$ARGS_PREFIX `"$Script`""
    $settings = New-ScheduledTaskSettingsSet `
        -ExecutionTimeLimit ([TimeSpan]::Zero) `
        -RestartCount 5 `
        -RestartInterval (New-TimeSpan -Minutes 2) `
        -StartWhenAvailable

    Register-ScheduledTask `
        -TaskName    $Name `
        -Action      $action `
        -Trigger     $Trigger `
        -Settings    $settings `
        -RunLevel    Limited `
        -Description $Description `
        -Force | Out-Null

    Write-Host "  registered: $Name" -ForegroundColor Green
    Write-AgentLog "install" "Registered task: $Name" "INFO"
}

Write-Host "`n=== waka-agents installer ===" -ForegroundColor Cyan

# 1. monitor-agent — every 30 minutes
$monitorTrigger = New-ScheduledTaskTrigger -RepetitionInterval (New-TimeSpan -Minutes 30) -Once -At (Get-Date)
Register-WakaTask `
    -Name        "WakaMonitorAgent" `
    -Script      "$WAKA_AGENTS\agents\monitor-agent.ps1" `
    -Trigger     $monitorTrigger `
    -Description "waka-agents: system health check every 30 min. Invokes waka-bot only when thresholds breached."

# 2. docker-agent watcher — at logon
$dockerTrigger = New-ScheduledTaskTrigger -AtLogOn
Register-WakaTask `
    -Name        "WakaDockerAgent" `
    -Script      "$WAKA_AGENTS\agents\docker-agent\watch.ps1" `
    -Trigger     $dockerTrigger `
    -Description "waka-agents: watches Docker containers every 60s. Calls fix.ps1 on crash."

# 3. security-agent watcher — at logon
$secTrigger = New-ScheduledTaskTrigger -AtLogOn
Register-WakaTask `
    -Name        "WakaSecurityAgent" `
    -Script      "$WAKA_AGENTS\agents\security-agent\watch.ps1" `
    -Trigger     $secTrigger `
    -Description "waka-agents: tails waka-waf logs, triggers report.ps1 on anomaly."

Write-Host @"

=== Install complete ===
  Tasks registered:
    WakaMonitorAgent   — every 30 min
    WakaDockerAgent    — at logon (always-on)
    WakaSecurityAgent  — at logon (always-on)

  Logs: $LOGS_DIR
  Run bloat-killer now:
    powershell -ExecutionPolicy Bypass -File "$WAKA_AGENTS\agents\bloat-killer.ps1"
  Or as admin (to also disable services):
    Start-Process powershell -Verb RunAs -ArgumentList '-File "$WAKA_AGENTS\agents\bloat-killer.ps1"'

  Start docker/security watchers now (without logoff):
    Start-ScheduledTask -TaskName WakaDockerAgent
    Start-ScheduledTask -TaskName WakaSecurityAgent
"@ -ForegroundColor Cyan
