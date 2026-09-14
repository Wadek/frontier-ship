<#!
.SYNOPSIS
    Create the Git Agent learning PR from the current consolidation branch.
#>

$ErrorActionPreference = "Stop"
$repoRoot = "D:\wakalabs\waka-bot"
$branch = "frontier/consolidate-waka-bot"

Push-Location $repoRoot
try {
    $current = (git branch --show-current).Trim()
    if ($current -ne $branch) { throw "Expected $branch, found $current" }
    git fetch origin --prune
    $base = (git rev-parse origin/main).Trim()
    $head = (git rev-parse "origin/$branch").Trim()
    git merge-base --is-ancestor $head $base
    $headInBase = $LASTEXITCODE -eq 0
    if ($headInBase) {
        throw "The branch has no changes beyond origin/main; no PR is needed."
    }
    $mergeBase = (git merge-base $base $head).Trim()
    if (-not $mergeBase) {
        throw "The branch and origin/main do not have a common integration ancestry."
    }
    Write-Host "merge-base=$mergeBase"
    $existing = gh pr list --head $branch --base main --state open --json number,url --jq '.[0]'
    if ($existing) {
        Write-Host "PR already exists: $existing"
    } else {
        gh pr create --base main --head $branch --title "Teach Waka Local Git Agent workflows" --body "Adds the Git Agent profile, Git/GitHub/Frontier handbook, branch audit, and history-repair guidance. Human review and merge required."
        if ($LASTEXITCODE -ne 0) { throw "gh pr create failed with exit code $LASTEXITCODE" }
    }
    gh pr view --json number,url,state,baseRefName,headRefName
} finally {
    Pop-Location
}