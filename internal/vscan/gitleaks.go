package vscan

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// GitleaksScanner runs gitleaks when installed (secrets in the tree).
type GitleaksScanner struct{}

func (GitleaksScanner) Name() string  { return "gitleaks" }
func (GitleaksScanner) Builtin() bool { return false }
func (GitleaksScanner) Available() bool {
	_, err := exec.LookPath("gitleaks")
	return err == nil
}

func (g GitleaksScanner) Scan(root string) (Result, error) {
	if !g.Available() {
		return Result{Source: "gitleaks", Skipped: true, SkipWhy: "gitleaks not on PATH"}, nil
	}
	cmd := exec.Command("gitleaks", "detect", "--source", root, "--report-format", "json", "--no-banner", "--exit-code", "0")
	cmd.Dir = root
	out, err := cmd.Output()
	if len(out) == 0 && err != nil {
		return Result{Source: "gitleaks", Skipped: true, SkipWhy: fmt.Sprintf("gitleaks run failed: %v", err)}, nil
	}
	findings, meta, parseErr := ParseGitleaksJSON(out)
	if parseErr != nil {
		return Result{Source: "gitleaks", Skipped: true, SkipWhy: parseErr.Error()}, nil
	}
	findings = DropSkippedPaths(findings)
	if meta == nil {
		meta = map[string]any{}
	}
	meta["count"] = len(findings)
	return Result{Source: "gitleaks", Findings: findings, Meta: meta}, nil
}

type gitleaksHit struct {
	Description string `json:"Description"`
	File        string `json:"File"`
	StartLine   int    `json:"StartLine"`
	RuleID      string `json:"RuleID"`
	Severity    string `json:"Severity"`
}

// ParseGitleaksJSON maps gitleaks report JSON (array) to Findings.
func ParseGitleaksJSON(raw []byte) ([]Finding, map[string]any, error) {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" || s == "[]" {
		return nil, map[string]any{"count": 0}, nil
	}
	var hits []gitleaksHit
	if err := json.Unmarshal([]byte(s), &hits); err != nil {
		return nil, nil, fmt.Errorf("gitleaks json: %w", err)
	}
	out := make([]Finding, 0, len(hits))
	for _, h := range hits {
		sev := h.Severity
		if sev == "" {
			sev = "High"
		}
		out = append(out, Finding{
			Source:   "gitleaks",
			RuleID:   nz(h.RuleID, "gitleaks.unknown"),
			Severity: titleSev(sev),
			Path:     h.File,
			Line:     h.StartLine,
			Snippet:  trimSnippet(h.Description),
		})
	}
	return out, map[string]any{"count": len(out)}, nil
}

func titleSev(s string) string {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "CRITICAL":
		return "Critical"
	case "HIGH":
		return "High"
	case "MEDIUM":
		return "Medium"
	case "LOW":
		return "Low"
	default:
		return "High"
	}
}
