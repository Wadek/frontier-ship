package fronticli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Wadek/frontier-ship/internal/catalog"
	"github.com/Wadek/frontier-ship/internal/gitx"
	"github.com/Wadek/frontier-ship/internal/hygiene"
	"github.com/Wadek/frontier-ship/internal/learn"
	"github.com/Wadek/frontier-ship/internal/ledger"
	"github.com/Wadek/frontier-ship/internal/monitor"
	"github.com/Wadek/frontier-ship/internal/optimize"
	"github.com/Wadek/frontier-ship/internal/owasp"
	"github.com/Wadek/frontier-ship/internal/policy"
	fruntime "github.com/Wadek/frontier-ship/internal/runtime"
	"github.com/Wadek/frontier-ship/internal/vscan"
)

// Set by SLSA / release ldflags.
var (
	version = "dev"
	commit  = "none"
)

// frontier-git: drop-in git wrapper. Most commands pass through.
// Frontier security applies to high-blast verbs (push; commit on main).
//
// Env:
//
//	FRONTIER_GIT_BIN   real git executable (optional)
//	FRONTIER_LEDGER    ledger path (optional; else .frontier/ledger.jsonl upward)
//	FRONTIER_SOFT=1    warn instead of deny (learning mode)
//	FRONTIER_STRICT=1  also require gate for `commit` always (optional hardness)

// Version/Commit set via ldflags from cmd wrappers.
var (
	Version = "dev"
	Commit  = "none"
)

// Run executes frontier subcommands (V, S, plan, apply, ...).
// Git passthrough stays in cmd/frontier-git.
func Run(args []string) {
	if Version != "dev" || Commit != "none" {
		version, commit = Version, Commit
	}
	handleMeta(args)
}

// GuardPush is used by the git shim.
func GuardPush(soft bool) error { return guardPush(soft) }

// GuardCommit is used by the git shim.
func GuardCommit(args []string, soft, strict bool) error {
	return guardCommit(args, soft, strict)
}

// FindRealGit exposes real git path for the shim.
func FindRealGit() string { return findRealGit() }

// RunGitPassthrough runs real git and returns exit code.
func RunGitPassthrough(git string, args []string) int {
	return runPassthrough(git, args)
}

func guardPush(soft bool) error {
	cwd, _ := os.Getwd()
	axiom("F0", "push.start", "push requested — evidence + gate required")
	repo := gitx.Repo{Dir: cwd}
	branch, err := repo.Branch()
	if err != nil {
		return fmt.Errorf("frontier: not a git work tree? %w", err)
	}
	head, _ := repo.RevParseHead()
	por, _ := repo.StatusPorcelain()

	axiom("F4", "push.recheck_policy", "re-scan OWASP V before authorizing remote mutate")
	findings, _ := owasp.ScanTree(cwd)
	if verbose() {
		fmt.Fprintln(os.Stderr, owasp.FormatReport(findings))
	}

	g := policy.EvaluatePushGate(branch, head, por, false)
	if owasp.BlocksGate(findings) {
		g.OK = false
		g.Reasons = append(g.Reasons, "OWASP V: untriaged High/Critical finding(s)")
		axiom("F4", "push.block", "High/Critical under V")
	}
	if !g.OK {
		axiom("F0", "push.deny", strings.Join(g.Reasons, "; "))
		msg := fmt.Sprintf("frontier deny push: %s", strings.Join(g.Reasons, "; "))
		if soft {
			fmt.Fprintln(os.Stderr, "WARNING:", msg, "(FRONTIER_SOFT=1 — allowing)")
			axiom("F3", "soft_allow", "learning mode weakens continuity — turn FRONTIER_SOFT off")
			return nil
		}
		return fmt.Errorf("%s\nhint: use a feature branch, commit cleanly, or: git frontier gate", msg)
	}

	ledPath := findLedger(cwd)
	led, err := ledger.Open(ledPath)
	if err != nil {
		return err
	}
	ok, detail := policy.FreshGateOK(led, branch, head)
	if !ok {
		_, _ = led.Append("frontier-git", "push.denied", map[string]any{
			"branch": branch, "head": head, "reason": detail,
		})
		axiom("F0", "push.deny", detail)
		msg := fmt.Sprintf("frontier deny push: %s", detail)
		if soft {
			fmt.Fprintln(os.Stderr, "WARNING:", msg, "(FRONTIER_SOFT=1 — allowing)")
			_, _ = led.Append("frontier-git", "push.soft_allow", map[string]any{"branch": branch, "head": head})
			return nil
		}
		return fmt.Errorf("%s\nrun: git frontier gate\nthen retry push", msg)
	}
	_, _ = led.Append("frontier-git", "push.authorized", map[string]any{
		"branch": branch, "head": head, "gate": detail, "owasp_findings": len(findings),
	})
	axiom("F0", "push.authorized", detail)
	axiom("F2", "push.execute", "passing through to real git push")
	fmt.Fprintln(os.Stderr, "frontier: push authorized (gate ok)")
	return nil
}

func guardCommit(args []string, soft, strict bool) error {
	cwd, _ := os.Getwd()
	axiom("F0", "commit.start", "commit is Operator-level; evidence path continues in ledger on gate")
	repo := gitx.Repo{Dir: cwd}
	branch, err := repo.Branch()
	if err != nil {
		return nil // let real git error
	}
	onMain := strings.EqualFold(branch, "main") || strings.EqualFold(branch, "master")
	if onMain {
		axiom("F1", "commit.deny_main", "direct commits on main expand blast radius")
		msg := "frontier deny commit on main/master — create a feature branch first (git checkout -b frontier/...)"
		if soft {
			fmt.Fprintln(os.Stderr, "WARNING:", msg, "(FRONTIER_SOFT=1 — allowing)")
			return nil
		}
		return fmt.Errorf("%s", msg)
	}
	if strict {
		ledPath := findLedger(cwd)
		led, err := ledger.Open(ledPath)
		if err == nil {
			_, _ = led.Append("frontier-git", "commit.attempt", map[string]any{
				"branch": branch, "args": strings.Join(args, " "),
			})
		}
	}
	return nil
}

