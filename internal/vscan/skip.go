package vscan

import (
	"path/filepath"
	"strings"
)

// adapterSkipDir matches Guard's labeled-fixture and junk dirs.
// testdata is scorecard input for built-in OWASP tests, not customer code.
var adapterSkipDir = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, ".frontier": true,
	"__pycache__": true, "dist": true, "build": true,
	"testdata": true,
	"venv":     true, ".venv": true, "site-packages": true,
}

// AdapterSkipPath is true when a finding lives under a skipped directory.
func AdapterSkipPath(p string) bool {
	p = filepath.ToSlash(p)
	p = strings.TrimPrefix(p, "./")
	for _, part := range strings.Split(p, "/") {
		if adapterSkipDir[strings.ToLower(part)] {
			return true
		}
	}
	return false
}

// DropSkippedPaths removes adapter hits under testdata and other skip dirs.
func DropSkippedPaths(in []Finding) []Finding {
	var out []Finding
	for _, f := range in {
		if AdapterSkipPath(f.Path) {
			continue
		}
		out = append(out, f)
	}
	return out
}
