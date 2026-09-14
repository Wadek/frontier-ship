package monitor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Wadek/frontier-ship/internal/ledger"
)

// mkRow builds one hand-crafted row with a valid evidence hash.
func mkRow(seq int64, ts, action, branch, head, prev string) ledger.Entry {
	e := ledger.Entry{
		Seq: seq, TS: ts, Actor: "frontier-git", Action: action,
		Payload: map[string]any{"branch": branch, "head": head}, PrevHash: prev,
	}
	e.EntryHash = ledger.EntryHashOf(e)
	return e
}

func writeRows(t *testing.T, path string, rows []ledger.Entry) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer f.Close()
	for _, e := range rows {
		b, err := json.Marshal(e)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if _, err := f.Write(append(b, '\n')); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
}

func ts(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

// chainOf builds a valid chain (prev hashes link) from rows lacking hashes.
func chainOf(rows []ledger.Entry) []ledger.Entry {
	prev := "genesis"
	for i := range rows {
		rows[i].PrevHash = prev
		rows[i].EntryHash = ledger.EntryHashOf(rows[i])
		prev = rows[i].EntryHash
	}
	return rows
}

func TestAuditCompliantFlow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ledger.jsonl")
	now := time.Now()
	rows := chainOf([]ledger.Entry{
		{Seq: 1, TS: ts(now), Actor: "frontier-git", Action: "plan.passed", Payload: map[string]any{"branch": "frontier/x", "head": "h1"}},
		{Seq: 2, TS: ts(now.Add(time.Second)), Actor: "frontier-git", Action: "gate.passed", Payload: map[string]any{"branch": "frontier/x", "head": "h1"}},
		{Seq: 3, TS: ts(now.Add(2 * time.Second)), Actor: "frontier-git", Action: "push.authorized", Payload: map[string]any{"branch": "frontier/x", "head": "h1"}},
	})
	writeRows(t, path, rows)

	rep, err := AuditLedger(path)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if !rep.ChainOK {
		t.Errorf("chain should verify, findings: %+v", rep.Findings)
	}
	if rep.Verdict != "clean" {
		t.Errorf("verdict = %s, want clean; findings: %+v", rep.Verdict, rep.Findings)
	}
	if rep.CompliantFlows != 1 {
		t.Errorf("compliant flows = %d, want 1", rep.CompliantFlows)
	}
}

func TestAuditPushWithoutGate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ledger.jsonl")
	writeRows(t, path, chainOf([]ledger.Entry{
		{Seq: 1, TS: ts(time.Now()), Actor: "frontier-git", Action: "push.authorized", Payload: map[string]any{"branch": "frontier/x", "head": "h1"}},
	}))

	rep, err := AuditLedger(path)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if rep.Verdict != "violation" {
		t.Fatalf("verdict = %s, want violation", rep.Verdict)
	}
	if len(rep.Findings) != 1 || rep.Findings[0].Directive != "D1" {
		t.Errorf("want one D1 finding, got: %+v", rep.Findings)
	}
}

func TestAuditGateWithoutPlan(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ledger.jsonl")
	now := time.Now()
	writeRows(t, path, chainOf([]ledger.Entry{
		{Seq: 1, TS: ts(now), Actor: "frontier-git", Action: "gate.passed", Payload: map[string]any{"branch": "frontier/x", "head": "h1"}},
		{Seq: 2, TS: ts(now.Add(time.Second)), Actor: "frontier-git", Action: "push.authorized", Payload: map[string]any{"branch": "frontier/x", "head": "h1"}},
	}))

	rep, err := AuditLedger(path)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if rep.Verdict != "violation" {
		t.Fatalf("verdict = %s, want violation", rep.Verdict)
	}
	found := false
	for _, f := range rep.Findings {
		if f.Directive == "D1" && f.Action == "gate.passed" {
			found = true
		}
	}
	if !found {
		t.Errorf("want D1 on gate.passed without plan, got: %+v", rep.Findings)
	}
}

func TestAuditSoftAllow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ledger.jsonl")
	writeRows(t, path, chainOf([]ledger.Entry{
		{Seq: 1, TS: ts(time.Now()), Actor: "frontier-git", Action: "push.soft_allow", Payload: map[string]any{"branch": "frontier/x", "head": "h1"}},
	}))

	rep, err := AuditLedger(path)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if rep.Verdict != "violation" {
		t.Fatalf("verdict = %s, want violation", rep.Verdict)
	}
	if len(rep.Findings) != 1 || rep.Findings[0].Directive != "D3" {
		t.Errorf("want one D3 finding, got: %+v", rep.Findings)
	}
}

func TestAuditDeniedAttemptsAreWatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ledger.jsonl")
	writeRows(t, path, chainOf([]ledger.Entry{
		{Seq: 1, TS: ts(time.Now()), Actor: "frontier-git", Action: "plan.failed", Payload: map[string]any{"branch": "frontier/x", "head": "h1", "reasons": []string{"working tree dirty"}}},
		{Seq: 2, TS: ts(time.Now()), Actor: "frontier-git", Action: "push.deny", Payload: map[string]any{"branch": "frontier/x", "head": "h1", "reason": "no gate"}},
	}))

	rep, err := AuditLedger(path)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if rep.Verdict != "watch" {
		t.Fatalf("verdict = %s, want watch", rep.Verdict)
	}
	for _, f := range rep.Findings {
		if f.Directive != "D5" || f.Severity != "incident" {
			t.Errorf("want D5 incidents only, got: %+v", rep.Findings)
		}
	}
}