func handleMeta(args []string) {
	if len(args) == 0 {
		fmt.Println(`git frontier commands (Terraform-like: plan → apply → push):

  git frontier plan         preview Guard (note Slim); FAIL stops the world
  git frontier apply        seal push authorization only if plan passed
  git frontier gate         alias of apply

  Words (preferred)     Letter aliases
  -------------------   --------------
  frontier learn        L   — Learn / Landscape (classify before change)
  frontier guard        G   — Guard / security exam (OWASP + secret surfaces)
  frontier hygiene      H   — Hygiene / AI provenance (watermarks-remover)
  frontier runtime      R   — Runtime / probe + bounded chaos (allowlist)
  frontier slim         S   — Slim / vibe-bloat (PLANNED — not enforced)
  frontier optimize     O   — Optimize report (behavior-preserving speed; advise)

  Onboarding (no letter):  frontier scm status|init|connect
  Supervision (no letter): frontier monitor  — audit agent behavior vs directives

  git frontier learn classify [path]
  git frontier guard list|checkov
  git frontier hygiene inspect|status|clean PATH
  git frontier runtime status|scan|chaos|budget
  git frontier optimize report|status|pr-body Opt-001
  git frontier monitor [all|status|directives]
  git frontier skills|agents [list|show NAME]
  git frontier enhance guard|optimize
  git frontier enhance status|seal
  git frontier mock-import

  git frontier status|ledger|demo|explain

Letter notes (avoid shell pain):
  Prefer full words in scripts. Single letters are aliases only.
  g is sometimes aliased to git in zsh — use "frontier guard" or "frontier G".
  Avoid overlapping common tools: ls, cd, ps, rm, git, go, gh, …

Env: FRONTIER_SOFT=1  FRONTIER_VERBOSE=1  FRONTIER_GIT_BIN  FRONTIER_LEDGER
     FRONTIER_V_AUTO=1  also run available adapters during enhance/guard pack

Nothing remote goes if plan/apply fails (like terraform).

Same as standalone:  frontier scm | learn | guard | hygiene | runtime | slim | optimize | plan | apply
(Not \"go frontier\" — go is the Go toolchain)`)
		return
	}
	cwd, _ := os.Getwd()
	switch args[0] {
	case "status":
		repo := gitx.Repo{Dir: cwd}
		b, _ := repo.Branch()
		p, _ := repo.StatusPorcelain()
		fmt.Printf("cwd: %s\nbranch: %s\ndirty: %v\nledger: %s\nreal_git: %s\nsoft: %v\n",
			cwd, b, policy.DirtyPorcelain(p), findLedger(cwd), findRealGit(), os.Getenv("FRONTIER_SOFT") == "1")
	case "scm":
		runSCM(cwd, args[1:])
	case "learn", "L", "l":
		runLearn(cwd, args[1:])
	case "guard", "G", "g", "exam":
		if len(args) > 1 {
			runGuardSub(cwd, args[1:])
			return
		}
		runExam(cwd, true)
	case "V", "v": // temporary aliases → guard
		fmt.Fprintln(os.Stderr, "note: frontier V is now frontier guard (G)")
		if len(args) > 1 {
			runGuardSub(cwd, args[1:])
			return
		}
		runExam(cwd, true)
	case "enhance":
		runEnhance(cwd, args[1:])
	case "slim", "S", "s":
		printSlimStub()
	case "optimize", "O", "o":
		runOptimize(cwd, args[1:])
	case "hygiene", "watermarks", "watermark", "marks", "H", "h":
		runHygiene(cwd, args[1:])
	case "runtime", "probe", "chaos", "R", "r":
		runRuntime(cwd, args[1:])
	case "monitor", "audit":
		runMonitor(cwd, args[1:])
	case "skills", "skill":
		runSkills(args[1:])
	case "agents", "agent":
		runAgents(args[1:])
	case "plan":
		runPlan(cwd, true)
	case "apply", "gate":
		runApply(cwd, true)
	case "mock-import":
		printMockImport()
	case "ledger":
		led, err := ledger.Open(findLedger(cwd))
		if err != nil {
			fail(err)
			return
		}
		rows, _ := led.Tail(15)
		for _, r := range rows {
			fmt.Printf("%d %s %s %s\n", r.Seq, r.TS, r.Actor, r.Action)
		}
	case "demo":
		printDemo(cwd)
	case "explain":
		fmt.Printf("frontier-git %s (%s)\n\n", version, commit)
		fmt.Println(`You are talking to Frontier through the git interface.

  Type:  git …
  Engine: FRONTIER_GIT_BIN (real git)

Terraform-like flow:
  git frontier plan    # preview — fails closed
  git frontier apply   # authorize — only if plan passed
  git push             # only if apply/gate sealed

Onboarding:
  frontier scm         # VCS detect/init/connect (before Learn if needed)

Policy families (word = primary, letter = alias):
  Learn     (L)  — ingest + classify before change
  Guard     (G)  — security + secret surfaces; enforced at changeset
  Hygiene   (H)  — AI provenance (watermarks-remover); advise
  Runtime   (R)  — post-ship probe + bounded chaos (allowlist)
  Slim      (S)  — vibe-code bloat; PLANNED
  Optimize  (O)  — behavior-preserving speed; report + small PRs
  Monitor        — audit ledgered agent behavior vs ship directives (frontier monitor)

Enhance:
  frontier enhance guard | optimize

Control points: changeset | review | runtime | engagement
Languages: English · Haskell · Go
State: ledger (like terraform state) — evidence of plan/apply`)
	default:
		fmt.Fprintf(os.Stderr, "unknown frontier subcommand %q\n", args[0])
		os.Exit(2)
	}
}

func printSlimStub() {
	fmt.Println(`╔══════════════════════════════════════════════╗
║  Slim (S) — PLANNED, not enforced yet        ║
╚══════════════════════════════════════════════╝
  Purpose: manage vibe-code bloat (least code that still works)
  Control: changeset (advise→block later) + review (intent)
  Today:   use frontier guard (G) for security baseline
  Later:   frontier slim   will report budgets / dead code
  First:   frontier learn classify   (learn before slim)
  After:   frontier optimize (O)     (speed without behavior change)

  Stick with:  frontier learn | guard | hygiene | plan | apply | push
╚══════════════════════════════════════════════╝`)
}

func printOptimizeStub() {
	fmt.Println(`╔══════════════════════════════════════════════╗
║  Optimize (O) — behavior-preserving speed    ║
╚══════════════════════════════════════════════╝
  Purpose: simplest correct change; intended behavior unchanged
  After:   learn → guard → slim
  Output:  .frontier/optimize/O-*  (PR body = Opt section)
  Docs:    english/O_OPTIMIZE.md

  frontier optimize              # run report (alias: report)
  frontier optimize report
  frontier optimize status
  frontier optimize pr-body Opt-001
  frontier enhance optimize      # residual CS fill (planned depth)

  One Opt-ID per small PR. Advise-only — does not block gate.
╚══════════════════════════════════════════════╝`)
}

func printHygieneStub() {
	fmt.Println(`╔══════════════════════════════════════════════╗
║  Hygiene (H) — AI provenance marks           ║
╚══════════════════════════════════════════════╝
  Word:    frontier hygiene   aliases: watermarks, H
  Service: watermarks-remover  http://127.0.0.1:8765
  Docs:    english/H_HYGIENE.md

  frontier hygiene              # inspect changeset (advise)
  frontier hygiene inspect
  frontier hygiene status       # health + capabilities
  frontier hygiene clean PATH   # write PATH.cleaned.ext
  frontier hygiene clean PATH --in-place

  Default does not block plan/apply.
  FRONTIER_HYGIENE_BLOCK=1  makes suspicious marks fail the gate.
  WATERMARKS_SERVICE_URL    override (default loopback :8765)

  Start service:
    python D:\wakalabs\watermarks-remover\service\scripts\server.py --host 127.0.0.1 --port 8765
╚══════════════════════════════════════════════╝`)
}

func runHygiene(cwd string, args []string) {
	inPlace := false
	var filtered []string
	for _, a := range args {
		switch strings.ToLower(a) {
		case "--in-place", "-i":
			inPlace = true
		case "help", "-h", "--help":
			printHygieneStub()
			return
		default:
			filtered = append(filtered, a)
		}
	}
	sub := "inspect"
	rest := filtered
	if len(filtered) > 0 {
		switch strings.ToLower(filtered[0]) {
		case "inspect", "status", "clean":
			sub = strings.ToLower(filtered[0])
			rest = filtered[1:]
		}
	}
	switch sub {
	case "status":
		runHygieneStatus()
	case "clean":
		if len(rest) == 0 {
			fail(fmt.Errorf("usage: frontier hygiene clean PATH [--in-place]"))
			return
		}
		runHygieneClean(cwd, rest[0], inPlace)
	default:
		runHygieneInspect(cwd, rest, true)
	}
}

