<#
.SYNOPSIS
    docker-agent/watch.ps1 — cheap always-on watcher for Docker containers.
    Checks health every 60 seconds. When a container is unhealthy or exited,
    calls fix.ps1 (which invokes waka-bot). No AI in this file — cost is zero.

.NOTES
    Registered as a Scheduled Task that restarts on failure.
    Does NOT poll Ollama or call any AI. The watcher itself is just a loop.
#>

$ErrorActionPreference = "SilentlyContinue"
. "$PSScriptRoot\..\..\config.ps1"

$FIX_SCRIPT   = "$PSScriptRoot\fix.ps1"
$COOL_DOWN    = @{}   # container -> last fix time, prevent fix-storm
$COOL_MIN     = 10    # minutes between fixes for the same container

Write-AgentLog "docker-watch" "Docker watcher started. Watching: $($DOCKER_WATCH -join ', ')" "INFO"

while ($true) {
    Start-Sleep -Seconds 60

    # Check daemon
    $daemonUp = $false
    try {
        $null = & docker info 2>&1
        $daemonUp = ($LASTEXITCODE -eq 0)
    } catch {}

    if (-not $daemonUp) {
        Write-AgentLog "docker-watch" "Docker daemon unreachable - skipping container checks" "WARN"
        continue
    }

    # Get all running/exited container statuses
    $raw = & docker ps -a --format '{{.Names}}|{{.Status}}' 2>&1
    if ($LASTEXITCODE -ne 0) { continue }

    $statuses = @{}
    foreach ($line in $raw) {
        $parts = $line -split '\|', 2
        if ($parts.Count -eq 2) { $statuses[$parts[0]] = $parts[1] }
    }

    foreach ($name in $DOCKER_WATCH) {
        $status = $statuses[$name]
        if (-not $status) { continue }   # container not defined yet, skip

        $isBad = (
            $status -match '^Exited' -or
            $status -match '\(unhealthy\)' -or
            $status -match '^Restarting'
        )

        if ($isBad) {
            # Cool-down: don't spam fix for the same container
            $lastFix = $COOL_DOWN[$name]
            if ($lastFix -and (Get-Date) -lt $lastFix.AddMinutes($COOL_MIN)) {
                Write-AgentLog "docker-watch" "Container $name bad ($status) - in cool-down, skipping" "INFO"
                continue
            }

            Write-AgentLog "docker-watch" "Container $name bad ($status) - invoking fix.ps1" "WARN"
            $COOL_DOWN[$name] = Get-Date

            & $FIX_SCRIPT -ContainerName $name -Status $status
        }
    }
}