func TestAuditMainBranchShip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ledger.jsonl")
	writeRows(t, path, chainOf([]ledger.Entry{
		{Seq: 1, TS: ts(time.Now()), Actor: "frontier-git", Action: "plan.passed", Payload: map[string]any{"branch": "main", "head": "h1"}},
	}))

	rep, err := AuditLedger(path)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if rep.Verdict != "violation" {
		t.Fatalf("verdict = %s, want violation", rep.Verdict)
	}
	if len(rep.Findings) != 1 || rep.Findings[0].Directive != "D2" {
		t.Errorf("want one D2 finding, got: %+v", rep.Findings)
	}
}

func TestAuditExamBlockThenGate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ledger.jsonl")
	now := time.Now()
	writeRows(t, path, chainOf([]ledger.Entry{
		{Seq: 1, TS: ts(now), Actor: "frontier-git", Action: "plan.passed", Payload: map[string]any{"branch": "frontier/x", "head": "h1"}},
		{Seq: 2, TS: ts(now.Add(time.Second)), Actor: "frontier-git", Action: "exam.owasp", Payload: map[string]any{"branch": "frontier/x", "head": "h1", "blocks_gate": true}},
		{Seq: 3, TS: ts(now.Add(2 * time.Second)), Actor: "frontier-git", Action: "gate.passed", Payload: map[string]any{"branch": "frontier/x", "head": "h1"}},
	}))

	rep, err := AuditLedger(path)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if rep.Verdict != "violation" {
		t.Fatalf("verdict = %s, want violation", rep.Verdict)
	}
	found := false
	for _, f := range rep.Findings {
		if f.Directive == "D4" {
			found = true
		}
	}
	if !found {
		t.Errorf("want D4 finding, got: %+v", rep.Findings)
	}
}

func TestAuditExpiredGate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ledger.jsonl")
	now := time.Now()
	writeRows(t, path, chainOf([]ledger.Entry{
		{Seq: 1, TS: ts(now), Actor: "frontier-git", Action: "plan.passed", Payload: map[string]any{"branch": "frontier/x", "head": "h1"}},
		{Seq: 2, TS: ts(now.Add(time.Second)), Actor: "frontier-git", Action: "gate.passed", Payload: map[string]any{"branch": "frontier/x", "head": "h1"}},
		{Seq: 3, TS: ts(now.Add(16 * time.Minute)), Actor: "frontier-git", Action: "push.authorized", Payload: map[string]any{"branch": "frontier/x", "head": "h1"}},
	}))

	rep, err := AuditLedger(path)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if rep.Verdict != "violation" {
		t.Fatalf("verdict = %s, want violation", rep.Verdict)
	}
	found := false
	for _, f := range rep.Findings {
		if f.Directive == "D1" && f.Action == "push.authorized" {
			found = true
		}
	}
	if !found {
		t.Errorf("want D1 on push after expired gate, got: %+v", rep.Findings)
	}
}

func TestAuditTamperedChain(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ledger.jsonl")
	now := time.Now()
	rows := chainOf([]ledger.Entry{
		{Seq: 1, TS: ts(now), Actor: "frontier-git", Action: "plan.passed", Payload: map[string]any{"branch": "frontier/x", "head": "h1"}},
	})
	// Hand-tamper: rewrite the row with broken hashes.
	rows[0].EntryHash = "deadbeef"
	rows[0].PrevHash = "not-genesis"
	writeRows(t, path, rows)

	rep, err := AuditLedger(path)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if rep.Verdict != "tampered" {
		t.Fatalf("verdict = %s, want tampered", rep.Verdict)
	}
}

func TestAuditAllAndLedgersRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("FRONTIER_HOME", home)
	root := LedgersRoot()
	if root != filepath.Join(home, "ledgers") {
		t.Fatalf("LedgersRoot = %s, want %s", root, filepath.Join(home, "ledgers"))
	}
	// one clean ledger, one violating ledger
	now := time.Now()
	good := filepath.Join(root, "aaa", "ledger.jsonl")
	bad := filepath.Join(root, "bbb", "ledger.jsonl")
	if err := os.MkdirAll(filepath.Dir(good), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(bad), 0o755); err != nil {
		t.Fatal(err)
	}
	writeRows(t, good, chainOf([]ledger.Entry{
		{Seq: 1, TS: ts(now), Actor: "frontier-git", Action: "plan.passed", Payload: map[string]any{"branch": "frontier/x", "head": "h1"}},
		{Seq: 2, TS: ts(now.Add(time.Second)), Actor: "frontier-git", Action: "gate.passed", Payload: map[string]any{"branch": "frontier/x", "head": "h1"}},
		{Seq: 3, TS: ts(now.Add(2 * time.Second)), Actor: "frontier-git", Action: "push.authorized", Payload: map[string]any{"branch": "frontier/x", "head": "h1"}},
	}))
	writeRows(t, bad, chainOf([]ledger.Entry{
		{Seq: 1, TS: ts(now), Actor: "frontier-git", Action: "push.soft_allow", Payload: map[string]any{"branch": "frontier/x", "head": "h1"}},
	}))

	reps, err := AuditAll(root)
	if err != nil {
		t.Fatalf("audit all: %v", err)
	}
	if len(reps) != 2 {
		t.Fatalf("reports = %d, want 2", len(reps))
	}
	if got := AggregateVerdict(reps); got != "violation" {
		t.Errorf("aggregate verdict = %s, want violation", got)
	}
}
