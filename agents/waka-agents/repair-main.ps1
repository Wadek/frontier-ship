<#!
.SYNOPSIS
    Repair Wadek/wakabot main from the verified consolidation commit.

.DESCRIPTION
    Archives the current remote main ref before force-updating it. This is a
    repository repair, not a normal feature push, and must be explicitly run.
#>

$ErrorActionPreference = "Stop"
$repoRoot = "D:\wakalabs\waka-bot"
$repo = "Wadek/wakabot"
$branch = "frontier/consolidate-waka-bot"

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
    $current = (git branch --show-current).Trim()
    if ($current -ne $branch) { throw "Expected $branch, found $current" }
    $head = (git rev-parse HEAD).Trim()
    $remoteHead = (git ls-remote --heads origin $branch).Trim()
    if (-not $remoteHead -or $remoteHead -notmatch $head) {
        throw "Local consolidation commit is not the published branch head: $head"
    }

    $mainJson = gh api "repos/$repo/git/ref/heads/main"
    $mainObject = $mainJson | ConvertFrom-Json
    $oldMain = $mainObject.object.sha
    $archive = "archive/main-before-history-repair"
    $oldErrorAction = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
        $archiveJson = gh api "repos/$repo/git/ref/heads/$archive" 2>$null
        $archiveCode = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $oldErrorAction
    }
    if ($archiveCode -ne 0) {
        Invoke-Checked "gh" @("api", "--method", "POST", "repos/$repo/git/refs", "-f", "ref=refs/heads/$archive", "-f", "sha=$oldMain")
    } else {
        Write-Host "Archive ref already exists: $archive"
    }

    Invoke-Checked "gh" @("api", "--method", "PATCH", "repos/$repo/git/refs/heads/main", "-f", "sha=$head", "-F", "force=true")
    Invoke-Checked "gh" @("repo", "edit", $repo, "--default-branch", "main")

    Write-Host "=== verified main repair ==="
    Invoke-Checked "git" @("ls-remote", "--heads", "origin", "main", $branch, $archive)
    Invoke-Checked "gh" @("repo", "view", $repo, "--json", "defaultBranchRef")
} finally {
    Pop-Location
}