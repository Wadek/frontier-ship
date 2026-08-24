package vscan

import "strings"

// Registry returns built-in + adapter scanners (stubs included for list UX).
func Registry() []Scanner {
	return []Scanner{
		OWASPScanner{},
		CheckovScanner{},
		GitleaksScanner{},
		TrivyScanner{},
		stubScanner{name: "semgrep", why: "planned adapter - broad SAST (use when wired)"},
	}
}

func Lookup(name string) (Scanner, bool) {
	n := strings.ToLower(strings.TrimSpace(name))
	for _, s := range Registry() {
		if s.Name() == n {
			return s, true
		}
	}
	return nil, false
}

type stubScanner struct {
	name, why string
}

func (s stubScanner) Name() string    { return s.name }
func (s stubScanner) Available() bool { return false }
func (s stubScanner) Builtin() bool   { return false }
func (s stubScanner) Scan(root string) (Result, error) {
	return Result{Source: s.name, Skipped: true, SkipWhy: s.why}, nil
}
