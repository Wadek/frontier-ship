package vscan

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EnhanceArtifacts are paths written by enhance V.
type EnhanceArtifacts struct {
	Dir      string
	Markdown string
	JSON     string
	Latest   string
	Stamp    string
}

// WriteEnhanceBrief builds a token-thrifty handoff for the host model.
func WriteEnhanceBrief(root string, p *Pack) (*EnhanceArtifacts, error) {
	dir := filepath.Join(root, ".frontier", "enhance")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	base := "V-" + stamp
	mdPath := filepath.Join(dir, base+".md")
	jsPath := filepath.Join(dir, base+".json")
	latest := filepath.Join(dir, "LATEST")

	md := renderBriefMarkdown(p, base)
	if len(md) > MaxBriefBytes {
		md = md[:MaxBriefBytes] + "\n\n...[truncated for token budget]\n"
	}
	if err := os.WriteFile(mdPath, []byte(md), 0o644); err != nil {
		return nil, err
	}
	js, _ := json.MarshalIndent(struct {
		Stamp        string            `json:"stamp"`
		Disposition  string            `json:"disposition"`
		BySource     map[string]int    `json:"by_source"`
		FindingsN    int               `json:"findings_n"`
		Findings     []Finding         `json:"findings_capped"`
		ScopeMode    string            `json:"scope_mode"`
		Paths        []string          `json:"changed_paths"`
		Manifests    []string          `json:"manifests"`
		LangCounts   map[string]int    `json:"lang_counts"`
		AdaptersRun  []string          `json:"adapters_run"`
		AdaptersSkip map[string]string `json:"adapters_skip"`
		Budget       map[string]int    `json:"token_budget"`
	}{
		Stamp: stamp, Disposition: p.Disposition, BySource: p.BySource,
		FindingsN: len(p.Findings), Findings: p.FindingsCapped,
		ScopeMode: p.ScopeMode, Paths: p.ChangedPaths, Manifests: p.Manifests,
		LangCounts: p.LangCounts, AdaptersRun: p.AdaptersRun, AdaptersSkip: p.AdaptersSkip,
		Budget: map[string]int{
			"max_findings": MaxBriefFindings,
			"max_paths":    MaxBriefPaths,
			"max_bytes":    MaxBriefBytes,
		},
	}, "", "  ")
	if err := os.WriteFile(jsPath, js, 0o644); err != nil {
		return nil, err
	}
	_ = os.WriteFile(latest, []byte(base+"\n"), 0o644)
	return &EnhanceArtifacts{Dir: dir, Markdown: mdPath, JSON: jsPath, Latest: latest, Stamp: stamp}, nil
}

