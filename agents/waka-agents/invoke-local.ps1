<#
.SYNOPSIS
    Invoke waka-cli as the local implementation agent.

.DESCRIPTION
    This is the bridge used by the VS Code Waka Local agent profile. It keeps
    the prompt and working directory as PowerShell arguments instead of
    assembling a shell command string.
#>

param(
    [string]$Prompt,
    [string]$PromptFile,
    [switch]$BootstrapRepo,
    [switch]$CreatePr,
    [switch]$Consolidate,
    [switch]$PublishConsolidated,
    [switch]$CreateMain,
    [switch]$CreatePrs,
    [switch]$RepairMain,
    [switch]$AuditBranches,
    [switch]$CreateGitAgentPr,
    [string]$Cwd = (Get-Location).Path,
    [string]$Model = "waka-coder",
    [int]$MaxTurns = 40
)

$ErrorActionPreference = "Stop"
$runnerPath = "D:\wakalabs\ai_learning\waka-cli\run.py"

if ($BootstrapRepo) {
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot "bootstrap-repo.ps1")
    exit $LASTEXITCODE
}

if ($CreatePr) {
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot "pr-workflow.ps1")
    exit $LASTEXITCODE
}

if ($Consolidate) {
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot "consolidate-wakabot.ps1")
    exit $LASTEXITCODE
}

if ($PublishConsolidated) {
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot "publish-consolidated.ps1")
    exit $LASTEXITCODE
}

if ($CreateMain) {
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot "create-main.ps1")
    exit $LASTEXITCODE
}

if ($CreatePrs) {
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot "create-prs.ps1")
    exit $LASTEXITCODE
}

if ($RepairMain) {
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot "repair-main.ps1")
    exit $LASTEXITCODE
}

if ($AuditBranches) {
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot "audit-branches.ps1")
    exit $LASTEXITCODE
}

if ($CreateGitAgentPr) {
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot "create-git-agent-pr.ps1")
    exit $LASTEXITCODE
}

if (-not $Prompt -and -not $PromptFile) {
    throw "Provide either -Prompt or -PromptFile."
}

if ($PromptFile) {
    if (-not (Test-Path -LiteralPath $PromptFile -PathType Leaf)) {
        throw "Prompt file was not found: $PromptFile"
    }
    $Prompt = Get-Content -LiteralPath $PromptFile -Raw -Encoding UTF8
}

if (-not (Get-Command waka-cli -ErrorAction SilentlyContinue)) {
    throw "waka-cli was not found on PATH. Run ai_learning\waka-cli\install.ps1 first."
}

if (-not (Get-Command python -ErrorAction SilentlyContinue)) {
    throw "python was not found on PATH. waka-cli cannot start."
}

if (-not (Test-Path -LiteralPath $runnerPath -PathType Leaf)) {
    throw "waka-cli runner was not found: $runnerPath"
}

if (-not (Test-Path -LiteralPath $Cwd -PathType Container)) {
    throw "Working directory does not exist: $Cwd"
}

& python -u $runnerPath --continue -p $Prompt --always-approve --cwd $Cwd --model $Model --max-turns $MaxTurns
exit $LASTEXITCODE