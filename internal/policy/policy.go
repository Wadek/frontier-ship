package policy

import (
	"fmt"
	"strings"
	"time"

	"github.com/Wadek/frontier-ship/internal/ledger"
	"github.com/Wadek/frontier-ship/internal/role"
)

// Deny > Ask > Allow evaluated in Require.
func Require(have role.Role, min role.Role, tool string) error {
	if !have.Can(min) {
		return fmt.Errorf("deny: tool %s needs role >= %s (have %s)", tool, min, have)
	}
	return nil
}

// GateResult is a sealed pre-push conscience check.
type GateResult struct {
	OK        bool           `json:"ok"`
	Reasons   []string       `json:"reasons"`
	Branch    string         `json:"branch"`
	Head      string         `json:"head"`
	Dirty     bool           `json:"dirty"`
	SealHash  string         `json:"seal_hash,omitempty"`
	ExpiresAt string         `json:"expires_at,omitempty"`
	Extras    map[string]any `json:"extras,omitempty"`
}

// GateTTL is how long a plan.passed / gate.passed seal stays fresh for push.
// Exported so the monitor can re-verify freshness from ledger timestamps.
const GateTTL = 15 * time.Minute

// DirtyPorcelain reports whether porcelain output has real changes,
// ignoring Frontier's own ledger/metadata under .frontier/.
func DirtyPorcelain(porcelain string) bool {
	for _, line := range strings.Split(porcelain, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		// porcelain: XY<space>path (path starts at index 3 when standard)
		path := line
		if len(line) >= 3 {
			path = strings.TrimSpace(line[3:])
		}
		path = strings.TrimPrefix(path, "\"")
		if strings.HasPrefix(path, ".frontier/") || path == ".frontier" {
			continue
		}
		return true
	}
	return false
}

// EvaluatePushGate is local-first and cheap: no model calls.
func EvaluatePushGate(branch, head, porcelain string, allowDirty bool) GateResult {
	var reasons []string
	dirty := DirtyPorcelain(porcelain)
	if branch == "" {
		reasons = append(reasons, "detached HEAD or empty branch")
	}
	if head == "" {
		reasons = append(reasons, "missing HEAD")
	}
	if dirty && !allowDirty {
		reasons = append(reasons, "working tree dirty; commit or clean before push")
	}
	if strings.EqualFold(branch, "main") || strings.EqualFold(branch, "master") {
		reasons = append(reasons, "refusing direct push to main/master (use a feature branch)")
	}
	ok := len(reasons) == 0
	return GateResult{
		OK:      ok,
		Reasons: reasons,
		Branch:  branch,
		Head:    head,
		Dirty:   dirty,
	}
}

// SealGate writes gate.passed or gate.failed to the ledger.
func SealGate(l *ledger.Ledger, actor string, g GateResult) (*GateResult, error) {
	action := "gate.failed"
	if g.OK {
		action = "gate.passed"
		g.ExpiresAt = time.Now().UTC().Add(GateTTL).Format(time.RFC3339)
	}
	payload := map[string]any{
		"ok":         g.OK,
		"reasons":    g.Reasons,
		"branch":     g.Branch,
		"head":       g.Head,
		"dirty":      g.Dirty,
		"expires_at": g.ExpiresAt,
	}
	e, err := l.Append(actor, action, payload)
	if err != nil {
		return nil, err
	}
	g.SealHash = e.EntryHash
	return &g, nil
}

// FreshGateOK reports whether a recent gate.passed matches current head/branch.
func FreshGateOK(l *ledger.Ledger, branch, head string) (bool, string) {
	return freshAction(l, "gate.passed", branch, head, "run: git frontier apply  (or gate)")
}

// FreshPlanOK reports whether a recent plan.passed matches current head/branch.
func FreshPlanOK(l *ledger.Ledger, branch, head string) (bool, string) {
	return freshAction(l, "plan.passed", branch, head, "run: git frontier plan")
}

func freshAction(l *ledger.Ledger, action, branch, head, hint string) (bool, string) {
	e, err := l.LastAction(action)
	if err != nil || e == nil {
		return false, "no " + action + " in ledger; " + hint
	}
	exp, _ := e.Payload["expires_at"].(string)
	if exp != "" {
		t, err := time.Parse(time.RFC3339, exp)
		if err == nil && time.Now().UTC().After(t) {
			return false, action + " expired; " + hint
		}
	}
	b, _ := e.Payload["branch"].(string)
	h, _ := e.Payload["head"].(string)
	if b != branch || h != head {
		return false, action + " was for different branch/HEAD; " + hint
	}
	return true, e.EntryHash
}

// SealPlan writes plan.passed or plan.failed (Terraform-like preview state).
func SealPlan(l *ledger.Ledger, actor string, g GateResult) (*GateResult, error) {
	action := "plan.failed"
	if g.OK {
		action = "plan.passed"
		g.ExpiresAt = time.Now().UTC().Add(GateTTL).Format(time.RFC3339)
	}
	payload := map[string]any{
		"ok":         g.OK,
		"reasons":    g.Reasons,
		"branch":     g.Branch,
		"head":       g.Head,
		"dirty":      g.Dirty,
		"expires_at": g.ExpiresAt,
		"V":          "OWASP-Top10-2021-v0",
		"S":          "not_enforced_yet",
	}
	e, err := l.Append(actor, action, payload)
	if err != nil {
		return nil, err
	}
	g.SealHash = e.EntryHash
	return &g, nil
}