func runHygieneStatus() {
	fmt.Println(`╔══════════════════════════════════════════════╗
║  HYGIENE (H) — service status                ║
╚══════════════════════════════════════════════╝`)
	base := hygiene.ServiceURL()
	fmt.Printf("url: %s\n", base)
	ver, err := hygiene.Health(base)
	if err != nil {
		fmt.Printf("healthy: no\nerror:   %s\n", err)
		fmt.Println("start:   python D:\\wakalabs\\watermarks-remover\\service\\scripts\\server.py --host 127.0.0.1 --port 8765")
		return
	}
	fmt.Printf("healthy: yes\nversion: %s\n", ver)
	if caps, err := hygiene.Capabilities(base); err == nil && caps != nil {
		b, _ := json.MarshalIndent(caps, "", "  ")
		fmt.Println(string(b))
	}
}

func hygieneTargets(cwd string, extra []string) []string {
	if len(extra) > 0 {
		return hygiene.ResolveTargets(cwd, extra)
	}
	repo := gitx.Repo{Dir: cwd}
	return hygiene.ResolveTargets(cwd, repo.ChangedPaths())
}

func runHygieneInspect(cwd string, extra []string, seal bool) *hygiene.Report {
	fmt.Println(`╔══════════════════════════════════════════════╗
║  HYGIENE (H) — changeset inspect             ║
╚══════════════════════════════════════════════╝`)
	axiom("F0", "hygiene.start", "inspect AI provenance on the changeset")
	targets := hygieneTargets(cwd, extra)
	rep := hygiene.InspectFiles(hygiene.ServiceURL(), cwd, targets)
	fmt.Print(hygiene.FormatReport(rep))
	fmt.Println("╚══════════════════════════════════════════════╝")
	if seal {
		led, err := ledger.Open(findLedger(cwd))
		if err == nil {
			action := "hygiene.inspected"
			if !rep.Healthy {
				action = "hygiene.service_down"
			}
			_, _ = led.Append("frontier-git", action, map[string]any{
				"healthy":     rep.Healthy,
				"scanned":     rep.Scanned,
				"suspicious":  rep.Suspicious,
				"disposition": hygiene.Disposition(rep),
				"service":     rep.Service,
			})
			axiom("F0", "ledger.append", action+" sealed")
		}
	}
	return rep
}

func runHygieneClean(cwd, path string, inPlace bool) {
	fmt.Println(`╔══════════════════════════════════════════════╗
║  HYGIENE (H) — clean                         ║
╚══════════════════════════════════════════════╝`)
	axiom("F0", "hygiene.clean", "explicit strip; not silent on plan")
	base := hygiene.ServiceURL()
	if _, err := hygiene.Health(base); err != nil {
		fail(fmt.Errorf("watermarks-remover down at %s: %w", base, err))
		return
	}
	targets := hygiene.ResolveTargets(cwd, []string{path})
	if len(targets) == 0 {
		fail(fmt.Errorf("no hygiene-eligible file at %s", path))
		return
	}
	abs := targets[0]
	out, f, err := hygiene.CleanFile(base, abs, path, inPlace)
	if err != nil {
		fail(err)
		return
	}
	fmt.Printf("wrote:  %s\nkind:   %s\nreport: %s\n", out, f.Kind, f.Report)
	led, err := ledger.Open(findLedger(cwd))
	if err == nil {
		_, _ = led.Append("frontier-git", "hygiene.cleaned", map[string]any{
			"src": path, "dst": out, "kind": f.Kind, "in_place": inPlace,
		})
		axiom("F0", "ledger.append", "hygiene.cleaned sealed")
	}
}

func inspectHygieneQuiet(cwd string) *hygiene.Report {
	targets := hygieneTargets(cwd, nil)
	return hygiene.InspectFiles(hygiene.ServiceURL(), cwd, targets)
}

func printRuntimeStub() {
	fmt.Println(`╔══════════════════════════════════════════════╗
║  Runtime (R) — probe + bounded chaos         ║
╚══════════════════════════════════════════════╝
  Word:    frontier runtime   aliases: probe, chaos, R
  Docs:    english/R_RUNTIME.md

  frontier runtime              # status
  frontier runtime scan         # GET allowlisted loopback URLs
  frontier runtime chaos        # dry-run inject plan (v1 does not inject)
  frontier runtime budget       # report configured vs consumed token %

  Allowlist: FRONTIER_RUNTIME_ALLOWLIST  (default D:\frontier\runtime\allowlist.json)
  Token %:   FRONTIER_RUNTIME_TOKEN_PCT  (default 5)
  Inject:    FRONTIER_RUNTIME_CHAOS=1    (still dry in v1 — no network mutate)
  v1 HTTP targets: 127.0.0.1 / localhost only.
╚══════════════════════════════════════════════╝`)
}

func runRuntime(cwd string, args []string) {
	sub := "status"
	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "help", "-h", "--help":
			printRuntimeStub()
			return
		case "status", "scan", "chaos", "budget":
			sub = strings.ToLower(args[0])
		default:
			fmt.Fprintf(os.Stderr, "unknown runtime subcommand %q (try: status|scan|chaos|budget)\n", args[0])
			os.Exit(2)
		}
	}
	switch sub {
	case "scan":
		runRuntimeScan(cwd)
	case "chaos":
		runRuntimeChaos(cwd)
	case "budget":
		runRuntimeBudget(cwd)
	default:
		runRuntimeStatus(cwd)
	}
}

func runRuntimeStatus(cwd string) {
	fmt.Println(`╔══════════════════════════════════════════════╗
║  RUNTIME (R) — status                        ║
╚══════════════════════════════════════════════╝`)
	path := fruntime.AllowlistPath()
	fmt.Printf("allowlist: %s\n", path)
	al, err := fruntime.LoadAllowlist(path)
	if err != nil {
		fmt.Printf("loaded:    no\nerror:     %s\n", err)
		fmt.Println("create an allowlist before scan/chaos. See english/R_RUNTIME.md")
		return
	}
	tr := fruntime.ReportTokens(al, 0)
	fmt.Printf("loaded:    yes\ntargets:   %d\nchaos.on:  %v  max_s=%d\ninject:    %v (env)\n",
		len(al.Targets), al.Chaos.Enabled, al.Chaos.MaxDurationS, fruntime.ChaosInjectFlag())
	fmt.Printf("tokens:    configured=%d%%  consumed=%.1f%%  remaining=%.1f%%  cap=%v\n",
		tr.ConfiguredPct, tr.ConsumedPct, tr.RemainingPct, tr.CapEnabled)
	for _, t := range al.Targets {
		fmt.Printf("  - %s  kind=%s  url=%s  net=%s\n", t.Name, t.Kind, t.URL, t.Net)
	}
	led, err := ledger.Open(findLedger(cwd))
	if err == nil {
		_, _ = led.Append("frontier-git", "runtime.status", map[string]any{
			"allowlist": path, "targets": len(al.Targets), "token_pct": fruntime.TokenPct(al),
		})
	}
}

func runRuntimeScan(cwd string) {
	fmt.Println(`╔══════════════════════════════════════════════╗
║  RUNTIME (R) — scan                          ║
╚══════════════════════════════════════════════╝`)
	axiom("F0", "runtime.scan", "allowlisted loopback only")
	al, err := fruntime.LoadAllowlist(fruntime.AllowlistPath())
	if err != nil {
		fail(err)
		return
	}
	probes := fruntime.ScanHTTP(al)
	if len(probes) == 0 {
		fmt.Println("no http targets in allowlist")
	}
	for _, p := range probes {
		if p.Err != "" {
			fmt.Printf("  FAIL %s  %s  %s\n", p.Name, p.URL, p.Err)
			continue
		}
		fmt.Printf("  %d   %s  %s\n", p.Status, p.Name, p.URL)
	}
	led, err := ledger.Open(findLedger(cwd))
	if err == nil {
		_, _ = led.Append("frontier-git", "runtime.scan", map[string]any{
			"n": len(probes), "allowlist": al.Path,
		})
		axiom("F0", "ledger.append", "runtime.scan sealed")
	}
}

