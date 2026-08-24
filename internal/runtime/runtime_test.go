package runtime

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReportTokensCapOff(t *testing.T) {
	t.Setenv("FRONTIER_RUNTIME_TOKEN_CAP", "")
	t.Setenv("FRONTIER_RUNTIME_TOKEN_PCT", "")
	al := &Allowlist{TokenPct: 5}
	r := ReportTokens(al, 0)
	if r.CapEnabled || r.ConfiguredPct != 5 || r.ConsumedPct != 0 || r.RemainingPct != 5 {
		t.Fatalf("%+v", r)
	}
}

func TestLoopbackHTTP(t *testing.T) {
	if !LoopbackHTTP("http://127.0.0.1:8765/health") || LoopbackHTTP("https://example.com/x") || LoopbackHTTP("http://0.0.0.0:80/") {
		t.Fatal("loopback filter")
	}
}

func TestValidateRejectsWAN(t *testing.T) {
	al := &Allowlist{Targets: []Target{{Name: "x", Kind: "http", URL: "https://evil.example/health"}}}
	if err := ValidateAllowlist(al); err == nil {
		t.Fatal("expected reject")
	}
}

func TestLoadAndScan(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	// httptest is 127.0.0.1
	dir := t.TempDir()
	p := filepath.Join(dir, "allow.json")
	raw, _ := json.Marshal(Allowlist{
		TokenPct: 5,
		Targets:  []Target{{Name: "h", Kind: "http", URL: srv.URL + "/health"}},
	})
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	al, err := LoadAllowlist(p)
	if err != nil {
		t.Fatal(err)
	}
	probes := ScanHTTP(al)
	if len(probes) != 1 || probes[0].Status != 200 {
		t.Fatalf("%+v", probes)
	}
	plan := PlanChaos(al, false)
	if plan.WouldInject || !strings.Contains(plan.Denied, "chaos.enabled") {
		t.Fatalf("%+v", plan)
	}
}

func TestChaosRequiresFlags(t *testing.T) {
	al := &Allowlist{
		Targets: []Target{{Name: "n", Kind: "docker-net", Net: "lab-net"}},
		Chaos:   ChaosCfg{Enabled: true, MaxDurationS: 30},
	}
	if err := ValidateAllowlist(al); err != nil {
		t.Fatal(err)
	}
	dry := PlanChaos(al, false)
	if dry.WouldInject {
		t.Fatal("dry must not inject")
	}
	goLive := PlanChaos(al, true)
	if !goLive.WouldInject || len(goLive.Steps) == 0 {
		t.Fatalf("%+v", goLive)
	}
}
