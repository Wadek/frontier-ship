<#!
.SYNOPSIS
    Audit local and remote branches and their ancestry without changing refs.
#>

$ErrorActionPreference = "Stop"
$repoRoot = "D:\wakalabs\waka-bot"
Push-Location $repoRoot
try {
    git fetch origin --prune
    Write-Host "=== local branches ==="
    git branch -vv
    Write-Host "=== remote branches ==="
    git ls-remote --heads origin
    Write-Host "=== current main ancestry ==="
    git log --oneline --decorate --graph --all -20
    $main = (git rev-parse refs/remotes/origin/main).Trim()
    Write-Host "origin/main=$main"
    foreach ($ref in @(git for-each-ref --format='%(refname:short)' refs/remotes/origin/heads refs/remotes/origin)) {
        if ($ref -notmatch '^origin/' -or $ref -eq 'origin/HEAD') { continue }
        $sha = (git rev-parse $ref).Trim()
        $ancestor = git merge-base --is-ancestor $sha $main
        $ancestorCode = $LASTEXITCODE
        $descendant = git merge-base --is-ancestor $main $sha
        $descendantCode = $LASTEXITCODE
        Write-Host "$ref=$sha ancestor-of-main=$($ancestorCode -eq 0) main-ancestor-of-branch=$($descendantCode -eq 0)"
    }
} finally {
    Pop-Location
}
