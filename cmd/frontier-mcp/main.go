package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Wadek/frontier-ship/internal/egress"
	"github.com/Wadek/frontier-ship/internal/gitx"
	"github.com/Wadek/frontier-ship/internal/hygiene"
	"github.com/Wadek/frontier-ship/internal/ledger"
	"github.com/Wadek/frontier-ship/internal/mcpstdio"
	"github.com/Wadek/frontier-ship/internal/policy"
	"github.com/Wadek/frontier-ship/internal/role"
)

// Set by SLSA / release ldflags.
var (
	version = "dev"
	commit  = "none"
)

type state struct {
	mu   sync.Mutex
	role role.Role
	repo gitx.Repo
	led  *ledger.Ledger
}

func main() {
	repoDir := env("FRONTIER_REPO", mustAbs("."))
	ledPath := env("FRONTIER_LEDGER", filepath.Join(repoDir, ".frontier", "ledger.jsonl"))
	startRole := env("FRONTIER_ROLE", "observer")

	r, err := role.Parse(startRole)
	if err != nil {
		fatal(err)
	}
	led, err := ledger.Open(ledPath)
	if err != nil {
		fatal(err)
	}
	st := &state{role: r, repo: gitx.Repo{Dir: repoDir}, led: led}
	_, _ = led.Append("system", "session.start", map[string]any{
		"repo": repoDir,
		"role": r.String(),
	})

	emptyObj := map[string]any{"type": "object", "properties": map[string]any{}, "additionalProperties": false}

	srv := &mcpstdio.Server{
		Name:    "frontier-ship",
		Version: version,
		Tools: []mcpstdio.Tool{
			{Name: "frontier_whoami", Description: "Show current role, repo, and frontier principles. Cheap. Start here.", InputSchema: emptyObj},
			{Name: "frontier_observe", Description: "Observer: git status, branch, recent log. No writes.", InputSchema: emptyObj},
			{Name: "frontier_analyze", Description: "Analyst: diffstat + short advice (egress-safe summary). No writes.", InputSchema: emptyObj},
			{Name: "frontier_hygiene", Description: "Analyst: inspect changeset for AI provenance marks (local watermarks-remover). Advise-only.", InputSchema: emptyObj},
			{Name: "frontier_elevate", Description: "Elevate exactly one role rung (observer→analyst→operator→executor). Logged.", InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"reason": map[string]any{"type": "string", "description": "why elevation is needed"},
				},
				"required": []string{"reason"},
			}},
			{Name: "frontier_prepare", Description: "Operator: create/reset branch, git add -A, commit. No push.", InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"branch":  map[string]any{"type": "string", "description": "feature branch name"},
					"message": map[string]any{"type": "string", "description": "commit message"},
				},
				"required": []string{"branch", "message"},
			}},
			{Name: "frontier_gate", Description: "Conscience check before push. Seals gate.passed/failed in ledger. Local, no model.", InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"allow_dirty": map[string]any{"type": "boolean", "description": "allow dirty tree (default false)"},
				},
			}},
			{Name: "frontier_push", Description: "Executor: git push only if a fresh gate.passed matches HEAD. No bypass.", InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"remote": map[string]any{"type": "string", "description": "default origin"},
				},
			}},
			{Name: "frontier_ledger", Description: "Observer: show last N ledger events.", InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"n": map[string]any{"type": "integer", "description": "default 10"},
				},
			}},
		},
		Call: st.dispatch,
	}

	if err := srv.Run(); err != nil {
		fatal(err)
	}
}

func (st *state) dispatch(name string, args map[string]any) (string, error) {
	st.mu.Lock()
	defer st.mu.Unlock()

	switch name {
	case "frontier_whoami":
		return st.whoami()
	case "frontier_observe":
		if err := policy.Require(st.role, role.Observer, name); err != nil {
			return "", err
		}
		return st.observe()
	case "frontier_analyze":
		if err := policy.Require(st.role, role.Analyst, name); err != nil {
			return "", err
		}
		return st.analyze()
	case "frontier_hygiene":
		if err := policy.Require(st.role, role.Analyst, name); err != nil {
			return "", err
		}
		return st.hygieneInspect()
	case "frontier_elevate":
		reason, _ := args["reason"].(string)
		if strings.TrimSpace(reason) == "" {
			return "", fmt.Errorf("reason required")
		}
		return st.elevate(reason)
	case "frontier_prepare":
		if err := policy.Require(st.role, role.Operator, name); err != nil {
			return "", err
		}
		branch, _ := args["branch"].(string)
		msg, _ := args["message"].(string)
		return st.prepare(branch, msg)
	case "frontier_gate":
		if err := policy.Require(st.role, role.Operator, name); err != nil {
			return "", err
		}
		allowDirty, _ := args["allow_dirty"].(bool)
		return st.gate(allowDirty)
	case "frontier_push":
		if err := policy.Require(st.role, role.Executor, name); err != nil {
			return "", err
		}
		remote, _ := args["remote"].(string)
		return st.push(remote)
	case "frontier_ledger":
		if err := policy.Require(st.role, role.Observer, name); err != nil {
			return "", err
		}
		n := 10
		if v, ok := args["n"].(float64); ok {
			n = int(v)
		}
		return st.ledgerTail(n)
	default:
		return "", fmt.Errorf("unknown tool %s", name)
	}
}

