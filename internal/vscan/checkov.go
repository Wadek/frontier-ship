package vscan

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// CheckovScanner runs Checkov when installed (IaC misconfig - programmatic, no tokens).
type CheckovScanner struct{}

func (CheckovScanner) Name() string  { return "checkov" }
func (CheckovScanner) Builtin() bool { return false }
func (CheckovScanner) Available() bool {
	_, err := exec.LookPath("checkov")
	return err == nil
}

func (c CheckovScanner) Scan(root string) (Result, error) {
	if !c.Available() {
		return Result{
			Source:  "checkov",
			Skipped: true,
			SkipWhy: "checkov not on PATH (pip install checkov)",
		}, nil
	}
	// Soft-fail: Checkov exits non-zero when findings exist.
	cmd := exec.Command("checkov", "-d", root, "-o", "json", "--quiet", "--compact")
	cmd.Dir = root
	out, err := cmd.Output()
	if len(out) == 0 && err != nil {
		return Result{Source: "checkov", Skipped: true, SkipWhy: fmt.Sprintf("checkov run failed: %v", err)}, nil
	}
	findings, meta, parseErr := ParseCheckovJSON(out)
	if parseErr != nil {
		return Result{Source: "checkov", Skipped: true, SkipWhy: parseErr.Error()}, nil
	}
	findings = DropSkippedPaths(findings)
	if meta == nil {
		meta = map[string]any{}
	}
	meta["count"] = len(findings)
	return Result{Source: "checkov", Findings: findings, Meta: meta}, nil
}

// ParseCheckovJSON maps Checkov JSON (single object or array of reports) to Findings.
func ParseCheckovJSON(raw []byte) ([]Finding, map[string]any, error) {
	raw = bytesTrim(raw)
	if len(raw) == 0 {
		return nil, map[string]any{"count": 0}, nil
	}
	var reports []checkovReport
	if raw[0] == '[' {
		if err := json.Unmarshal(raw, &reports); err != nil {
			return nil, nil, fmt.Errorf("checkov json array: %w", err)
		}
	} else {
		var one checkovReport
		if err := json.Unmarshal(raw, &one); err != nil {
			return nil, nil, fmt.Errorf("checkov json: %w", err)
		}
		reports = []checkovReport{one}
	}
	var out []Finding
	passed, failed := 0, 0
	for _, rep := range reports {
		passed += rep.Summary.Passed
		failed += rep.Summary.Failed
		for _, r := range rep.Results.FailedChecks {
			sev := mapCheckovSeverity(r.Severity)
			path := r.FilePath
			if path == "" {
				path = r.RepoFilePath
			}
			snip := r.CheckName
			if len(snip) > 120 {
				snip = snip[:120] + "..."
			}
			out = append(out, Finding{
				Source:   "checkov",
				RuleID:   nz(r.CheckID, "checkov.unknown"),
				Severity: sev,
				Path:     path,
				Line:     firstLine(r.FileLineRange),
				Snippet:  snip,
			})
		}
	}
	return out, map[string]any{"passed": passed, "failed": failed, "count": len(out)}, nil
}

type checkovReport struct {
	Summary struct {
		Passed int `json:"passed"`
		Failed int `json:"failed"`
	} `json:"summary"`
	Results struct {
		FailedChecks []checkovFailed `json:"failed_checks"`
	} `json:"results"`
}

type checkovFailed struct {
	CheckID       string `json:"check_id"`
	CheckName     string `json:"check_name"`
	FilePath      string `json:"file_path"`
	RepoFilePath  string `json:"repo_file_path"`
	FileLineRange []int  `json:"file_line_range"`
	Severity      string `json:"severity"`
}

func mapCheckovSeverity(s string) string {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "CRITICAL":
		return "Critical"
	case "HIGH":
		return "High"
	case "LOW":
		return "Low"
	case "INFO", "INFORMATIONAL":
		return "Info"
	default:
		// Checkov often omits severity; treat failed IaC as Medium (advise, not auto-block gate).
		return "Medium"
	}
}

func firstLine(rng []int) int {
	if len(rng) > 0 {
		return rng[0]
	}
	return 0
}

func nz(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}

func bytesTrim(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}
