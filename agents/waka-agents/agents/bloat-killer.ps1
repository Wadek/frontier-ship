<#
.SYNOPSIS
    bloat-killer.ps1 — kills and permanently disables known bloat on Wade's habitat.
    Run once on demand. Safe to re-run (idempotent).
    No model, no API key. Pure PowerShell.

.NOTES
    Some service changes require elevation. Run as admin for full effect,
    or accept that process-kills work but service-disable may be skipped.
#>

$ErrorActionPreference = "SilentlyContinue"
. "$PSScriptRoot\..\config.ps1"

$killed   = [System.Collections.Generic.List[string]]::new()
$disabled = [System.Collections.Generic.List[string]]::new()
$skipped  = [System.Collections.Generic.List[string]]::new()
$ramFreed = 0

Write-AgentLog "bloat-killer" "Starting bloat-killer run" "INFO"

# ── 1. Kill processes ─────────────────────────────────────────────────────
Write-Host "`n[1/3] Killing bloat processes..." -ForegroundColor Cyan

foreach ($name in $BLOAT_PROCESSES) {
    $procs = Get-Process -Name $name -ErrorAction SilentlyContinue
    if ($procs) {
        $mb = [math]::Round(($procs | Measure-Object WorkingSet -Sum).Sum / 1MB, 1)
        $procs | Stop-Process -Force -ErrorAction SilentlyContinue
        $killed.Add("$name (${mb} MB)")
        $ramFreed += $mb
        Write-Host "  killed  $name  (-${mb} MB)" -ForegroundColor Green
    }
}

# ── 2. Disable services ───────────────────────────────────────────────────
Write-Host "`n[2/3] Disabling bloat services..." -ForegroundColor Cyan

$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole(
    [Security.Principal.WindowsBuiltInRole]::Administrator
)

foreach ($svc in $BLOAT_SERVICES) {
    $s = Get-Service -Name $svc.Name -ErrorAction SilentlyContinue
    if (-not $s) {
        # Try partial match (some iCloud services have GUIDs appended)
        $s = Get-Service | Where-Object { $_.Name -like "*$($svc.Name)*" } | Select-Object -First 1
    }
    if ($s) {
        if ($s.Status -eq "Running") {
            Stop-Service -Name $s.Name -Force -ErrorAction SilentlyContinue
        }
        if ($isAdmin) {
            Set-Service -Name $s.Name -StartupType Disabled -ErrorAction SilentlyContinue
            $disabled.Add($svc.Display)
            Write-Host "  disabled  $($svc.Display)" -ForegroundColor Green
        } else {
            # Non-admin: stop only, leave startup type (requires elevation to change)
            $skipped.Add("$($svc.Display) [needs admin to disable permanently]")
            Write-Host "  stopped (not admin - re-run as admin to disable)  $($svc.Display)" -ForegroundColor Yellow
        }
    }
}

# ── 3. Disable scheduled tasks ────────────────────────────────────────────
Write-Host "`n[3/3] Disabling bloat scheduled tasks..." -ForegroundColor Cyan

foreach ($t in $BLOAT_TASKS) {
    try {
        $task = Get-ScheduledTask -TaskPath $t.Path -TaskName $t.Name -ErrorAction SilentlyContinue
        if ($task -and $task.State -ne "Disabled") {
            Disable-ScheduledTask -TaskPath $t.Path -TaskName $t.Name -ErrorAction Stop | Out-Null
            $disabled.Add("Task: $($t.Name)")
            Write-Host "  disabled task  $($t.Name)" -ForegroundColor Green
        } elseif ($task) {
            Write-Host "  already disabled  $($t.Name)" -ForegroundColor DarkGray
        }
    } catch {
        $skipped.Add("Task: $($t.Name) [$_]")
        Write-Host "  skipped task  $($t.Name): $_" -ForegroundColor Yellow
    }
}

# ── Summary ────────────────────────────────────────────────────────────────
$summary = @"

=== bloat-killer summary ===
  RAM freed (this run): ${ramFreed} MB
  Processes killed    : $($killed.Count)  — $($killed -join ", ")
  Services disabled   : $($disabled.Count)  — $($disabled -join ", ")
  Skipped (needs admin): $($skipped.Count)  — $($skipped -join ", ")
"@

Write-Host $summary -ForegroundColor Cyan
Write-AgentLog "bloat-killer" "RAM freed: ${ramFreed}MB | killed: $($killed -join ',') | disabled: $($disabled -join ',')" "INFO"

if ($skipped.Count -gt 0 -and -not $isAdmin) {
    Write-Host "`n  Re-run as Administrator to permanently disable services:" -ForegroundColor Yellow
    Write-Host "  Start-Process powershell -Verb RunAs -ArgumentList '-File $PSCommandPath'" -ForegroundColor Yellow
}
