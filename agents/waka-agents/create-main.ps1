<#!
.SYNOPSIS
    Create the remote main branch from the current published commit.
#>

$ErrorActionPreference = "Stop"
$repoRoot = "D:\wakalabs\waka-bot"
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
    $current = git branch --show-current
    if ($current -ne "frontier/consolidate-waka-bot") {
        throw "Expected the published consolidation branch; found $current"
    }
    $head = (git rev-parse HEAD).Trim()
    $main = git ls-remote --heads origin main
    if ($main) {
        Write-Host "origin/main already exists"
    } else {
        & git show-ref --verify --quiet refs/heads/main
        if ($LASTEXITCODE -ne 0) {
            Invoke-Checked "git" @("branch", "main", $head)
        } else {
            Write-Host "local main already exists"
        }
        Invoke-Checked "git" @("frontier", "plan")
        Invoke-Checked "git" @("frontier", "apply")
        Invoke-Checked "git" @("push", "origin", "main")
    }
    Invoke-Checked "gh" @("repo", "edit", "Wadek/wakabot", "--default-branch", "main")
    Write-Host "=== verified main ==="
    Invoke-Checked "git" @("ls-remote", "--heads", "origin", "main")
    Invoke-Checked "git" @("branch", "-a")
    Invoke-Checked "gh" @("repo", "view", "Wadek/wakabot", "--json", "defaultBranchRef")
} finally {
    Pop-Location
}