func runRuntimeChaos(cwd string) {
	fmt.Println(`╔══════════════════════════════════════════════╗
║  RUNTIME (R) — chaos                         ║
╚══════════════════════════════════════════════╝`)
	al, err := fruntime.LoadAllowlist(fruntime.AllowlistPath())
	if err != nil {
		fail(err)
		return
	}
	plan := fruntime.PlanChaos(al, fruntime.ChaosInjectFlag())
	fmt.Printf("would_inject: %v\n", plan.WouldInject)
	if plan.Denied != "" {
		fmt.Printf("denied:       %s\n", plan.Denied)
	}
	fmt.Printf("duration_s:   %d\nsteps:\n", plan.DurationS)
	for _, s := range plan.Steps {
		fmt.Printf("  - %s\n", s)
	}
	action := "runtime.chaos_dry"
	if plan.WouldInject {
		action = "runtime.chaos_denied"
		fmt.Println("v1 does not inject. Plan is sealed; a later runner may execute under the same allowlist.")
	}
	if plan.Denied != "" && !plan.WouldInject {
		action = "runtime.chaos_dry"
	}
	led, err := ledger.Open(findLedger(cwd))
	if err == nil {
		_, _ = led.Append("frontier-git", action, map[string]any{
			"denied": plan.Denied, "steps": plan.Steps, "duration_s": plan.DurationS,
		})
		axiom("F0", "ledger.append", action+" sealed")
	}
}

func runRuntimeBudget(cwd string) {
	fmt.Println(`╔══════════════════════════════════════════════╗
║  RUNTIME (R) — token consumption report      ║
╚══════════════════════════════════════════════╝`)
	axiom("F0", "runtime.tokens", "report configured vs consumed share (cap off by default)")
	path := fruntime.AllowlistPath()
	al, err := fruntime.LoadAllowlist(path)
	if err != nil {
		fail(err)
		return
	}
	tr := fruntime.ReportTokens(al, 0)
	fmt.Printf("allowlist:     %s\n", path)
	fmt.Printf("configured:    %d%%\nconsumed:      %.1f%%\nremaining:     %.1f%%\ncap_enabled:   %v\n",
		tr.ConfiguredPct, tr.ConsumedPct, tr.RemainingPct, tr.CapEnabled)
	fmt.Println(tr.Note)
	led, err := ledger.Open(findLedger(cwd))
	if err == nil {
		_, _ = led.Append("frontier-git", "runtime.tokens", map[string]any{
			"configured_pct": tr.ConfiguredPct,
			"consumed_pct":   tr.ConsumedPct,
			"remaining_pct":  tr.RemainingPct,
			"cap_enabled":    tr.CapEnabled,
			"allowlist":      path,
		})
		axiom("F0", "ledger.append", "runtime.tokens sealed")
	}
}

func printMonitorStub() {
	fmt.Println(`╔══════════════════════════════════════════════╗
║  MONITOR — audit agent behavior vs directives ║
╚══════════════════════════════════════════════╝
  Purpose: examine ledgered agent behavior and verify the ship
           directives (F0-F4, plan -> apply -> push, feature branches).
  Evidence: every consequential action is sealed in the ledger (F0).
  Docs: english/MONITOR.md

  frontier monitor             # audit this repo's ledger
  frontier monitor all         # audit every ledger under D:\frontier\ledgers
  frontier monitor status      # recent monitor.* seals
  frontier monitor directives  # print the D0-D7 reference set

  Verdicts: clean | watch (denied attempts) | violation | tampered
  Advise-only by default; FRONTIER_MONITOR_BLOCK=1 fails the run on
  violation/tampered verdicts.

  Watcher: scripts\frontier-monitor-watch.ps1  (event-driven; no poll)
╚══════════════════════════════════════════════╝`)
}

func monitorBlock() bool {
	v := os.Getenv("FRONTIER_MONITOR_BLOCK")
	return v == "1" || strings.EqualFold(v, "true")
}

func printMonitorDirectives() {
	fmt.Println(`╔══════════════════════════════════════════════╗
║  MONITOR — directive reference (v0)           ║
╚══════════════════════════════════════════════╝`)
	for _, d := range monitor.Directives {
		fmt.Printf("  %s  %s\n", d.ID, d.Text)
	}
	fmt.Println("╚══════════════════════════════════════════════╝")
}

func runMonitor(cwd string, args []string) {
	sub := ""
	if len(args) > 0 {
		sub = strings.ToLower(args[0])
	}
	switch sub {
	case "help", "-h", "--help":
		printMonitorStub()
		return
	case "directives", "rules":
		printMonitorDirectives()
		return
	case "status":
		runMonitorStatus(cwd)
		return
	}
	fmt.Println(`╔══════════════════════════════════════════════╗
║  MONITOR — directive audit of ledger evidence ║
╚══════════════════════════════════════════════╝`)
	axiom("F0", "monitor.start", "examine ledgered agent behavior against ship directives")
	var reps []*monitor.Report
	var err error
	if sub == "all" || sub == "sweep" {
		reps, err = monitor.AuditAll(monitor.LedgersRoot())
	} else {
		var rep *monitor.Report
		rep, err = monitor.AuditLedger(findLedger(cwd))
		if rep != nil {
			reps = []*monitor.Report{rep}
		}
	}
	if err != nil {
		fail(err)
		return
	}
	if len(reps) == 0 {
		fmt.Println("no ledgers found — run frontier plan/apply/push somewhere first")
		reps = nil
	}
	rows, findings := 0, 0
	for _, r := range reps {
		rows += r.Rows
		findings += len(r.Findings)
		fmt.Println(monitor.FormatReport(r))
		fmt.Println()
	}
	verdict := monitor.AggregateVerdict(reps)
	led, err := ledger.Open(findLedger(cwd))
	if err == nil {
		_, _ = led.Append("frontier-git", "monitor.audited", map[string]any{
			"verdict":    verdict,
			"ledgers":    len(reps),
			"rows":       rows,
			"findings":   findings,
			"blocks_gate": false,
		})
		axiom("F0", "ledger.append", "monitor.audited sealed")
	}
	axiom("F4", "monitor.done", verdict)
	fmt.Printf("verdict: %s  (ledgers=%d rows=%d findings=%d)\n", verdict, len(reps), rows, findings)
	if (verdict == "violation" || verdict == "tampered") && monitorBlock() {
		fmt.Fprintln(os.Stderr, "FRONTIER_MONITOR_BLOCK=1 and verdict is "+verdict)
		os.Exit(2)
	}
}

func runMonitorStatus(cwd string) {
	led, err := ledger.Open(findLedger(cwd))
	if err != nil {
		fail(err)
		return
	}
	rows, _ := led.Tail(40)
	n := 0
	for _, r := range rows {
		if strings.HasPrefix(r.Action, "monitor.") {
			fmt.Printf("%d %s %s %v\n", r.Seq, r.TS, r.Action, r.Payload)
			n++
		}
	}
	if n == 0 {
		fmt.Println("no monitor.* seals yet — run: frontier monitor")
	}
}

func runSkills(args []string) { runCatalog("skills", catalog.SkillsDir(), args) }
func runAgents(args []string) { runCatalog("agents", catalog.AgentsDir(), args) }

