<#
.SYNOPSIS
    monitor-agent.ps1 — collects a system health snapshot and passes it to
    waka-bot (local 7B model) for analysis and action.
    Run by Task Scheduler every 30 minutes. Exits when done. No polling.
#>

$ErrorActionPreference = "SilentlyContinue"
. "$PSScriptRoot\..\config.ps1"

Write-AgentLog "monitor-agent" "Starting snapshot collection" "INFO"

# ── Collect snapshot ──────────────────────────────────────────────────────

# Top CPU hogs
$topCPU = Get-Process |
    Sort-Object CPU -Descending |
    Select-Object -First 8 Name, @{N='RAM_MB';E={[math]::Round($_.WorkingSet/1MB,1)}}, @{N='CPU';E={[math]::Round($_.CPU,1)}} |
    ForEach-Object { "$($_.Name) cpu=$($_.CPU) ram=$($_.RAM_MB)MB" }

# RAM
$os      = Get-CimInstance Win32_OperatingSystem
$totalMB = [math]::Round($os.TotalVisibleMemorySize / 1024, 0)
$freeMB  = [math]::Round($os.FreePhysicalMemory     / 1024, 0)
$usedPct = [math]::Round(100 * (1 - $freeMB / $totalMB), 1)

# Disk
$diskC = Get-PSDrive C | Select-Object @{N='Free_GB';E={[math]::Round($_.Free/1GB,1)}},@{N='Used_GB';E={[math]::Round($_.Used/1GB,1)}}
$diskD = Get-PSDrive D -ErrorAction SilentlyContinue | Select-Object @{N='Free_GB';E={[math]::Round($_.Free/1GB,1)}},@{N='Used_GB';E={[math]::Round($_.Used/1GB,1)}}

# Ollama status
$ollamaOk = $false
try {
    $resp = Invoke-RestMethod -Uri "$OLLAMA_HOST/api/tags" -TimeoutSec 5 -ErrorAction Stop
    $ollamaModels = ($resp.models | ForEach-Object { $_.name }) -join ", "
    $ollamaOk = $true
} catch {
    $ollamaModels = "UNREACHABLE"
}

# Docker status
$dockerOk = $false
$dockerContainers = "unknown"
try {
    $dockerOut = & docker ps --format "{{.Names}}:{{.Status}}" 2>&1
    if ($LASTEXITCODE -eq 0) {
        $dockerOk = $true
        $dockerContainers = $dockerOut -join "; "
    } else {
        $dockerContainers = "DAEMON_DOWN"
    }
} catch {
    $dockerContainers = "DAEMON_DOWN"
}

# Bloat processes that crept back
$bloatBack = @($BLOAT_PROCESSES | Where-Object { Get-Process -Name $_ -ErrorAction SilentlyContinue })

# Build snapshot text
$snapshot = @"
=== waka-habitat monitor snapshot $(Get-Date -Format 'yyyy-MM-dd HH:mm') ===

RAM: ${usedPct}% used  (free ${freeMB} MB / total ${totalMB} MB)
Disk C: free=$($diskC.Free_GB)GB used=$($diskC.Used_GB)GB
Disk D: free=$($diskD.Free_GB)GB used=$($diskD.Used_GB)GB

Ollama: $(if($ollamaOk){"OK"}else{"UNREACHABLE"})
  Models: $ollamaModels

Docker: $(if($dockerOk){"OK"}else{"DAEMON_DOWN"})
  Containers: $dockerContainers

Top CPU processes:
$($topCPU -join "`n")

Bloat processes running (should be zero):
$(if($bloatBack){"  " + ($bloatBack -join ", ")}else{"  none"})
"@

Write-AgentLog "monitor-agent" "Snapshot collected. RAM ${usedPct}% | Docker $(if($dockerOk){'OK'}else{'DOWN'}) | Ollama $(if($ollamaOk){'OK'}else{'DOWN'})" "INFO"

# ── Thresholds: only invoke waka-bot if something needs attention ─────────
$needsAttention = (
    $usedPct -gt $RAM_WARN_PCT -or
    $diskC.Free_GB -lt $DISK_WARN_GB -or
    ($diskD -and $diskD.Free_GB -lt $DISK_WARN_GB) -or
    (-not $ollamaOk) -or
    (-not $dockerOk) -or
    $bloatBack.Count -gt 0
)

if (-not $needsAttention) {
    Write-AgentLog "monitor-agent" "All green. No model invocation needed." "INFO"
    Write-Host "monitor-agent: all green, skipping model call."
    exit 0
}

# ── Load prompt template and inject snapshot ───────────────────────────────
$promptTemplate = Get-Content "$PROMPTS_DIR\monitor.txt" -Raw -Encoding UTF8
$prompt = $promptTemplate.Replace('{SNAPSHOT}', $snapshot)

# Write snapshot to a temp file so we can pass long content safely
$tmpPrompt = "$LOGS_DIR\monitor-prompt-$(Get-Date -Format 'yyyyMMddHHmm').txt"
$prompt | Set-Content -Path $tmpPrompt -Encoding UTF8

Write-AgentLog "monitor-agent" "Invoking waka-bot for remediation..." "INFO"

# ── Call waka-bot (one-shot, exits when done) ─────────────────────────────
$output = & $WAKA_BOT -p (Get-Content $tmpPrompt -Raw) --yolo --model $WAKA_MODEL 2>&1
$output | Add-Content -Path "$LOGS_DIR\monitor-$(Get-Date -Format 'yyyyMMdd').log" -Encoding UTF8

Write-AgentLog "monitor-agent" "waka-bot run complete." "INFO"
Remove-Item $tmpPrompt -ErrorAction SilentlyContinue
