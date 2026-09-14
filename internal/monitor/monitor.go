// Package monitor audits Frontier ledger evidence against the ship directives.
//
// It is the examination engine behind `frontier monitor`: whenever an agent uses
// frontier-ship, its consequential actions are sealed into the ledger (F0), and
// the monitor replays those rows to verify the agent followed the directives
// (feature branch, plan -> apply -> push, no soft bypass, no blocked ship).
//
// Language layers (F5):
//   - English:  english/MONITOR.md            (what we mean)
//   - Haskell:  haskell/src/Frontier/Monitor.hs (pure witness of the replay rules)
//   - Go:       this package                   (what runs)
//
// Read-only over ledger files; the CLI seals one monitor.audited row per run.
package monitor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Wadek/frontier-ship/internal/ledger"
	"github.com/Wadek/frontier-ship/internal/policy"
)

// Severity orders findings: tamper > violation > incident > advisory.
type Severity string

const (
	SevTamper    Severity = "tamper"
	SevViolation Severity = "violation"
	SevIncident  Severity = "incident"
	SevAdvisory  Severity = "advisory"
)

func sevRank(s Severity) int {
	switch s {
	case SevTamper:
		return 4
	case SevViolation:
		return 3
	case SevIncident:
		return 2
	case SevAdvisory:
		return 1
	}
	return 0
}

// Directive is one named ship rule the monitor verifies against ledger rows.
type Directive struct {
	ID   string
	Text string
}

// Directives is the v0 reference set. D0 (cryptographic chain integrity) is
// checked here in Go; the Haskell layer witnesses D1-D4 in pure form.
var Directives = []Directive{
	{ID: "D0", Text: "F0 evidence: ledger hash chain is intact (prev_hash links, entry_hash recomputes)"},
	{ID: "D1", Text: "no remote effect without a fresh sealed gate: push.authorized needs a fresh gate.passed that itself follows a fresh plan.passed (same branch+HEAD, 15 min TTL)"},
	{ID: "D2", Text: "no ship from main/master: plan/gate/push/commit rows must be on feature branches"},
	{ID: "D3", Text: "no soft bypass: push.soft_allow (FRONTIER_SOFT=1) is forbidden for real ship"},
	{ID: "D4", Text: "no ship with untriaged High/Critical under V: gate.passed must not follow a blocking exam.owasp for the same branch+HEAD"},
	{ID: "D5", Text: "denied attempts are incidents: push.deny, apply.deny, commit.deny_main, plan.failed, gate.failed"},
	{ID: "D6", Text: "hygiene service reachable during inspect: hygiene.service_down is an advisory"},
	{ID: "D7", Text: "runtime chaos stays dry: chaos injection is not implemented; chaos_denied is an advisory"},
}

// Finding is one directive violation, incident, advisory, or tamper sighting.
type Finding struct {
	Directive string `json:"directive"`
	Severity  string `json:"severity"`
	Seq       int64  `json:"seq,omitempty"`
	Actor     string `json:"actor,omitempty"`
	Action    string `json:"action,omitempty"`
	Reason    string `json:"reason"`
}

// Report is the audit of one ledger file.
type Report struct {
	Path           string    `json:"path"`
	Rows           int       `json:"rows"`
	ChainOK        bool      `json:"chain_ok"`
	Actors         []string  `json:"actors,omitempty"`
	CompliantFlows int       `json:"compliant_flows"`
	Findings       []Finding `json:"findings"`
	Verdict        string    `json:"verdict"`
}

// LedgersRoot is where evidence ledgers live (FRONTIER_HOME/ledgers).
func LedgersRoot() string {
	if v := strings.TrimSpace(os.Getenv("FRONTIER_HOME")); v != "" {
		return filepath.Join(v, "ledgers")
	}
	return filepath.Join("D:\\frontier", "ledgers")
}

func isMain(branch string) bool {
	b := strings.ToLower(strings.TrimSpace(branch))
	return b == "main" || b == "master"
}

// sealKey identifies one branch+HEAD pair.
func sealKey(branch, head string) string { return branch + "\x00" + head }

func tsOf(e ledger.Entry) time.Time {
	t, err := time.Parse(time.RFC3339Nano, e.TS)
	if err != nil {
		t, err = time.Parse(time.RFC3339, e.TS)
	}
	if err != nil {
		return time.Time{}
	}
	return t
}