func runCatalog(kind, root string, args []string) {
	sub := "list"
	name := ""
	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "list", "ls", "":
			if len(args) > 1 {
				name = args[1]
				sub = "show"
			}
		case "show", "cat":
			sub = "show"
			if len(args) > 1 {
				name = args[1]
			}
		case "help", "-h", "--help":
			fmt.Printf("usage: frontier %s [list|show <name>]\nroot: %s\n", kind, root)
			return
		default:
			fmt.Fprintf(os.Stderr, "unknown %s subcommand %q (try: list | show <name>)\n", kind, args[0])
			os.Exit(2)
		}
	}
	if sub == "show" {
		if name == "" {
			fail(fmt.Errorf("usage: frontier %s show <name>", kind))
			return
		}
		path, err := catalog.Show(root, name)
		if err != nil {
			fail(err)
			return
		}
		if path == "" {
			fail(fmt.Errorf("%s %q: no primary doc found", kind, name))
			return
		}
		b, err := os.ReadFile(path)
		if err != nil {
			fail(err)
			return
		}
		fmt.Println(string(b))
		return
	}
	entries, err := catalog.List(root)
	if err != nil {
		fail(fmt.Errorf("%s root %s: %w (skills/agents live in the frontier-ship source tree)", kind, root, err))
		return
	}
	fmt.Printf("%s (%d) — %s\n", strings.ToUpper(kind), len(entries), root)
	for _, e := range entries {
		fmt.Printf("  %-36s %s\n", e.Name, e.Title)
	}
	fmt.Printf("\nshow: frontier %s show <name>\n", kind)
}

func runOptimize(cwd string, args []string) {
	if len(args) == 0 {
		runOptimizeReport(cwd)
		return
	}
	switch strings.ToLower(args[0]) {
	case "report", "run":
		runOptimizeReport(cwd)
	case "status":
		runOptimizeStatus(cwd)
	case "pr-body", "prbody":
		if len(args) < 2 {
			fail(fmt.Errorf("usage: frontier optimize pr-body Opt-001"))
			return
		}
		runOptimizePRBody(cwd, args[1])
	case "help", "-h", "--help":
		printOptimizeStub()
	default:
		fmt.Fprintf(os.Stderr, "unknown optimize subcommand %q (try: report|status|pr-body)\n", args[0])
		os.Exit(2)
	}
}

func runOptimizeReport(cwd string) {
	fmt.Println(`╔══════════════════════════════════════════════╗
║  OPTIMIZE (O) — report (advise, no mutate)   ║
╚══════════════════════════════════════════════╝`)
	axiom("F0", "optimize.start", "programmatic hotspots; behavior must stay equivalent")
	r, err := optimize.BuildReport(cwd)
	if err != nil {
		fail(err)
		return
	}
	art, err := optimize.WriteArtifacts(r)
	if err != nil {
		fail(err)
		return
	}
	fmt.Printf("project:   %s\n", r.Name)
	fmt.Printf("findings:  %d (advise-only)\n", len(r.Findings))
	for _, f := range r.Findings {
		fmt.Printf("  - %s  %s  %s\n", f.ID, f.Path, f.Title)
	}
	fmt.Printf("\nbrief:  %s\njson:   %s\n", art.Markdown, art.JSON)
	fmt.Println("Next: frontier optimize pr-body Opt-001  → paste into a small PR")

	led, err := ledger.Open(findLedger(cwd))
	if err != nil {
		fail(err)
		return
	}
	ids := optimize.ListFindingIDs(r)
	_, _ = led.Append("frontier-git", "optimize.reported", map[string]any{
		"root":        r.Root,
		"name":        r.Name,
		"findings":    len(r.Findings),
		"ids":         ids,
		"brief":       art.Markdown,
		"json":        art.JSON,
		"blocks_gate": false,
	})
	axiom("F0", "ledger.append", "optimize.reported sealed")
	axiom("F4", "optimize.done", fmt.Sprintf("%d Opt-*(s)", len(r.Findings)))
}

func runOptimizeStatus(cwd string) {
	r, err := optimize.LoadLatest(cwd)
	if err != nil {
		fmt.Println(err.Error())
		led, e2 := ledger.Open(findLedger(cwd))
		if e2 != nil {
			return
		}
		rows, _ := led.Tail(40)
		n := 0
		for _, row := range rows {
			if strings.HasPrefix(row.Action, "optimize.") {
				fmt.Printf("%d %s %s %v\n", row.Seq, row.TS, row.Action, row.Payload)
				n++
			}
		}
		if n == 0 {
			fmt.Println("run: frontier optimize report")
		}
		return
	}
	fmt.Printf("latest optimize: %s  findings=%d\n", r.Stamp, len(r.Findings))
	for _, f := range r.Findings {
		fmt.Printf("  %s  [%s] %s:%d  %s\n", f.ID, f.Disposition, f.Path, f.StartLine, f.Title)
	}
}

func runOptimizePRBody(cwd, id string) {
	r, err := optimize.LoadLatest(cwd)
	if err != nil {
		fail(err)
		return
	}
	f, err := optimize.FindingByID(r, id)
	if err != nil {
		fail(err)
		return
	}
	fmt.Println("<!-- frontier optimize — paste as PR body; one Opt-ID per PR -->")
	fmt.Printf("## Optimize: %s\n\n", f.ID)
	fmt.Println(optimize.FormatFinding(*f))
	fmt.Println("---")
	fmt.Println("Branch suggestion: `frontier/opt-" + strings.ToLower(strings.ReplaceAll(f.ID, "Opt-", "")) + "-…`")
	fmt.Println("Remember: behavior-preserving only; run tests covering this function.")
}

func runSCM(cwd string, args []string) {
	sub := "status"
	if len(args) > 0 {
		sub = strings.ToLower(args[0])
	}
	switch sub {
	case "status", "init", "connect", "":
		printSCMStub(cwd, sub)
	default:
		fmt.Fprintf(os.Stderr, "unknown scm subcommand %q (try: status|init|connect)\n", args[0])
		os.Exit(2)
	}
}

func printSCMStub(cwd, sub string) {
	fmt.Println(`╔══════════════════════════════════════════════╗
║  SCM — onboarding (separate from Learn)      ║
╚══════════════════════════════════════════════╝
  Purpose: how code is managed (git / GitHub / none)
  When:    BEFORE learn if the customer has no VCS
  Docs:    english/SCM.md

  Planned:
    frontier scm status   — detect git + remotes + host
    frontier scm init     — local git init (human confirm)
    frontier scm connect  — guide remote setup (human auth)

  Never create remotes silently.`)
	fmt.Printf("\n  cwd: %s\n  requested: scm %s  (stub — detect coming next)\n", cwd, sub)
	// Light detect for dogfood visibility (read-only).
	repo := gitx.Repo{Dir: cwd}
	if b, err := repo.Branch(); err == nil && b != "" {
		fmt.Printf("  hint: git branch = %s (work tree looks present)\n", b)
	} else {
		fmt.Println("  hint: no git branch detected here — scm init may be needed")
	}
	fmt.Println("╚══════════════════════════════════════════════╝")
}

func runLearn(cwd string, args []string) {
	if len(args) == 0 {
		runLearnClassify(cwd)
		return
	}
	switch strings.ToLower(args[0]) {
	case "classify":
		root := cwd
		if len(args) > 1 {
			root = args[1]
		}
		runLearnClassify(root)
	case "status":
		runLearnStatus(cwd)
	default:
		fmt.Fprintf(os.Stderr, "unknown L subcommand %q (try: frontier L classify [path])\n", args[0])
		os.Exit(2)
	}
}