func renderBriefMarkdown(p *Pack, base string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Frontier enhance V - host handoff (%s)\n\n", base)
	b.WriteString("## Mission\n\n")
	b.WriteString("Do **residual** SAST/DAST/pentest only. Do **not** re-litigate programmatic hits below - ")
	b.WriteString("they were decided without tokens. Prefer promoting durable rules into V later over one-off chat findings.\n\n")

	b.WriteString("## Already decided (no tokens)\n\n")
	fmt.Fprintf(&b, "- disposition (programmatic): **%s**\n", p.Disposition)
	fmt.Fprintf(&b, "- findings total: **%d** (showing <=%d)\n", len(p.Findings), MaxBriefFindings)
	fmt.Fprintf(&b, "- adapters run: %s\n", strings.Join(p.AdaptersRun, ", "))
	if len(p.AdaptersSkip) > 0 {
		b.WriteString("- adapters skipped:\n")
		for k, v := range p.AdaptersSkip {
			fmt.Fprintf(&b, "  - %s: %s\n", k, v)
		}
	}
	if len(p.BySource) > 0 {
		b.WriteString("- by source:")
		for k, v := range p.BySource {
			fmt.Fprintf(&b, " %s=%d", k, v)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	if len(p.FindingsCapped) == 0 {
		b.WriteString("_No programmatic findings under current V + adapters._\n\n")
	} else {
		for _, f := range p.FindingsCapped {
			fmt.Fprintf(&b, "- [%s/%s] %s %s:%d  `%s`\n", f.Severity, f.Source, f.RuleID, f.Path, f.Line, trimSnippet(f.Snippet))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Scope\n\n")
	fmt.Fprintf(&b, "- mode: `%s`\n", p.ScopeMode)
	if len(p.ChangedPaths) == 0 {
		b.WriteString("- paths: _(full tree - no merge-base diff)_\n\n")
	} else {
		fmt.Fprintf(&b, "- changed paths (capped %d):\n", MaxBriefPaths)
		for _, path := range p.ChangedPaths {
			fmt.Fprintf(&b, "  - %s\n", path)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Inventory\n\n")
	if len(p.LangCounts) == 0 {
		b.WriteString("- langs: _(none counted)_\n")
	} else {
		b.WriteString("- langs:")
		for k, v := range p.LangCounts {
			fmt.Fprintf(&b, " %s=%d", k, v)
		}
		b.WriteString("\n")
	}
	if len(p.Manifests) > 0 {
		b.WriteString("- manifests:\n")
		for _, m := range p.Manifests {
			fmt.Fprintf(&b, "  - %s\n", m)
		}
	}
	b.WriteString("\n")

	b.WriteString("## Gaps for host model (residual only)\n\n")
	for _, g := range residualGaps(p) {
		fmt.Fprintf(&b, "- %s\n", g)
	}
	b.WriteString("\n")

	b.WriteString("## Seal\n\n")
	b.WriteString("Write JSON `{summary, findings[], disposition_suggest, tools_used[], residual_risk}` then:\n\n")
	b.WriteString("```text\nfrontier enhance seal .frontier/enhance/<your-result>.json\n```\n\n")
	b.WriteString("Default disposition for enhance results: **advise** (does not block gate until promoted into V).\n")
	return b.String()
}

func residualGaps(p *Pack) []string {
	ran := map[string]bool{}
	for _, a := range p.AdaptersRun {
		ran[a] = true
	}
	gaps := []string{
		"Authz / business-logic flaws not expressible as regex (review sensitive routes in scope).",
		"DAST: only if an app is runnable - probe auth, IDOR, injection on live endpoints (control_point=runtime).",
		"Pentest confirm: do not claim exploit without evidence (control_point=engagement).",
	}
	if !ran["checkov"] {
		gaps = append(gaps, "IaC: Checkov not run - review Terraform/K8s/Dockerfile manually or install checkov.")
	}
	if !ran["gitleaks"] {
		gaps = append(gaps, "Secrets beyond PEM/AKIA patterns: install gitleaks or run frontier guard gitleaks.")
	}
	if !ran["trivy"] {
		gaps = append(gaps, "Dependency/config CVEs: install trivy or run frontier guard trivy.")
	}
	if !ran["semgrep"] {
		gaps = append(gaps, "Broader SAST dataflow: semgrep/CodeQL outside Frontier until adapter exists.")
	}
	hasLock := false
	for _, m := range p.Manifests {
		base := strings.ToLower(filepath.Base(m))
		if base == "go.sum" || base == "package-lock.json" || base == "yarn.lock" || base == "poetry.lock" {
			hasLock = true
		}
	}
	if !hasLock {
		gaps = append(gaps, "Dependency CVEs: no lockfile spotted in inventory - check manifests manually.")
	} else {
		gaps = append(gaps, "Dependency CVEs: lockfile present but no CVE scanner in programmatic V yet - optional host pass.")
	}
	return gaps
}

func trimSnippet(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 80 {
		return s[:80] + "..."
	}
	return s
}

// SealPayload is what the host agent writes for enhance seal.
type SealPayload struct {
	Summary            string    `json:"summary"`
	Findings           []Finding `json:"findings"`
	DispositionSuggest string    `json:"disposition_suggest"`
	ToolsUsed          []string  `json:"tools_used"`
	ResidualRisk       string    `json:"residual_risk"`
}

// ReadSealFile loads an enhance seal JSON.
func ReadSealFile(path string) (*SealPayload, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p SealPayload
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	if p.DispositionSuggest == "" {
		p.DispositionSuggest = "advise"
	}
	return &p, nil
}
