// Package runtime is Frontier family R: post-ship probe and bounded chaos.
package runtime

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultAllowlist = `D:\frontier\runtime\allowlist.json`
	DefaultTokenPct  = 5
	HTTPTimeout      = 8 * time.Second
)

// Target is one allowlisted runtime endpoint.
type Target struct {
	Name string `json:"name"`
	Kind string `json:"kind"` // http | docker-net
	URL  string `json:"url,omitempty"`
	Net  string `json:"network,omitempty"`
}

// ChaosCfg bounds injects.
type ChaosCfg struct {
	Enabled      bool `json:"enabled"`
	MaxDurationS int  `json:"max_duration_s"`
}

// Allowlist is operator-owned, outside the work tree.
type Allowlist struct {
	TokenPct int      `json:"token_pct"`
	Targets  []Target `json:"targets"`
	Chaos    ChaosCfg `json:"chaos"`
	Path     string   `json:"-"`
}

// Probe is one HTTP scan result.
type Probe struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Status int    `json:"status"`
	Err    string `json:"error,omitempty"`
}

func AllowlistPath() string {
	if v := strings.TrimSpace(os.Getenv("FRONTIER_RUNTIME_ALLOWLIST")); v != "" {
		return v
	}
	return DefaultAllowlist
}

func TokenPct(al *Allowlist) int {
	if v := strings.TrimSpace(os.Getenv("FRONTIER_RUNTIME_TOKEN_PCT")); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n >= 0 && n <= 100 {
			return n
		}
	}
	if al != nil && al.TokenPct > 0 {
		return al.TokenPct
	}
	return DefaultTokenPct
}

// TokenCapEnabled is off unless FRONTIER_RUNTIME_TOKEN_CAP=1.
// The percentage logic stays; v1 only reports consumption.
func TokenCapEnabled() bool {
	v := os.Getenv("FRONTIER_RUNTIME_TOKEN_CAP")
	return v == "1" || strings.EqualFold(v, "true")
}

// TokenReport is a snapshot of intended share vs consumed share.
type TokenReport struct {
	CapEnabled    bool    `json:"cap_enabled"`
	ConfiguredPct int     `json:"configured_pct"`
	ConsumedPct   float64 `json:"consumed_pct"`
	RemainingPct  float64 `json:"remaining_pct"`
	Note          string  `json:"note"`
}

// ReportTokens builds the report. ConsumedPct is 0 until a boxed job records spend.
func ReportTokens(al *Allowlist, consumedPct float64) TokenReport {
	cfg := TokenPct(al)
	if consumedPct < 0 {
		consumedPct = 0
	}
	rem := float64(cfg) - consumedPct
	if rem < 0 {
		rem = 0
	}
	note := "reporting only — cap is off. A later runner records spend here."
	if TokenCapEnabled() {
		note = "cap ON — jobs must stay at or under configured_pct"
	}
	return TokenReport{
		CapEnabled:    TokenCapEnabled(),
		ConfiguredPct: cfg,
		ConsumedPct:   consumedPct,
		RemainingPct:  rem,
		Note:          note,
	}
}

func ChaosInjectFlag() bool {
	v := os.Getenv("FRONTIER_RUNTIME_CHAOS")
	return v == "1" || strings.EqualFold(v, "true")
}

// LoadAllowlist reads and validates the operator allowlist.
func LoadAllowlist(path string) (*Allowlist, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("runtime allowlist missing at %s (create it before scan/chaos)", path)
	}
	var al Allowlist
	if err := json.Unmarshal(b, &al); err != nil {
		return nil, fmt.Errorf("allowlist json: %w", err)
	}
	al.Path = path
	if err := ValidateAllowlist(&al); err != nil {
		return nil, err
	}
	return &al, nil
}

// ValidateAllowlist enforces v1 blast-radius rules.
func ValidateAllowlist(al *Allowlist) error {
	if al == nil || len(al.Targets) == 0 {
		return fmt.Errorf("allowlist has no targets")
	}
	if al.Chaos.MaxDurationS == 0 {
		al.Chaos.MaxDurationS = 60
	}
	if al.Chaos.MaxDurationS > 300 {
		return fmt.Errorf("chaos max_duration_s %d exceeds 300s cap", al.Chaos.MaxDurationS)
	}
	for i, t := range al.Targets {
		kind := strings.ToLower(strings.TrimSpace(t.Kind))
		switch kind {
		case "http":
			if !LoopbackHTTP(t.URL) {
				return fmt.Errorf("target %d %q: v1 http must be 127.0.0.1 or localhost, got %s", i, t.Name, t.URL)
			}
		case "docker-net":
			if strings.TrimSpace(t.Net) == "" {
				return fmt.Errorf("target %d %q: docker-net needs network name", i, t.Name)
			}
		default:
			return fmt.Errorf("target %d %q: unknown kind %q", i, t.Name, t.Kind)
		}
	}
	return nil
}

// LoopbackHTTP is the v1 HTTP allow rule.
func LoopbackHTTP(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}

// ScanHTTP GETs allowlisted http targets.
func ScanHTTP(al *Allowlist) []Probe {
	var out []Probe
	client := &http.Client{Timeout: HTTPTimeout}
	for _, t := range al.Targets {
		if strings.ToLower(t.Kind) != "http" {
			continue
		}
		p := Probe{Name: t.Name, URL: t.URL}
		resp, err := client.Get(t.URL)
		if err != nil {
			p.Err = err.Error()
			out = append(out, p)
			continue
		}
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64*1024))
		resp.Body.Close()
		p.Status = resp.StatusCode
		out = append(out, p)
	}
	return out
}

// ChaosPlan is a dry description of what inject would do.
type ChaosPlan struct {
	WouldInject bool     `json:"would_inject"`
	Denied      string   `json:"denied,omitempty"`
	Steps       []string `json:"steps"`
	DurationS   int      `json:"duration_s"`
}

// PlanChaos never mutates. Inject is a separate, flagged step.
func PlanChaos(al *Allowlist, injectFlag bool) ChaosPlan {
	plan := ChaosPlan{DurationS: al.Chaos.MaxDurationS}
	if !al.Chaos.Enabled {
		plan.Denied = "allowlist chaos.enabled is false"
		return plan
	}
	if !injectFlag {
		plan.Denied = "dry-run (set FRONTIER_RUNTIME_CHAOS=1 to inject)"
		plan.Steps = chaosSteps(al)
		return plan
	}
	plan.WouldInject = true
	plan.Steps = chaosSteps(al)
	return plan
}

func chaosSteps(al *Allowlist) []string {
	var s []string
	for _, t := range al.Targets {
		switch strings.ToLower(t.Kind) {
		case "http":
			s = append(s, "briefly refuse connections on "+t.URL+" for <= "+itoa(al.Chaos.MaxDurationS)+"s (not implemented in v1 inject)")
		case "docker-net":
			s = append(s, "pause/isolate docker network "+t.Net+" for <= "+itoa(al.Chaos.MaxDurationS)+"s (not implemented in v1 inject)")
		}
	}
	if len(s) == 0 {
		s = append(s, "(no chaos steps)")
	}
	return s
}

func itoa(n int) string { return strconv.Itoa(n) }