func runLearnClassify(root string) {
	fmt.Println(`╔══════════════════════════════════════════════╗
║  L CLASSIFY — learn before change (no mutate)║
╚══════════════════════════════════════════════╝`)
	axiom("F0", "learn.start", "programmatic landscape; no remote mutate")
	ls, err := learn.Classify(root)
	if err != nil {
		fail(err)
		return
	}
	art, err := learn.WriteArtifacts(ls)
	if err != nil {
		fail(err)
		return
	}
	fmt.Printf("name:       %s\n", ls.Name)
	fmt.Printf("kind:       %s (%s)\n", ls.Kind, ls.Confidence)
	fmt.Printf("git:        %v\n", ls.HasGit)
	fmt.Printf("compose:    %v %v\n", ls.HasCompose, ls.ComposeServices)
	fmt.Printf("files~:     %d\n", ls.FileCountApprox)
	if len(ls.Reasons) > 0 {
		fmt.Printf("why:        %s\n", strings.Join(ls.Reasons, "; "))
	}
	fmt.Printf("\nbrief:  %s\njson:   %s\n", art.Markdown, art.JSON)

	led, err := ledger.Open(findLedger(root))
	if err != nil {
		fail(err)
		return
	}
	_, _ = led.Append("frontier-git", "learn.classified", map[string]any{
		"root":        ls.Root,
		"name":        ls.Name,
		"kind":        ls.Kind,
		"confidence":  ls.Confidence,
		"has_git":     ls.HasGit,
		"has_compose": ls.HasCompose,
		"brief":       art.Markdown,
		"json":        art.JSON,
	})
	axiom("F0", "ledger.append", "learn.classified sealed")
	axiom("F4", "learn.done", ls.Kind)
}

func runLearnStatus(cwd string) {
	led, err := ledger.Open(findLedger(cwd))
	if err != nil {
		fail(err)
		return
	}
	rows, _ := led.Tail(40)
	n := 0
	for _, r := range rows {
		if strings.HasPrefix(r.Action, "learn.") {
			fmt.Printf("%d %s %s %v\n", r.Seq, r.TS, r.Action, r.Payload)
			n++
		}
	}
	latest := filepath.Join(cwd, ".frontier", "learn", "LATEST")
	if b, err := os.ReadFile(latest); err == nil {
		fmt.Printf("LATEST artifacts: %s\n", strings.TrimSpace(string(b)))
	}
	if n == 0 {
		fmt.Println("no learn.* seals yet — run: frontier L classify")
	}
}

func axiom(id, when, detail string) {
	fmt.Fprintf(os.Stderr, "AXIOM %-4s | %-28s | %s\n", id, when, detail)
}

func verbose() bool {
	return os.Getenv("FRONTIER_VERBOSE") == "1" || os.Getenv("FRONTIER_VERBOSE") == "true"
}

func runExam(cwd string, seal bool) ([]owasp.Finding, error) {
	fmt.Println(`╔══════════════════════════════════════════════╗
║  FRONTIER GUARD (G) — control=changeset      ║
╚══════════════════════════════════════════════╝`)
	axiom("F0", "exam.ledger", "evidence path open")
	axiom("F4", "exam.start", "Guard policy = OWASP Top 10 v0 (English→Haskell→Go)")

	findings, err := owasp.ScanTree(cwd)
	if err != nil {
		return nil, err
	}
	fmt.Println()
	fmt.Println(owasp.FormatReport(findings))
	fmt.Println()

	surfaces, _ := vscan.ListSecretSurfaces(cwd)
	if len(surfaces) == 0 {
		fmt.Println("secret surfaces: none named (.env/.pem/credentials…)")
	} else {
		fmt.Printf("secret surfaces: %d (names only — review under Guard)\n", len(surfaces))
		for _, s := range surfaces {
			fmt.Printf("  - %s\n", s)
		}
		fmt.Println()
	}

	block := owasp.BlocksGate(findings)
	disposition := "record"
	switch {
	case block:
		disposition = "block"
		axiom("F4", "disposition", "block — High/Critical under Guard")
	case len(findings) > 0 || len(surfaces) > 0:
		disposition = "advise"
		axiom("F4", "disposition", "advise — findings or secret surfaces present")
	default:
		axiom("F4", "disposition", "record — Clean under current Guard")
	}

	fmt.Printf("control_point: changeset\n")
	fmt.Printf("disposition:   %s\n", disposition)
	fmt.Printf("guard_policy:  OWASP-Top10-2021-v0\n")
	fmt.Printf("findings:      %d\n", len(findings))
	fmt.Printf("secret_paths:  %d\n", len(surfaces))
	fmt.Println("╚══════════════════════════════════════════════╝")

	if seal {
		led, err := ledger.Open(findLedger(cwd))
		if err != nil {
			return findings, err
		}
		_, _ = led.Append("frontier-git", "exam.owasp", map[string]any{
			"policy":          "OWASP-Top10-2021-v0",
			"control":         "changeset",
			"disposition":     disposition,
			"findings":        len(findings),
			"secret_surfaces": len(surfaces),
			"blocks_gate":     block,
		})
		axiom("F0", "ledger.append", "exam.owasp sealed")
	}
	axiom("F4", "exam.done", fmt.Sprintf("%d finding(s)", len(findings)))
	return findings, nil
}

func runGuardSub(cwd string, args []string) {
	if len(args) == 0 {
		runExam(cwd, true)
		return
	}
	sub := strings.ToLower(args[0])
	switch sub {
	case "list":
		fmt.Println(`╔══════════════════════════════════════════════╗
║     FRONTIER GUARD — programmatic scanners   ║
╚══════════════════════════════════════════════╝
name        builtin  available  notes
----        -------  ---------  -----`)
		for _, s := range vscan.Registry() {
			avail := "no"
			if s.Available() {
				avail = "yes"
			}
			builtin := "no"
			if s.Builtin() {
				builtin = "yes"
			}
			note := ""
			if !s.Available() && !s.Builtin() {
				note = "install or planned"
			}
			if s.Name() == "checkov" && !s.Available() {
				note = "pip install checkov"
			}
			fmt.Printf("%-11s %-7s  %-9s  %s\n", s.Name(), builtin, avail, note)
		}
		fmt.Println(`
Gate/plan still hard-block only on built-in owasp-v0 High/Critical.
Adapters enrich Guard / enhance briefs without burning model tokens.
Secret surfaces (.env, keys, …) are listed by: frontier guard`)
	default:
		sc, ok := vscan.Lookup(sub)
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown Guard scanner %q (try: frontier guard list)\n", sub)
			os.Exit(2)
		}
		fmt.Printf("╔══════════════════════════════════════════════╗\n║   FRONTIER GUARD — adapter %-12s     ║\n╚══════════════════════════════════════════════╝\n", sc.Name())
		axiom("F4", "exam.adapter", sc.Name())
		res, err := sc.Scan(cwd)
		if err != nil {
			fail(err)
			return
		}
		if res.Skipped {
			fmt.Printf("skipped: %s\n", res.SkipWhy)
			return
		}
		fmt.Printf("source: %s\nfindings: %d\n", res.Source, len(res.Findings))
		for _, f := range res.Findings {
			fmt.Printf("  [%s] %s %s:%d  %s\n", f.Severity, f.RuleID, f.Path, f.Line, f.Snippet)
		}
		led, err := ledger.Open(findLedger(cwd))
		if err == nil {
			_, _ = led.Append("frontier-git", "exam.adapter", map[string]any{
				"source": res.Source, "findings": len(res.Findings), "meta": res.Meta,
			})
			axiom("F0", "ledger.append", "exam.adapter sealed")
		}
	}
}

