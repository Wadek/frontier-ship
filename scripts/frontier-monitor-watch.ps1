<#
.SYNOPSIS
    frontier-monitor-watch.ps1 — event-driven watcher over the Frontier
    ledgers. Whenever an agent appends evidence (plan/apply/push/…), this
    runs `frontier monitor all` once and prints one line per examined event.

    Cheap watcher, one-shot work, no polling loop of the model — the same
    doctrine as the waka-agents team. Logs to D:\frontier\ledgers\monitor\watch.log.

.PARAMETER Path
    Ledger root to watch (default: D:\frontier\ledgers).

.PARAMETER Once
    Run a single sweep and exit (no watcher).

.PARAMETER FrontierExe
    frontier.exe to invoke (default: D:\frontier\bin\frontier.exe).

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File scripts\frontier-monitor-watch.ps1

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File scripts\frontier-monitor-watch.ps1 -Once
#>
param(
    [string]$Path = "D:\frontier\ledgers",
    [switch]$Once,
    [string]$FrontierExe = "D:\frontier\bin\frontier.exe"
)

$ErrorActionPreference = "SilentlyContinue"
$LogDir = Join-Path $Path "monitor"
New-Item -ItemType Directory -Path $LogDir -Force | Out-Null
$LogFile = Join-Path $LogDir "watch.log"

function Write-WatchLog([string]$msg) {
    $line = "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')  $msg"
    Write-Host $line
    Add-Content -Path $LogFile -Value $line -Encoding UTF8
}

function Invoke-Sweep {
    if (-not (Test-Path $FrontierExe)) {
        Write-WatchLog "MONITOR frontier.exe missing at $FrontierExe"
        return
    }
    $out = & $FrontierExe monitor all 2>&1 | Out-String
    $verdict = ($out -split "`n" | Select-String -Pattern '^verdict:' | Select-Object -Last 1)
    Write-WatchLog ("MONITOR sweep: {0}" -f ($verdict -replace '\s+', ' ').Trim())
}

if ($Once) {
    Invoke-Sweep
    exit 0
}

if (-not (Test-Path $Path)) {
    Write-WatchLog "MONITOR ledger root missing: $Path (nothing to watch yet)"
    Write-WatchLog "MONITOR watcher armed — it will pick up events when ledgers exist"
}

$watcher = New-Object System.IO.FileSystemWatcher
$watcher.Path = $Path
$watcher.Filter = "ledger.jsonl"
$watcher.IncludeSubdirectories = $true
$watcher.NotifyFilter = [System.IO.NotifyFilters]::LastWrite -bor
                        [System.IO.NotifyFilters]::FileName -bor
                        [System.IO.NotifyFilters]::CreationTime

# Debounce: one sweep per burst of events (agents seal several rows per run).
$lastSweep = [DateTime]::MinValue
$script:sweeping = $false

$onEvent = {
    if ($script:sweeping) { return }
    $now = Get-Date
    if (($now - $lastSweep).TotalSeconds -lt 3) { return }
    $lastSweep = $now
    $script:sweeping = $true
    try { Invoke-Sweep } finally { $script:sweeping = $false }
}

Register-ObjectEvent -InputObject $watcher -EventName Changed -Action $onEvent | Out-Null
Register-ObjectEvent -InputObject $watcher -EventName Created -Action $onEvent | Out-Null
$watcher.EnableRaisingEvents = $true

Write-WatchLog "MONITOR watcher armed on $Path (Ctrl+C to stop)"
try {
    Wait-Event
} finally {
    $watcher.EnableRaisingEvents = $false
    $watcher.Dispose()
    Get-EventSubscriber | Unregister-Event -ErrorAction SilentlyContinue
    Write-WatchLog "MONITOR watcher stopped"
}
