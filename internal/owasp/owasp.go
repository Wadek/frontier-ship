package owasp

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Finding is one match under current V (OWASP policy).
type Finding struct {
	RuleID   string `json:"rule_id"`
	OWASP    string `json:"owasp"`
	Severity string `json:"severity"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Snippet  string `json:"snippet"`
}

type rule struct {
	id, owasp, severity string
	re                  *regexp.Regexp
}

var rules = []rule{
	{"owasp.a01.hardcoded_auth_bypass", "A01", "High", regexp.MustCompile(`(?i)(if\s*\(.*(?:user|role|auth).*(?:==|!=)\s*["']admin["'])|bypass[_-]?auth|skip[_-]?auth`)},
	{"owasp.a02.secret_material", "A02", "Critical", regexp.MustCompile(`BEGIN (RSA |OPENSSH |EC )?PRIVATE KEY|AKIA[0-9A-Z]{16}`)},
	// Note: bare `.exec(` (e.g. better-sqlite3 db.exec DDL) is NOT a sink — require classic injection shapes.
	{"owasp.a03.injection_sink", "A03", "High", regexp.MustCompile(`(?i)(SELECT\s+.+\s*\+|os\.system\s*\(|subprocess\.[a-z]+\([^)]*shell\s*=\s*True|[^.\w]eval\s*\(|Runtime\.getRuntime\(\)\.exec\s*\(|child_process|shell\s*=\s*True)`)},
	{"owasp.a04.insecure_flag", "A04", "Medium", regexp.MustCompile(`(?i)(VERIFY_NONE|InsecureSkipVerify\s*[:=]\s*true|csrf\s*=\s*False|SSL_VERIFYPEER.*false)`)},
	{"owasp.a05.misconfig", "A05", "Medium", regexp.MustCompile(`(?i)(debug\s*=\s*True|APP_DEBUG\s*=\s*true|NODE_ENV\s*=\s*development.*=\s*true)`)},
	{"owasp.a06.unpinned_latest", "A06", "Medium", regexp.MustCompile(`(?i)^FROM\s+\S+:latest\b`)},
	{"owasp.a07.hardcoded_credential", "A07", "High", regexp.MustCompile(`(?i)(password\s*=\s*["'][^"']{3,}["']|api[_-]?key\s*=\s*["'][^"']{8,}["']|secret\s*=\s*["'][^"']{8,}["']|jwt[_-]?secret\s*=\s*["'])`)},
	{"owasp.a08.curl_bash", "A08", "High", regexp.MustCompile(`(?i)curl\s+[^\n|]*\|\s*(ba)?sh`)},
	// Avoid HTML autocomplete="…password…" false positives — require log/console context.
	{"owasp.a09.sensitive_log", "A09", "Medium", regexp.MustCompile(`(?i)((console\.(log|debug|info|warn)|logger\.[a-z]+|log\.(info|debug|warn|error))\([^;]*(password|passwd|authorization)|log\s*\([^;]*(password|passwd)\s*[:=])`)},
	{"owasp.a10.open_url_fetch", "A10", "Medium", regexp.MustCompile(`(?i)(requests\.(get|post)\(\s*(url|user)|urllib\.request\.urlopen\(\s*(url|user)|fetch\(\s*[a-z_]+\s*\))`)},
}

var skipDir = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, ".frontier": true,
	"__pycache__": true, "dist": true, "build": true,
	// Labeled fixtures are scored by go test, not by repo-wide gate (dogfood).
	"testdata": true,
	// Python virtualenvs are not the customer's code.
	"venv": true, ".venv": true, "site-packages": true,
}

// ScanTree walks root and returns findings under OWASP V.
func ScanTree(root string) ([]Finding, error) {
	var out []Finding
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if skipDir[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !scannable(path) {
			return nil
		}
		fs, err := scanFile(root, path)
		if err != nil {
			return nil
		}
		out = append(out, fs...)
		return nil
	})
	return out, err
}

func scannable(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	// Vendor / minified noise burns tokens and is not "our" vibe code.
	if strings.HasSuffix(base, ".min.js") || strings.HasSuffix(base, ".min.css") {
		return false
	}
	// Policy engine source embeds regexes that match themselves.
	if strings.Contains(filepath.ToSlash(path), "/internal/owasp/owasp.go") || base == "owasp.go" && strings.Contains(filepath.ToSlash(path), "/owasp/") {
		return false
	}
	// Unit tests embed intentional bad snippets as strings.
	if strings.HasSuffix(base, "_test.go") && strings.Contains(filepath.ToSlash(path), "/internal/owasp/") {
		return false
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go", ".py", ".js", ".ts", ".tsx", ".jsx", ".rb", ".php", ".java", ".cs",
		".sh", ".bash", ".ps1", ".yml", ".yaml", ".json", ".env", ".pem", ".key", ".dockerfile", "":
		if base == "dockerfile" || strings.HasPrefix(base, "dockerfile.") {
			return true
		}
		return ext != "" || base == "dockerfile"
	default:
		return base == "dockerfile"
	}
}

func scanFile(root, path string) ([]Finding, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rel, _ := filepath.Rel(root, path)
	var out []Finding
	sc := bufio.NewScanner(f)
	// allow long lines
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := sc.Text()
		for _, r := range rules {
			if r.re.MatchString(line) {
				snip := strings.TrimSpace(line)
				if len(snip) > 120 {
					snip = snip[:120] + "…"
				}
				out = append(out, Finding{
					RuleID: r.id, OWASP: r.owasp, Severity: r.severity,
					Path: rel, Line: lineNo, Snippet: snip,
				})
			}
		}
	}
	return out, nil
}

// BlocksGate reports whether findings include untriaged High/Critical.
func BlocksGate(findings []Finding) bool {
	for _, f := range findings {
		if f.Severity == "High" || f.Severity == "Critical" {
			return true
		}
	}
	return false
}

func FormatReport(findings []Finding) string {
	if len(findings) == 0 {
		return "owasp V: no matches (Clean under current definitions)"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "owasp V: %d finding(s)\n", len(findings))
	for _, f := range findings {
		fmt.Fprintf(&b, "  [%s/%s] %s:%d  %s\n    %s\n", f.Severity, f.OWASP, f.Path, f.Line, f.RuleID, f.Snippet)
	}
	return b.String()
}
