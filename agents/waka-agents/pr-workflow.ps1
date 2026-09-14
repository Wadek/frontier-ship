<#!
.SYNOPSIS
    Verify main, teach the local workflow, and open a feature-to-main PR.
#>

$ErrorActionPreference = "Stop"
$repoRoot = $PSScriptRoot
$branch = "frontier/bootstrap-waka-agents"

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
    if ($current -ne $branch) { throw "Expected branch $branch, found $current" }

    $mainRef = git ls-remote --heads origin main
    if ($LASTEXITCODE -ne 0 -or -not $mainRef) {
        throw "origin/main does not exist. Create the initial main branch through the repository owner before opening this PR."
    }

    $readme = Join-Path $repoRoot "README.md"
    $text = Get-Content -LiteralPath $readme -Raw -Encoding UTF8
    if ($text -notmatch "## Feature branch to main PR workflow") {
        Add-Content -LiteralPath $readme -Encoding UTF8 -Value @"

## Feature branch to main PR workflow

Waka Local creates changes on a `frontier/*` branch and opens a pull request into
`main`; it never merges the pull request. The repeatable order is:

1. Inspect `git status --short --branch` and preserve user changes.
2. Run focused tests and verbose validation.
3. Run `frontier hygiene`, `frontier plan`, and `frontier apply`.
4. Push the feature branch with normal hooks.
5. Create the PR with `gh pr create --base main --head <feature> --fill`.
6. Verify the PR URL, base, head, and state. A human merges it.

Never use `--no-verify`, override `core.hooksPath`, use GitHub MCP write tools,
or merge directly into `main`.
"@
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

    if (git status --porcelain) {
        Invoke-Checked "git" @("add", "README.md", "pr-workflow.ps1")
        Invoke-Checked "git" @("commit", "-m", "Teach Waka Local feature PR workflow")
    }
    Invoke-Checked "frontier" @("hygiene")
    Invoke-Checked "frontier" @("plan")
    Invoke-Checked "frontier" @("apply")
    Invoke-Checked "git" @("push", "-u", "origin", "HEAD")

    $existing = gh pr list --head $branch --base main --state open --json number --jq '.[0].number'
    if (-not $existing) {
        Invoke-Checked "gh" @("pr", "create", "--base", "main", "--head", $branch, "--fill")
    } else {
        Write-Host "PR already open: #$existing"
    }
    Invoke-Checked "gh" @("pr", "view", "--json", "number,url,state,baseRefName,headRefName")
    Invoke-Checked "git" @("ls-remote", "--heads", "origin", "main", $branch)
} finally {
    Pop-Location
}