func (st *state) whoami() (string, error) {
	out := fmt.Sprintf(`frontier-ship MCP
role:    %s
repo:    %s
ledger:  local append-only

principles:
  - local first
  - simple is better
  - simple models do simple things (one tool = one cheap step)
  - evidence before remote mutate
  - no bypass: push requires fresh gate.passed

ladder: observer -> analyst -> operator -> executor
`, st.role, st.repo.Dir)
	_, _ = st.led.Append(st.role.String(), "whoami", map[string]any{"role": st.role.String()})
	return out, nil
}

func (st *state) observe() (string, error) {
	branch, _ := st.repo.Branch()
	status, err := st.repo.StatusPorcelain()
	if err != nil {
		return "", err
	}
	log, _ := st.repo.Log(5)
	_, _ = st.led.Append(st.role.String(), "observe", map[string]any{
		"branch": branch,
		"dirty":  strings.TrimSpace(status) != "",
	})
	return fmt.Sprintf("branch: %s\ndirty: %v\n\nstatus:\n%s\n\nrecent:\n%s\n",
		branch, strings.TrimSpace(status) != "", status, log), nil
}

func (st *state) hygieneInspect() (string, error) {
	rels := st.repo.ChangedPaths()
	targets := hygiene.ResolveTargets(st.repo.Dir, rels)
	rep := hygiene.InspectFiles(hygiene.ServiceURL(), st.repo.Dir, targets)
	_, _ = st.led.Append(st.role.String(), "hygiene.inspected", map[string]any{
		"healthy":     rep.Healthy,
		"scanned":     rep.Scanned,
		"suspicious":  rep.Suspicious,
		"disposition": hygiene.Disposition(rep),
	})
	return hygiene.FormatReport(rep), nil
}

func (st *state) analyze() (string, error) {
	stat, err := st.repo.DiffStat()
	if err != nil {
		return "", err
	}
	sum := egress.SummarizeDiff(stat)
	advice := egress.AdviceFromStat(sum)
	_, _ = st.led.Append(st.role.String(), "analyze", map[string]any{
		"summary": sum,
		"advice":  advice,
	})
	return fmt.Sprintf("egress-safe summary:\n%s\n\nadvice: %s\n", sum, advice), nil
}

func (st *state) elevate(reason string) (string, error) {
	next, err := st.role.Elevate()
	if err != nil {
		return "", err
	}
	prev := st.role
	st.role = next
	_, _ = st.led.Append(prev.String(), "elevate", map[string]any{
		"from":   prev.String(),
		"to":     next.String(),
		"reason": reason,
	})
	return fmt.Sprintf("elevated %s -> %s\nreason: %s\n", prev, next, reason), nil
}

func (st *state) prepare(branch, message string) (string, error) {
	branch = strings.TrimSpace(branch)
	message = strings.TrimSpace(message)
	if branch == "" || message == "" {
		return "", fmt.Errorf("branch and message required")
	}
	if strings.EqualFold(branch, "main") || strings.EqualFold(branch, "master") {
		return "", fmt.Errorf("deny: refuse commits directly on main/master via this tool")
	}
	if err := st.repo.CheckoutNew(branch); err != nil {
		return "", err
	}
	if err := st.repo.AddAll(); err != nil {
		return "", err
	}
	if err := st.repo.Commit(message); err != nil {
		return "", err
	}
	head, _ := st.repo.RevParseHead()
	_, _ = st.led.Append(st.role.String(), "prepare.commit", map[string]any{
		"branch":  branch,
		"message": message,
		"head":    head,
	})
	return fmt.Sprintf("committed on %s\nHEAD %s\nnext: frontier_gate then elevate to executor and frontier_push\n", branch, head), nil
}

func (st *state) gate(allowDirty bool) (string, error) {
	branch, _ := st.repo.Branch()
	head, _ := st.repo.RevParseHead()
	por, _ := st.repo.StatusPorcelain()
	g := policy.EvaluatePushGate(branch, head, por, allowDirty)
	sealed, err := policy.SealGate(st.led, st.role.String(), g)
	if err != nil {
		return "", err
	}
	b, _ := json.MarshalIndent(sealed, "", "  ")
	return string(b) + "\n", nil
}

func (st *state) push(remote string) (string, error) {
	branch, _ := st.repo.Branch()
	head, _ := st.repo.RevParseHead()
	ok, detail := policy.FreshGateOK(st.led, branch, head)
	if !ok {
		_, _ = st.led.Append(st.role.String(), "push.denied", map[string]any{"reason": detail})
		return "", fmt.Errorf("deny: %s", detail)
	}
	if err := st.repo.Push(remote, branch); err != nil {
		_, _ = st.led.Append(st.role.String(), "push.failed", map[string]any{"error": err.Error()})
		return "", err
	}
	_, _ = st.led.Append(st.role.String(), "push.ok", map[string]any{
		"branch":    branch,
		"head":      head,
		"remote":    emptyDefault(remote, "origin"),
		"gate_seal": detail,
	})
	return fmt.Sprintf("pushed %s to %s\ngate_seal: %s\n", branch, emptyDefault(remote, "origin"), detail), nil
}

func (st *state) ledgerTail(n int) (string, error) {
	entries, err := st.led.Tail(n)
	if err != nil {
		return "", err
	}
	b, _ := json.MarshalIndent(entries, "", "  ")
	return string(b) + "\n", nil
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func mustAbs(p string) string {
	a, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return a
}

func emptyDefault(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "frontier-mcp:", err)
	os.Exit(1)
}
