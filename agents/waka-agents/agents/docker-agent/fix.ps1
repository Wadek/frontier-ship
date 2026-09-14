<#
.SYNOPSIS
    docker-agent/fix.ps1 — called by watch.ps1 when a container is bad.
    Invokes waka-bot (local 7B) to diagnose and fix. One-shot, then exits.

.PARAMETER ContainerName  Name of the crashed container
.PARAMETER Status         docker ps status string
#>

param(
    [Parameter(Mandatory)][string]$ContainerName,
    [string]$Status = "unknown"
)

$ErrorActionPreference = "SilentlyContinue"
. "$PSScriptRoot\..\..\config.ps1"

Write-AgentLog "docker-fix" "Fixing container: $ContainerName (status: $Status)" "WARN"

# Grab last 80 lines of container logs for context
$dockerLogs = & docker logs $ContainerName --tail 80 2>&1 | Out-String

$promptTemplate = Get-Content "$PROMPTS_DIR\docker-fix.txt" -Raw -Encoding UTF8
$prompt = $promptTemplate.Replace('{CONTAINER}', $ContainerName)
$prompt = $prompt.Replace('{STATUS}', $Status)
$prompt = $prompt.Replace('{LOGS}', $dockerLogs)

Write-AgentLog "docker-fix" "Invoking waka-bot for $ContainerName..." "INFO"

$output = & $WAKA_BOT -p $prompt --yolo --model $WAKA_MODEL 2>&1
$output | Add-Content -Path "$LOGS_DIR\docker-fix-$(Get-Date -Format 'yyyyMMdd').log" -Encoding UTF8

Write-AgentLog "docker-fix" "waka-bot fix attempt complete for $ContainerName." "INFO"