// freshDelta reports whether two row timestamps sit inside the gate TTL.
// Unparseable timestamps count as fresh (avoid false violations on old rows).
func freshDelta(after, before time.Time) bool {
	if after.IsZero() || before.IsZero() {
		return true
	}
	return after.Sub(before) <= policy.GateTTL
}

// AuditLedger reads one ledger file and replays its rows against the directives.
func AuditLedger(path string) (*Report, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	rep := &Report{Path: path, ChainOK: true, Findings: nil}
	lines := strings.Split(string(raw), "\n")
	var rows []ledger.Entry
	actorSeen := map[string]bool{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var e ledger.Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			rep.ChainOK = false
			rep.Findings = append(rep.Findings, Finding{Directive: "D0", Severity: string(SevTamper), Reason: "unparseable row: " + truncate(line, 80)})
			continue
		}
		rows = append(rows, e)
		if e.Actor != "" {
			actorSeen[e.Actor] = true
		}
	}
	rep.Rows = len(rows)
	for actor := range actorSeen {
		rep.Actors = append(rep.Actors, actor)
	}
	sort.Strings(rep.Actors)

	// D0 — chain integrity.
	prev := "genesis"
	for i := range rows {
		e := &rows[i]
		if e.PrevHash != prev {
			rep.ChainOK = false
			rep.Findings = append(rep.Findings, Finding{
				Directive: "D0", Severity: string(SevTamper), Seq: e.Seq, Actor: e.Actor, Action: e.Action,
				Reason: fmt.Sprintf("prev_hash mismatch at seq %d", e.Seq),
			})
		}
		if ledger.EntryHashOf(*e) != e.EntryHash {
			rep.ChainOK = false
			rep.Findings = append(rep.Findings, Finding{
				Directive: "D0", Severity: string(SevTamper), Seq: e.Seq, Actor: e.Actor, Action: e.Action,
				Reason: fmt.Sprintf("entry_hash does not recompute at seq %d", e.Seq),
			})
		}
		prev = e.EntryHash
	}

	// D1-D7 — chronological replay.
	planPassed := map[string]time.Time{}
	gatePassed := map[string]time.Time{}
	examBlocked := map[string]bool{}

	add := func(sev Severity, dir string, e ledger.Entry, reason string) {
		rep.Findings = append(rep.Findings, Finding{
			Directive: dir, Severity: string(sev), Seq: e.Seq, Actor: e.Actor, Action: e.Action, Reason: reason,
		})
	}
	branchOf := func(e ledger.Entry) string {
		b, _ := e.Payload["branch"].(string)
		return b
	}
	headOf := func(e ledger.Entry) string {
		h, _ := e.Payload["head"].(string)
		return h
	}

	for i := range rows {
		e := &rows[i]
		ts := tsOf(*e)
		branch, head := branchOf(*e), headOf(*e)
		key := sealKey(branch, head)
		switch e.Action {
		case "plan.passed":
			if isMain(branch) {
				add(SevViolation, "D2", *e, "plan sealed from "+branch)
			}
			planPassed[key] = ts
		case "plan.failed":
			add(SevIncident, "D5", *e, "plan failed: "+strings.Join(reasonsOf(*e), "; "))
		case "gate.passed":
			if isMain(branch) {
				add(SevViolation, "D2", *e, "gate sealed from "+branch)
			}
			if p, ok := planPassed[key]; !ok || !freshDelta(ts, p) {
				add(SevViolation, "D1", *e, "gate.passed without a fresh plan.passed for this branch+HEAD")
			}
			if examBlocked[key] {
				add(SevViolation, "D4", *e, "gate.passed while a blocking exam.owasp stands for this branch+HEAD")
			}
			gatePassed[key] = ts
		case "gate.failed":
			add(SevIncident, "D5", *e, "gate failed: "+strings.Join(reasonsOf(*e), "; "))
		case "push.authorized":
			if isMain(branch) {
				add(SevViolation, "D2", *e, "push authorized from "+branch)
			}
			if g, ok := gatePassed[key]; !ok || !freshDelta(ts, g) {
				add(SevViolation, "D1", *e, "push.authorized without a fresh gate.passed for this branch+HEAD")
			} else {
				rep.CompliantFlows++
			}
		case "push.soft_allow":
			add(SevViolation, "D3", *e, "FRONTIER_SOFT=1 soft-allow of a would-be deny (turn FRONTIER_SOFT off)")
		case "push.deny":
			add(SevIncident, "D5", *e, "push denied: "+firstReason(*e))
		case "apply.deny":
			add(SevIncident, "D5", *e, "apply denied: "+firstReason(*e))
		case "commit.deny_main":
			add(SevIncident, "D5", *e, "commit on main/master attempted and denied")
		case "commit.attempt":
			if isMain(branch) {
				add(SevViolation, "D2", *e, "commit attempted on "+branch)
			}
		case "exam.owasp":
			block, _ := e.Payload["blocks_gate"].(bool)
			examBlocked[key] = block
		case "hygiene.service_down":
			add(SevAdvisory, "D6", *e, "watermarks-remover unreachable during hygiene inspect")
		case "runtime.chaos_denied":
			add(SevAdvisory, "D7", *e, "chaos plan denied/dry — v1 never injects")
		case "runtime.chaos_injected":
			add(SevViolation, "D7", *e, "chaos injection executed (v1 must not inject)")
		}
	}

	rep.Verdict = VerdictName(rep)
	return rep, nil
}

