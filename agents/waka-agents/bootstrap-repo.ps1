<#!
.SYNOPSIS
    Initialize and publish waka-agents as Wadek/wakabot.

.DESCRIPTION
    Deterministic GitHub bootstrap used by the Waka Local bridge. It never
    pushes main/master and stops on the first failed command.
#>

$ErrorActionPreference = "Stop"
$repoRoot = $PSScriptRoot
$branch = "frontier/bootstrap-waka-agents"
$remoteUrl = "https://github.com/Wadek/wakabot.git"

function Invoke-Checked {
    param([string]$File, [string[]]$Arguments)
    $previousErrorAction = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
        $output = & $File @Arguments 2>&1
    } finally {
        $ErrorActionPreference = $previousErrorAction
    }
    $code = $LASTEXITCODE
    $output | ForEach-Object { Write-Host $_ }
    if ($code -ne 0) {
        throw "$File failed with exit code $code"
    }
}

Push-Location $repoRoot
try {
    if (-not (Get-Command git -ErrorAction SilentlyContinue)) { throw "git not found" }
    if (-not (Get-Command gh -ErrorAction SilentlyContinue)) { throw "gh not found" }

    if (-not (Test-Path -LiteralPath (Join-Path $repoRoot ".git"))) {
        Invoke-Checked "git" @("init", "-b", $branch)
    } else {
        & git show-ref --verify --quiet "refs/heads/$branch"
        if ($LASTEXITCODE -eq 0) {
            Invoke-Checked "git" @("switch", $branch)
        } else {
            Invoke-Checked "git" @("switch", "-c", $branch)
        }
    }

    $tokens = $null
    $parseErrors = $null
    Get-ChildItem -Path $repoRoot -Recurse -Filter *.ps1 | ForEach-Object {
        [System.Management.Automation.Language.Parser]::ParseFile($_.FullName, [ref]$tokens, [ref]$parseErrors) | Out-Null
        if ($parseErrors.Count -gt 0) { throw "$($_.FullName) has PowerShell syntax errors" }
    }

    Push-Location "D:\wakalabs\ai_learning\waka-cli"
    try { Invoke-Checked "python" @("-m", "unittest", "discover", "-s", "tests", "-v") }
    finally { Pop-Location }

    $pending = git status --porcelain
    if ($pending) {
        Invoke-Checked "git" @("add", "-A")
        Invoke-Checked "git" @("commit", "-m", "Bootstrap local waka agent team")
    } else {
        Write-Host "No uncommitted changes; keeping existing commit."
    }
    Invoke-Checked "frontier" @("hygiene")
    Invoke-Checked "frontier" @("plan")
    Invoke-Checked "frontier" @("apply")

    $previousErrorAction = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
        $remote = & git remote get-url origin 2>$null
        $remoteCode = $LASTEXITCODE
        $repo = & gh repo view Wadek/wakabot --json nameWithOwner 2>$null
        $repoCode = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $previousErrorAction
    }
    if ($remoteCode -ne 0) {
        if ($repoCode -eq 0) {
            Invoke-Checked "git" @("remote", "add", "origin", $remoteUrl)
            Invoke-Checked "git" @("push", "-u", "origin", "HEAD")
        } else {
            Invoke-Checked "gh" @("repo", "create", "Wadek/wakabot", "--private", "--source", $repoRoot, "--remote", "origin", "--push")
        }
    } else {
        Invoke-Checked "git" @("push", "-u", "origin", "HEAD")
    }

    Write-Host "=== verified repository state ==="
    Invoke-Checked "git" @("log", "-1", "--oneline")
    Invoke-Checked "git" @("remote", "-v")
    Invoke-Checked "git" @("ls-remote", "--heads", "origin", $branch)
} finally {
    Pop-Location
}