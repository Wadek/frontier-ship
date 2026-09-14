<#
.SYNOPSIS
    security-agent/report.ps1 — called by watch.ps1 on WAF anomaly.
    Invokes waka-bot to analyse the threat and write a markdown alert.
    One-shot, exits after.
#>

param(
    [string]$Excerpt = "(no excerpt)",
    [string]$Pattern = "(unknown)"
)

$ErrorActionPreference = "SilentlyContinue"
. "$PSScriptRoot\..\..\config.ps1"

Write-AgentLog "security-report" "Anomaly: $Pattern" "WARN"

# Grab last 5 minutes of WAF logs for full context
$recentLogs = & docker logs waka-waf --since 5m 2>&1 | Out-String

$promptTemplate = Get-Content "$PROMPTS_DIR\security-report.txt" -Raw -Encoding UTF8
$prompt = $promptTemplate.Replace('{PATTERN}', $Pattern)
$prompt = $prompt.Replace('{EXCERPT}', $Excerpt)
$prompt = $prompt.Replace('{LOGS}', $recentLogs)

Write-AgentLog "security-report" "Invoking waka-bot for threat analysis..." "INFO"

$output = & $WAKA_BOT -p $prompt --yolo --model $WAKA_MODEL 2>&1
$output | Add-Content -Path "$LOGS_DIR\security-$(Get-Date -Format 'yyyyMMdd').log" -Encoding UTF8

# Write a markdown alert file for easy review
$alertFile = "$LOGS_DIR\security-alert-$(Get-Date -Format 'yyyyMMdd-HHmmss').md"
@"
# Security Alert — $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')

**Pattern:** $Pattern
**Excerpt:** $Excerpt

## waka-bot Analysis

$($output -join "`n")
"@ | Set-Content -Path $alertFile -Encoding UTF8

Write-AgentLog "security-report" "Alert written to $alertFile" "INFO"
