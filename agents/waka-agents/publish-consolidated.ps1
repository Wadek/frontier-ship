<#!
.SYNOPSIS
    Commit and push the canonical D:\wakalabs\waka-bot tree.
#>

$ErrorActionPreference = "Stop"
$repoRoot = "D:\wakalabs\waka-bot"
$branch = "frontier/consolidate-waka-bot"
$remoteUrl = "https://github.com/Wadek/wakabot.git"

function Invoke-Checked {
    param([string]$File, [string[]]$Arguments)
    $old = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try { $output = & $File @Arguments 2>&1 }
    finally { $ErrorActionPreference = $old }
    $code = $LASTEXITCODE
    $output | ForEach-Object { Write-Host $_ }
    if ($code -ne 0) { throw "$File failed with exit code $code" }
}

Push-Location $repoRoot
try {
    if (-not (Test-Path -LiteralPath "$repoRoot\.git")) {
        Invoke-Checked "git" @("init", "-b", $branch)
    } else {
        $current = git branch --show-current
        if ($current -ne $branch) {
            & git show-ref --verify --quiet "refs/heads/$branch"
            if ($LASTEXITCODE -eq 0) { Invoke-Checked "git" @("switch", $branch) }
            else { Invoke-Checked "git" @("switch", "-c", $branch) }
        }
    }

    $old = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
        $remote = git remote get-url origin 2>$null
        $remoteCode = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $old
    }
    if ($remoteCode -ne 0) { Invoke-Checked "git" @("remote", "add", "origin", $remoteUrl) }

    $tokens = $null
    $parseErrors = $null
    Get-ChildItem -Path $repoRoot -Recurse -Filter *.ps1 | ForEach-Object {
        [System.Management.Automation.Language.Parser]::ParseFile($_.FullName, [ref]$tokens, [ref]$parseErrors) | Out-Null
        if ($parseErrors.Count -gt 0) { throw "$($_.FullName) has PowerShell syntax errors" }
    }
    Push-Location "$repoRoot\cli"
    try { Invoke-Checked "python" @("-m", "unittest", "discover", "-s", "tests", "-v") }
    finally { Pop-Location }

    Get-ChildItem -Path $repoRoot -Recurse -Directory -Filter __pycache__ | Remove-Item -Recurse -Force -ErrorAction SilentlyContinue
    Get-ChildItem -Path $repoRoot -Recurse -File -Filter *.pyc | Remove-Item -Force -ErrorAction SilentlyContinue

    Invoke-Checked "git" @("add", "-A")
    git diff --cached --quiet
    $hasStagedChanges = $LASTEXITCODE -ne 0
    if (-not $hasStagedChanges) {
        Write-Host "No staged changes; pushing existing canonical commit."
    } else {
        Invoke-Checked "git" @("commit", "-m", "Consolidate waka-bot runtime and agents")
    }
    Invoke-Checked "frontier" @("hygiene")
    Invoke-Checked "frontier" @("plan")
    Invoke-Checked "frontier" @("apply")
    Invoke-Checked "git" @("push", "-u", "origin", "HEAD")
    Write-Host "=== verified canonical publish ==="
    Invoke-Checked "git" @("log", "-1", "--oneline")
    Invoke-Checked "git" @("remote", "-v")
    Invoke-Checked "git" @("ls-remote", "--heads", "origin", $branch)
} finally {
    Pop-Location
}