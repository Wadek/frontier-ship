<#!
.SYNOPSIS
    Create missing PRs from the published waka-bot feature branches into main.
#>

$ErrorActionPreference = "Stop"
$repoRoot = "D:\wakalabs\waka-bot"
$branches = @("frontier/bootstrap-waka-agents", "frontier/consolidate-waka-bot")

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
    $main = git ls-remote --heads origin main
    if ($LASTEXITCODE -ne 0 -or -not $main) {
        throw "origin/main does not exist; GitHub cannot create PRs targeting main yet."
    }
    Invoke-Checked "git" @(
        "fetch", "origin",
        "refs/heads/main:refs/remotes/origin/main",
        "refs/heads/frontier/bootstrap-waka-agents:refs/remotes/origin/frontier/bootstrap-waka-agents",
        "refs/heads/frontier/consolidate-waka-bot:refs/remotes/origin/frontier/consolidate-waka-bot"
    )
    foreach ($branch in $branches) {
        $remote = git ls-remote --heads origin $branch
        if (-not $remote) {
            Write-Host "Skipping missing remote branch: $branch"
            continue
        }
        $existing = gh pr list --head $branch --base main --state open --json number,url --jq '.[0]'
        if ($existing) {
            Write-Host "PR already exists for ${branch}: $existing"
        } else {
            $title = "Merge $branch into main"
            $body = "Automated Waka Local PR for the published feature branch $branch. Human review and merge required."
            Invoke-Checked "gh" @("pr", "create", "--base", "main", "--head", $branch, "--title", $title, "--body", $body)
        }
    }
    foreach ($branch in $branches) {
        Invoke-Checked "gh" @("pr", "list", "--head", $branch, "--base", "main", "--state", "all", "--json", "number,url,state,baseRefName,headRefName")
    }
} finally {
    Pop-Location
}