# waka-agents shared configuration
# Dot-source this from every agent: . "$PSScriptRoot\..\config.ps1"

$WAKA_AGENTS = Split-Path -Parent $PSScriptRoot
$WAKA_HOME   = "D:\wakalabs"
$LOGS_DIR    = "$WAKA_AGENTS\logs"
$PROMPTS_DIR = "$WAKA_AGENTS\prompts"
$WAKA_BOT    = "waka-bot"   # on PATH via C:\Users\waka\bin\waka-bot.cmd

# Ollama
$OLLAMA_HOST  = "http://127.0.0.1:11434"
$WAKA_MODEL   = "waka-coder"   # local 7B, no API key

# ── Bloat: processes to kill ───────────────────────────────────────────────
$BLOAT_PROCESSES = @(
    "iCloudHome",
    "ApplePhotoStreams",
    "iCloudCKKS",
    "iCloudServices",
    "gamingservices",
    "gamingservicesnet",
    "GameInputRedistService",
    "GameInputSvc",
    "AsusUpdateCheck",
    "OneApp.IGCC.WinService",
    "jhi_service"
)

# ── Bloat: services to set Disabled ───────────────────────────────────────
$BLOAT_SERVICES = @(
    @{ Name = "iCloudServices";           Display = "iCloud Services"                },
    @{ Name = "Apple Push";               Display = "Apple Push"                     },
    @{ Name = "GamingServices";           Display = "Gaming Services"                },
    @{ Name = "GamingServicesNet";        Display = "Gaming Services Net"            },
    @{ Name = "GameInputRedistService";   Display = "GameInput Redist Service"       },
    @{ Name = "GameInputSvc";             Display = "GameInput Service"              },
    @{ Name = "AsusUpdateCheck";          Display = "AsusUpdateCheck"                },
    @{ Name = "igccservice";              Display = "Intel Graphics Command Center"  },
    @{ Name = "jhi_service";             Display = "Intel Dynamic App Loader"       },
    @{ Name = "DiagTrack";               Display = "Connected User Experiences / Telemetry" },
    @{ Name = "DoSvc";                   Display = "Delivery Optimization"          }
)

# ── Bloat: scheduled tasks to disable ────────────────────────────────────
$BLOAT_TASKS = @(
    @{ Path = "\";          Name = "NvDriverUpdateCheckDaily_{B2FE1952-0186-46C3-BAEC-A80AA35AC5B8}"             },
    @{ Path = "\";          Name = "NVIDIA GeForce Experience SelfUpdate_{B2FE1952-0186-46C3-BAEC-A80AA35AC5B8}" },
    @{ Path = "\";          Name = "NvNodeLauncher_{B2FE1952-0186-46C3-BAEC-A80AA35AC5B8}"                      },
    @{ Path = "\";          Name = "NvProfileUpdaterDaily_{B2FE1952-0186-46C3-BAEC-A80AA35AC5B8}"               },
    @{ Path = "\";          Name = "NvProfileUpdaterOnLogon_{B2FE1952-0186-46C3-BAEC-A80AA35AC5B8}"             },
    @{ Path = "\";          Name = "NvTmRep_CrashReport1_{B2FE1952-0186-46C3-BAEC-A80AA35AC5B8}"               },
    @{ Path = "\";          Name = "NvTmRep_CrashReport2_{B2FE1952-0186-46C3-BAEC-A80AA35AC5B8}"               },
    @{ Path = "\";          Name = "NvTmRep_CrashReport3_{B2FE1952-0186-46C3-BAEC-A80AA35AC5B8}"               },
    @{ Path = "\";          Name = "NvTmRep_CrashReport4_{B2FE1952-0186-46C3-BAEC-A80AA35AC5B8}"               },
    @{ Path = "\Ubisoft\";  Name = "Ubisoft Connect Background Update"                                          }
)

# ── Docker: containers to watch ──────────────────────────────────────────
$DOCKER_WATCH = @(
    "waka-gateway",
    "waka-waf",
    "ollama",
    "food",
    "satokori",
    "opengym-web",
    "polly-engine"
)

# ── Security: WAF patterns that trigger a report ─────────────────────────
$WAF_CRITICAL_PATTERNS = @(
    "\[CRITICAL\]",
    "\[ALERT\]",
    "id 942",    # SQL injection
    "id 941",    # XSS
    "id 930",    # Path traversal
    "ATTACK-"
)
$WAF_RATE_WINDOW_SEC  = 60
$WAF_RATE_THRESHOLD   = 10   # hits from same IP in window = anomaly

# ── Monitor: thresholds ────────────────────────────────────────────────────
$DISK_WARN_GB         = 20   # alert if C: or D: free < this
$RAM_WARN_PCT         = 90   # alert if RAM usage > this %
$MONITOR_INTERVAL_MIN = 30

# ── Logging helper ────────────────────────────────────────────────────────
function Write-AgentLog {
    param([string]$Agent, [string]$Message, [string]$Level = "INFO")
    $null = New-Item -ItemType Directory -Force -Path $LOGS_DIR
    $ts   = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $line = "[$ts] [$Level] [$Agent] $Message"
    Write-Host $line
    Add-Content -Path "$LOGS_DIR\$Agent-$(Get-Date -Format 'yyyyMMdd').log" -Value $line -Encoding UTF8
}
