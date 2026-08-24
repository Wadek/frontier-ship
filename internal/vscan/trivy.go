package vscan

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// TrivyScanner runs trivy filesystem+config scan when installed.
type TrivyScanner struct{}

func (TrivyScanner) Name() string  { return "trivy" }
func (TrivyScanner) Builtin() bool { return false }
func (TrivyScanner) Available() bool {
	_, err := exec.LookPath("trivy")
	return err == nil
}

func (t TrivyScanner) Scan(root string) (Result, error) {
	if !t.Available() {
		return Result{Source: "trivy", Skipped: true, SkipWhy: "trivy not on PATH"}, nil
	}
	cmd := exec.Command("trivy", "fs", "--scanners", "vuln,misconfig,secret",
		"--skip-dirs", "testdata", "--format", "json", "--quiet", root)
	cmd.Dir = root
	out, err := cmd.Output()
	if len(out) == 0 && err != nil {
		return Result{Source: "trivy", Skipped: true, SkipWhy: fmt.Sprintf("trivy run failed: %v", err)}, nil
	}
	findings, meta, parseErr := ParseTrivyJSON(out)
	if parseErr != nil {
		return Result{Source: "trivy", Skipped: true, SkipWhy: parseErr.Error()}, nil
	}
	findings = DropSkippedPaths(findings)
	if meta == nil {
		meta = map[string]any{}
	}
	meta["count"] = len(findings)
	return Result{Source: "trivy", Findings: findings, Meta: meta}, nil
}

type trivyReport struct {
	Results []trivyResult `json:"Results"`
}

type trivyResult struct {
	Target            string           `json:"Target"`
	Vulnerabilities   []trivyVuln      `json:"Vulnerabilities"`
	Misconfigurations []trivyMisconfig `json:"Misconfigurations"`
	Secrets           []trivySecret    `json:"Secrets"`
}

type trivyVuln struct {
	VulnerabilityID string `json:"VulnerabilityID"`
	Title           string `json:"Title"`
	Severity        string `json:"Severity"`
	PkgName         string `json:"PkgName"`
}

type trivyMisconfig struct {
	ID       string `json:"ID"`
	Title    string `json:"Title"`
	Severity string `json:"Severity"`
}

type trivySecret struct {
	RuleID   string `json:"RuleID"`
	Title    string `json:"Title"`
	Severity string `json:"Severity"`
}

// ParseTrivyJSON maps `trivy fs --format json` to Findings.
func ParseTrivyJSON(raw []byte) ([]Finding, map[string]any, error) {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return nil, map[string]any{"count": 0}, nil
	}
	var rep trivyReport
	if err := json.Unmarshal([]byte(s), &rep); err != nil {
		return nil, nil, fmt.Errorf("trivy json: %w", err)
	}
	var out []Finding
	for _, r := range rep.Results {
		for _, v := range r.Vulnerabilities {
			out = append(out, Finding{
				Source:   "trivy",
				RuleID:   nz(v.VulnerabilityID, "trivy.vuln"),
				Severity: titleSev(v.Severity),
				Path:     r.Target,
				Snippet:  trimSnippet(v.PkgName + " " + v.Title),
			})
		}
		for _, m := range r.Misconfigurations {
			out = append(out, Finding{
				Source:   "trivy",
				RuleID:   nz(m.ID, "trivy.misconfig"),
				Severity: titleSev(m.Severity),
				Path:     r.Target,
				Snippet:  trimSnippet(m.Title),
			})
		}
		for _, sec := range r.Secrets {
			out = append(out, Finding{
				Source:   "trivy",
				RuleID:   nz(sec.RuleID, "trivy.secret"),
				Severity: titleSev(sec.Severity),
				Path:     r.Target,
				Snippet:  trimSnippet(sec.Title),
			})
		}
	}
	return out, map[string]any{"count": len(out)}, nil
}
