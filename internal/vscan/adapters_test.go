package vscan

import "testing"

func TestParseGitleaksJSON(t *testing.T) {
	raw := []byte(`[
	  {"Description":"AWS key","File":"app.env","StartLine":3,"RuleID":"aws-access-key","Severity":"HIGH"}
	]`)
	fs, meta, err := ParseGitleaksJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 1 || fs[0].RuleID != "aws-access-key" || fs[0].Severity != "High" {
		t.Fatalf("%+v meta=%v", fs, meta)
	}
}

func TestParseGitleaksEmpty(t *testing.T) {
	fs, _, err := ParseGitleaksJSON([]byte("[]"))
	if err != nil || len(fs) != 0 {
		t.Fatalf("%v %v", fs, err)
	}
}

func TestParseTrivyJSON(t *testing.T) {
	raw := []byte(`{
	  "Results": [{
	    "Target": "go.mod",
	    "Vulnerabilities": [{"VulnerabilityID":"CVE-1","Title":"x","Severity":"CRITICAL","PkgName":"foo"}],
	    "Misconfigurations": [{"ID":"DS-1","Title":"latest tag","Severity":"MEDIUM"}]
	  }]
	}`)
	fs, meta, err := ParseTrivyJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 2 {
		t.Fatalf("want 2 got %d meta=%v", len(fs), meta)
	}
}

func TestAdapterSkipTestdata(t *testing.T) {
	if !AdapterSkipPath("testdata/owasp/a06_pos/Dockerfile") {
		t.Fatal("want skip testdata")
	}
	in := []Finding{
		{Path: "testdata/owasp/a06_pos/Dockerfile", RuleID: "DS-0002"},
		{Path: "cmd/frontier/main.go", RuleID: "keep"},
	}
	got := DropSkippedPaths(in)
	if len(got) != 1 || got[0].RuleID != "keep" {
		t.Fatalf("%+v", got)
	}
}