func reasonsOf(e ledger.Entry) []string {
	switch v := e.Payload["reasons"].(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, x := range v {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	if s, ok := e.Payload["reason"].(string); ok && s != "" {
		return []string{s}
	}
	return nil
}

func firstReason(e ledger.Entry) string {
	if r := reasonsOf(e); len(r) > 0 {
		return truncate(strings.Join(r, "; "), 140)
	}
	return "no reason recorded"
}

// VerdictName computes the single-word verdict for one report.
func VerdictName(r *Report) string {
	worst := SevAdvisory
	if !r.ChainOK {
		worst = SevTamper
	}
	for _, f := range r.Findings {
		if sevRank(Severity(f.Severity)) > sevRank(worst) {
			worst = Severity(f.Severity)
		}
	}
	switch worst {
	case SevTamper:
		return "tampered"
	case SevViolation:
		return "violation"
	case SevIncident:
		return "watch"
	default:
		return "clean"
	}
}

// AggregateVerdict returns the worst verdict across several reports.
func AggregateVerdict(reps []*Report) string {
	order := map[string]int{"clean": 0, "watch": 1, "violation": 2, "tampered": 3}
	worst := "clean"
	for _, r := range reps {
		if order[r.Verdict] > order[worst] {
			worst = r.Verdict
		}
	}
	return worst
}

// AuditAll audits every ledger under the ledgers root.
func AuditAll(root string) ([]*Report, error) {
	pattern := filepath.Join(root, "*", "ledger.jsonl")
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	var reps []*Report
	for _, p := range paths {
		rep, err := AuditLedger(p)
		if err != nil {
			continue // one unreadable ledger must not sink the sweep
		}
		reps = append(reps, rep)
	}
	return reps, nil
}

// FormatReport is the terminal view of one audited ledger.
func FormatReport(r *Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "ledger:  %s\n", r.Path)
	fmt.Fprintf(&b, "rows:    %d\n", r.Rows)
	fmt.Fprintf(&b, "chain:   %v\n", r.ChainOK)
	if len(r.Actors) > 0 {
		fmt.Fprintf(&b, "actors:  %s\n", strings.Join(r.Actors, ", "))
	}
	fmt.Fprintf(&b, "compliant flows: %d\n", r.CompliantFlows)
	if len(r.Findings) == 0 {
		fmt.Fprintf(&b, "verdict: %s (no findings)\n", r.Verdict)
		return b.String()
	}
	fmt.Fprintf(&b, "findings: %d\n", len(r.Findings))
	for _, f := range r.Findings {
		loc := ""
		if f.Seq > 0 {
			loc = fmt.Sprintf(" seq=%d %s %s", f.Seq, f.Actor, f.Action)
		}
		fmt.Fprintf(&b, "  [%s %s]%s  %s\n", f.Severity, f.Directive, loc, f.Reason)
	}
	fmt.Fprintf(&b, "verdict: %s\n", r.Verdict)
	return b.String()
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