func runEnhance(cwd string, args []string) {
	if len(args) == 0 {
		fmt.Println(`enhance commands:
  frontier enhance guard     programmatic Guard pack + lean host brief
  frontier enhance optimize  CS speed residual (PLANNED stub)
  frontier enhance status    last enhance.* ledger seals
  frontier enhance seal PATH ingest host-model result JSON (advise)`)
		return
	}
	switch strings.ToLower(args[0]) {
	case "guard", "g", "v":
		runEnhanceGuard(cwd)
	case "optimize", "o":
		fmt.Println("enhance optimize: programmatic report first; host fills residual CS detail in PR.")
		runOptimizeReport(cwd)
	case "status":
		runEnhanceStatus(cwd)
	case "seal":
		if len(args) < 2 {
			fail(fmt.Errorf("usage: frontier enhance seal <result.json>"))
			return
		}
		runEnhanceSeal(cwd, args[1])
	default:
		fmt.Fprintf(os.Stderr, "unknown enhance subcommand %q\n", args[0])
		os.Exit(2)
	}
}

func runEnhanceGuard(cwd string) {
	fmt.Println(`╔══════════════════════════════════════════════╗
║  ENHANCE GUARD — programmatic first, host    ║
╚══════════════════════════════════════════════╝`)
	axiom("F0", "enhance.start", "build pack without tokens; hand residual to host model")
	opts := vscan.Options{}
	for _, name := range []string{"checkov", "gitleaks", "trivy"} {
		if s, ok := vscan.Lookup(name); ok && s.Available() {
			opts.Adapters = append(opts.Adapters, name)
		}
	}
	pack, err := vscan.BuildPack(cwd, opts)
	if err != nil {
		fail(err)
		return
	}
	art, err := vscan.WriteEnhanceBrief(cwd, pack)
	if err != nil {
		fail(err)
		return
	}
	fmt.Printf("disposition (programmatic): %s\n", pack.Disposition)
	fmt.Printf("findings: %d (brief shows <=%d)\n", len(pack.Findings), vscan.MaxBriefFindings)
	fmt.Printf("adapters: %s\n", strings.Join(pack.AdaptersRun, ", "))
	fmt.Printf("scope: %s\n", pack.ScopeMode)
	fmt.Printf("\nbrief:  %s\njson:   %s\n", art.Markdown, art.JSON)
	fmt.Println("\nHost (Grok / Fable / …): read the brief. Do residual work only. Then:")
	fmt.Println("  frontier enhance seal .frontier/enhance/<result>.json")
	led, err := ledger.Open(findLedger(cwd))
	if err != nil {
		fail(err)
		return
	}
	_, _ = led.Append("frontier-git", "enhance.requested", map[string]any{
		"control":               "changeset",
		"disposition":           pack.Disposition,
		"findings_programmatic": len(pack.Findings),
		"brief":                 art.Markdown,
		"json":                  art.JSON,
		"adapters":              pack.AdaptersRun,
		"token_budget": map[string]int{
			"max_findings": vscan.MaxBriefFindings,
			"max_paths":    vscan.MaxBriefPaths,
			"max_bytes":    vscan.MaxBriefBytes,
		},
	})
	axiom("F0", "ledger.append", "enhance.requested sealed")
	axiom("F4", "enhance.handoff", "waiting on host model — no gate change")
}

func runEnhanceStatus(cwd string) {
	led, err := ledger.Open(findLedger(cwd))
	if err != nil {
		fail(err)
		return
	}
	rows, _ := led.Tail(40)
	n := 0
	for _, r := range rows {
		if strings.HasPrefix(r.Action, "enhance.") {
			fmt.Printf("%d %s %s %v\n", r.Seq, r.TS, r.Action, r.Payload)
			n++
		}
	}
	if n == 0 {
		fmt.Println("no enhance.* seals yet — run: frontier enhance V")
	}
}

func runEnhanceSeal(cwd, path string) {
	payload, err := vscan.ReadSealFile(path)
	if err != nil {
		fail(err)
		return
	}
	disp := payload.DispositionSuggest
	if disp == "" || disp == "block" {
		// Enhance never auto-blocks gate; promote into V definitions instead.
		if disp == "block" {
			axiom("F4", "enhance.advise_only", "host suggested block — sealed as advise until V promotion")
		}
		disp = "advise"
	}
	led, err := ledger.Open(findLedger(cwd))
	if err != nil {
		fail(err)
		return
	}
	_, _ = led.Append("frontier-git", "enhance.completed", map[string]any{
		"summary":             payload.Summary,
		"findings":            len(payload.Findings),
		"disposition_suggest": disp,
		"tools_used":          payload.ToolsUsed,
		"residual_risk":       payload.ResidualRisk,
		"seal_path":           path,
		"blocks_gate":         false,
	})
	axiom("F0", "ledger.append", "enhance.completed sealed (advise)")
	fmt.Printf("enhance sealed: disposition=%s findings=%d\n", disp, len(payload.Findings))
	fmt.Println("note: does not change gate — promote durable rules into V to block.")
}

func printMockImport() {
	fmt.Println(`╔══════════════════════════════════════════════╗
║   MOCK V-IMPORTER (discussion → visible)     ║
║   Source seed: cyber skill table + OWASP     ║
╚══════════════════════════════════════════════╝
id                        control_point   disposition  note
------------------------  --------------  -----------  ----
CAPEC-66                  changeset       block        SQLi — gateable now (in Go V)
CAPEC-63                  changeset       block        XSS — partially gateable
OWASP-A01..A10            changeset       block/advise Top10 v0 implemented
PENT-DOMAIN-WEB           catalog         record       umbrella; speciate later
PENT-DOMAIN-API           catalog         record       umbrella
PENT-DOMAIN-AD            engagement      record       needs confirm / not push-regex
PENT-DOMAIN-NET           engagement      record       runtime/engagement
PENT-DOMAIN-CLOUD         review          advise       often config/IaC later
PENT-DOMAIN-LLM           changeset       advise       some patterns gateable later
PENT-DOMAIN-AGENT         changeset       advise       tool-abuse patterns later
ATTCK-TA0043              catalog         record       recon knowledge
ATTCK-TA0001              engagement      record       initial access confirm
ATTCK-TA0006              engagement      record       credential access
ATTCK-TA0008              engagement      record       lateral movement
ATTCK-TA0040              engagement      record       impact

Legend:
  changeset  = git frontier exam/gate
  review     = PR / human
  runtime    = deployed system
  engagement = offensive confirm (Argus-style later)
  catalog    = in V as knowledge only until materialized

Next real importer: harvest MITRE/CWE/CAPEC → same columns → Haskell V.
╚══════════════════════════════════════════════╝`)
}

func evaluateForShip(cwd string) (policy.GateResult, []owasp.Finding, error) {
	repo := gitx.Repo{Dir: cwd}
	b, _ := repo.Branch()
	h, _ := repo.RevParseHead()
	p, _ := repo.StatusPorcelain()
	findings, err := runExam(cwd, true)
	if err != nil {
		return policy.GateResult{}, nil, err
	}
	g := policy.EvaluatePushGate(b, h, p, false)
	g.Branch, g.Head = b, h
	g.Dirty = policy.DirtyPorcelain(p)
	if owasp.BlocksGate(findings) {
		g.OK = false
		g.Reasons = append(g.Reasons, "OWASP V: untriaged High/Critical finding(s)")
		axiom("F4", "exam.block", "High/Critical under V blocks ship")
	}
	hrep := inspectHygieneQuiet(cwd)
	fmt.Printf("Hygiene (H): disposition=%s  scanned=%d  suspicious=%d  healthy=%v\n",
		hygiene.Disposition(hrep), hrep.Scanned, hrep.Suspicious, hrep.Healthy)
	if hygiene.BlocksGate(hrep) {
		g.OK = false
		g.Reasons = append(g.Reasons, "Hygiene H: untriaged provenance marks (FRONTIER_HYGIENE_BLOCK=1)")
		axiom("F4", "hygiene.block", "operator asked Hygiene to fail closed")
	}
	if strings.EqualFold(b, "main") || strings.EqualFold(b, "master") {
		axiom("F1", "harm.boundary", "refuse direct ship to main/master")
	}
	return g, findings, nil
}

