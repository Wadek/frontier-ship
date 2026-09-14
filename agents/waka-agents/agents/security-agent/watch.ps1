<#
.SYNOPSIS
    security-agent/watch.ps1 — tails the WAF (waka-waf) container log in
    real-time. No AI. Calls report.ps1 only when a suspicious pattern is
    detected. Always-on, registered as a Scheduled Task.
#>

$ErrorActionPreference = "SilentlyContinue"
. "$PSScriptRoot\..\..\config.ps1"

$REPORT_SCRIPT = "$PSScriptRoot\report.ps1"
$COOL_DOWN     = @{}   # ip -> last report time
$COOL_MIN      = 15

Write-AgentLog "security-watch" "Security watcher started. Tailing waka-waf logs." "INFO"

# Track IP hit counts in the current rolling window
$ipHits    = @{}
$windowStart = Get-Date

function Reset-Window {
    $script:ipHits    = @{}
    $script:windowStart = Get-Date
}

# Stream docker logs -f (follows, never exits unless container stops)
$proc = $null
while ($true) {
    # Check if waf container is up
    $wafUp = $false
    try {
        $info = & docker inspect waka-waf 2>&1 | ConvertFrom-Json -ErrorAction Stop
        $wafUp = $info[0].State.Running
    } catch {}

    if (-not $wafUp) {
        Write-AgentLog "security-watch" "waka-waf not running - waiting 60s" "WARN"
        Start-Sleep -Seconds 60
        continue
    }

    # Start streaming log process
    $tmpLog = "$LOGS_DIR\waf-stream-$(Get-Date -Format 'yyyyMMddHHmm').tmp"
    $proc   = Start-Process docker -ArgumentList "logs waka-waf -f --since 1m" `
                -RedirectStandardOutput $tmpLog -RedirectStandardError "$tmpLog.err" `
                -NoNewWindow -PassThru

    Write-AgentLog "security-watch" "Streaming waka-waf logs (pid $($proc.Id))" "INFO"

    # Tail the file line by line
    $pos = 0
    $reader = $null
    try {
        Start-Sleep -Seconds 2   # let the file appear
        $reader = [System.IO.StreamReader]::new(
            $tmpLog,
            [System.Text.Encoding]::UTF8,
            $true,
            4096,
            $false
        )

        while (-not $proc.HasExited) {
            $line = $reader.ReadLine()
            if ($null -eq $line) {
                Start-Sleep -Milliseconds 500
                # Roll the window every $WAF_RATE_WINDOW_SEC seconds
                if (((Get-Date) - $windowStart).TotalSeconds -gt $WAF_RATE_WINDOW_SEC) {
                    Reset-Window
                }
                continue
            }

            # ── Pattern match ──────────────────────────────────────────
            $matched = $false
            $matchedPattern = ""
            foreach ($pat in $WAF_CRITICAL_PATTERNS) {
                if ($line -match $pat) {
                    $matched = $true
                    $matchedPattern = $pat
                    break
                }
            }

            # ── Rate limiting by client IP ─────────────────────────────
            $ip = ""
            if ($line -match '\b(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})\b') {
                $ip = $matches[1]
                $ipHits[$ip] = ($ipHits[$ip] -as [int]) + 1
                if ($ipHits[$ip] -ge $WAF_RATE_THRESHOLD) {
                    $matched = $true
                    $matchedPattern = "rate:$WAF_RATE_THRESHOLD hits from $ip"
                }
            }

            if ($matched) {
                # Cool-down per IP
                $key = if ($ip) { $ip } else { "general" }
                $lastReport = $COOL_DOWN[$key]
                if ($lastReport -and (Get-Date) -lt $lastReport.AddMinutes($COOL_MIN)) {
                    continue
                }
                $COOL_DOWN[$key] = Get-Date

                Write-AgentLog "security-watch" "ANOMALY detected: $matchedPattern | line: $($line.Substring(0,[math]::Min(200,$line.Length)))" "WARN"

                # Grab context lines
                $excerpt = $line

                & $REPORT_SCRIPT -Excerpt $excerpt -Pattern $matchedPattern
            }
        }
    } finally {
        if ($reader) { $reader.Close() }
        if (-not $proc.HasExited) { $proc.Kill() }
        Remove-Item $tmpLog, "$tmpLog.err" -ErrorAction SilentlyContinue
    }

    Write-AgentLog "security-watch" "Log stream ended. Restarting in 10s." "INFO"
    Start-Sleep -Seconds 10
}