// runPlan = terraform plan: preview only; nothing remote; fail closed.
func runPlan(cwd string, exitNonZero bool) {
	fmt.Println(`╔══════════════════════════════════════════════╗
║  PLAN (like terraform plan) — V enforced     ║
║  S (Slim): not enforced yet                  ║
╚══════════════════════════════════════════════╝`)
	axiom("F0", "plan.start", "preview ship decision; no remote mutate")
	g, _, err := evaluateForShip(cwd)
	if err != nil {
		fail(err)
		return
	}
	led, err := ledger.Open(findLedger(cwd))
	if err != nil {
		fail(err)
		return
	}
	sealed, err := policy.SealPlan(led, "frontier-git", g)
	if err != nil {
		fail(err)
		return
	}
	fmt.Println()
	if sealed.OK {
		fmt.Println("Plan: OK — may run: git frontier apply")
		axiom("F0", "plan.passed", sealed.SealHash)
	} else {
		fmt.Println("Plan: FAILED — fix issues; nothing will apply/push")
		fmt.Printf("Reasons: %v\n", sealed.Reasons)
		axiom("F0", "plan.failed", strings.Join(sealed.Reasons, "; "))
		axiom("F3", "continuity", "fail closed — like terraform")
	}
	fmt.Printf("ok=%v seal=%s branch=%s head=%s\n", sealed.OK, sealed.SealHash, sealed.Branch, sealed.Head)
	fmt.Println("╚══════════════════════════════════════════════╝")
	if !sealed.OK && exitNonZero {
		os.Exit(2)
	}
}

// runApply = terraform apply: only if fresh plan.passed; seals gate.passed.
func runApply(cwd string, exitNonZero bool) {
	fmt.Println(`╔══════════════════════════════════════════════╗
║  APPLY (like terraform apply)                ║
╚══════════════════════════════════════════════╝`)
	axiom("F0", "apply.start", "authorize push only from successful plan")
	repo := gitx.Repo{Dir: cwd}
	b, _ := repo.Branch()
	h, _ := repo.RevParseHead()
	led, err := ledger.Open(findLedger(cwd))
	if err != nil {
		fail(err)
		return
	}
	ok, detail := policy.FreshPlanOK(led, b, h)
	if !ok {
		axiom("F0", "apply.deny", detail)
		fmt.Printf("Apply refused: %s\n", detail)
		if exitNonZero {
			os.Exit(2)
		}
		return
	}
	// Re-validate (refresh) like a careful apply.
	g, _, err := evaluateForShip(cwd)
	if err != nil {
		fail(err)
		return
	}
	if !g.OK {
		_, _ = policy.SealGate(led, "frontier-git", g)
		axiom("F0", "apply.deny", strings.Join(g.Reasons, "; "))
		fmt.Printf("Apply refused after refresh: %v\n", g.Reasons)
		if exitNonZero {
			os.Exit(2)
		}
		return
	}
	sealed, err := policy.SealGate(led, "frontier-git", g)
	if err != nil {
		fail(err)
		return
	}
	axiom("F0", "gate.passed", sealed.SealHash)
	axiom("F2", "ready", "authorized human may git push")
	fmt.Printf("Apply: OK — sealed gate.passed\nplan_seal=%s gate_seal=%s\n", detail, sealed.SealHash)
	fmt.Println("Next: git push")
	fmt.Println("╚══════════════════════════════════════════════╝")
}

func printDemo(cwd string) {
	repo := gitx.Repo{Dir: cwd}
	b, _ := repo.Branch()
	h, _ := repo.RevParseHead()
	p, _ := repo.StatusPorcelain()
	dirty := policy.DirtyPorcelain(p)
	g := policy.EvaluatePushGate(b, h, p, false)
	ledPath := findLedger(cwd)
	var lastGate string
	if led, err := ledger.Open(ledPath); err == nil {
		if e, _ := led.LastAction("gate.passed"); e != nil {
			lastGate = "gate.passed@" + e.EntryHash[:12]
		} else if e, _ := led.LastAction("gate.failed"); e != nil {
			lastGate = "gate.failed"
		} else {
			lastGate = "(none yet)"
		}
	} else {
		lastGate = "(no ledger)"
	}

	onMain := strings.EqualFold(b, "main") || strings.EqualFold(b, "master")
	fmt.Println(`╔══════════════════════════════════════════════╗
║           FRONTIER  —  visible test          ║
╚══════════════════════════════════════════════╝`)
	fmt.Printf("  branch     %s\n", nz(b, "(none)"))
	fmt.Printf("  HEAD       %s\n", short(h))
	fmt.Printf("  dirty      %v\n", dirty)
	fmt.Printf("  on_main    %v\n", onMain)
	fmt.Printf("  gate_now   ok=%v  %v\n", g.OK, g.Reasons)
	fmt.Printf("  last_seal  %s\n", lastGate)
	fmt.Printf("  ledger     %s\n", ledPath)
	fmt.Println()
	fmt.Println("  ladder     Observer → Analyst → Operator → Executor")
	fmt.Println("  push?      only Executor + fresh gate.passed + feature branch")
	fmt.Println("  languages  English · Haskell · Go   (draft in any, prove in Haskell)")
	fmt.Println("  minimality least code that still proves the result")
	fmt.Println()
	if g.OK {
		fmt.Println("  SEE: gate would PASS right now. Next: git push (if role allows).")
	} else {
		fmt.Println("  SEE: gate would FAIL right now. Fix reasons, then: git frontier gate")
	}
	fmt.Println("╚══════════════════════════════════════════════╝")
}

func short(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	if h == "" {
		return "(none)"
	}
	return h
}

func nz(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}

func findLedger(cwd string) string {
	if v := os.Getenv("FRONTIER_LEDGER"); v != "" {
		return v
	}
	// Prefer in-repo ledger only if it already exists (user opted in).
	dir := cwd
	for {
		cand := filepath.Join(dir, ".frontier", "ledger.jsonl")
		if _, err := os.Stat(cand); err == nil {
			return cand
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	// Default: ledger OUTSIDE the work tree so evidence never dirties the diff.
	// (Local-first, still on disk — not cloud.)
	sum := sha256Short(cwd)
	root := os.Getenv("FRONTIER_HOME")
	if root == "" {
		root = filepath.Join("D:\\frontier", "ledgers")
	}
	return filepath.Join(root, sum, "ledger.jsonl")
}

func sha256Short(s string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(s))))
	return hex.EncodeToString(sum[:8])
}

func findRealGit() string {
	if v := os.Getenv("FRONTIER_GIT_BIN"); v != "" {
		return v
	}
	// Prefer system Git for Windows, not ourselves.
	candidates := []string{
		`C:\Program Files\Git\cmd\git.exe`,
		`C:\Program Files\Git\bin\git.exe`,
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	if p, err := exec.LookPath("git"); err == nil {
		// Avoid infinite recursion if frontier-git is named git on PATH
		if abs, err2 := filepath.Abs(os.Args[0]); err2 == nil {
			if filepath.Clean(p) == filepath.Clean(abs) {
				fmt.Fprintln(os.Stderr, "frontier: set FRONTIER_GIT_BIN to real git")
				os.Exit(127)
			}
		}
		return p
	}
	fmt.Fprintln(os.Stderr, "frontier: real git not found")
	os.Exit(127)
	return ""
}

func runPassthrough(git string, args []string) int {
	cmd := exec.Command(git, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode()
	}
	fmt.Fprintln(os.Stderr, err)
	return 1
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
