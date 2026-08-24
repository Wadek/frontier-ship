// Package hygiene is Frontier family H: AI-provenance hygiene via watermarks-remover.
// Advise-only at the changeset. Cleaning is an explicit Operator step.
package hygiene

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultURL    = "http://127.0.0.1:8765"
	MaxFiles      = 30
	MaxFileBytes  = 1_500_000
	HTTPTimeout   = 12 * time.Second
	HealthTimeout = 2 * time.Second
)

// Finding is one inspected file.
type Finding struct {
	Path       string `json:"path"`
	Kind       string `json:"kind"`
	Suspicious bool   `json:"suspicious"`
	Error      string `json:"error,omitempty"`
	Report     string `json:"report,omitempty"`
}

// Report is one Hygiene pass.
type Report struct {
	Root       string    `json:"root"`
	Service    string    `json:"service"`
	Healthy    bool      `json:"healthy"`
	Version    string    `json:"version,omitempty"`
	Note       string    `json:"note,omitempty"`
	Scanned    int       `json:"scanned"`
	Suspicious int       `json:"suspicious"`
	Findings   []Finding `json:"findings"`
	Caps       *Caps     `json:"capabilities,omitempty"`
}

// Caps is GET /capabilities (subset).
type Caps struct {
	OK      bool           `json:"ok"`
	Version string         `json:"version"`
	Tools   map[string]any `json:"tools,omitempty"`
}

// ServiceURL reads WATERMARKS_SERVICE_URL then FRONTIER_HYGIENE_URL.
func ServiceURL() string {
	for _, k := range []string{"WATERMARKS_SERVICE_URL", "FRONTIER_HYGIENE_URL"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return strings.TrimRight(v, "/")
		}
	}
	return DefaultURL
}

func client() *http.Client {
	return &http.Client{Timeout: HTTPTimeout}
}

func healthClient() *http.Client {
	return &http.Client{Timeout: HealthTimeout}
}

// Health hits GET /health.
func Health(base string) (version string, err error) {
	resp, err := healthClient().Get(base + "/health")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("health HTTP %d", resp.StatusCode)
	}
	var payload struct {
		OK      bool   `json:"ok"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	if !payload.OK {
		return "", fmt.Errorf("health not ok")
	}
	return payload.Version, nil
}

// Capabilities hits GET /capabilities.
func Capabilities(base string) (*Caps, error) {
	resp, err := client().Get(base + "/capabilities")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	var c Caps
	if err := json.Unmarshal(body, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

type inspectResp struct {
	OK         bool            `json:"ok"`
	Kind       string          `json:"kind"`
	Suspicious bool            `json:"suspicious"`
	Error      string          `json:"error"`
	Report     json.RawMessage `json:"report"`
}

type cleanResp struct {
	OK      bool            `json:"ok"`
	Kind    string          `json:"kind"`
	Cleaned string          `json:"cleaned"`
	Error   string          `json:"error"`
	Report  json.RawMessage `json:"report"`
}

func postJSON(base, path string, payload any, dest any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := client().Post(base+path, "application/json", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s HTTP %d: %s", path, resp.StatusCode, truncate(string(body), 240))
	}
	return json.Unmarshal(body, dest)
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func reportString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return truncate(s, 400)
	}
	return truncate(string(raw), 400)
}

// InspectFile POSTs /inspect for one file.
func InspectFile(base, absPath, displayName string) Finding {
	f := Finding{Path: displayName}
	info, err := os.Stat(absPath)
	if err != nil {
		f.Error = err.Error()
		return f
	}
	if info.Size() > MaxFileBytes {
		f.Error = fmt.Sprintf("skip: %d bytes > %d cap", info.Size(), MaxFileBytes)
		return f
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		f.Error = err.Error()
		return f
	}
	var out inspectResp
	err = postJSON(base, "/inspect", map[string]any{
		"file": base64.StdEncoding.EncodeToString(data),
		"name": filepath.Base(displayName),
	}, &out)
	if err != nil {
		f.Error = err.Error()
		return f
	}
	if out.Error != "" && !out.OK {
		f.Error = out.Error
	}
	f.Kind = out.Kind
	f.Suspicious = out.Suspicious
	f.Report = reportString(out.Report)
	return f
}

// CleanFile POSTs /clean. inPlace overwrites; otherwise writes path.cleaned.ext.
func CleanFile(base, absPath, displayName string, inPlace bool) (outPath string, f Finding, err error) {
	f = Finding{Path: displayName}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", f, err
	}
	if int64(len(data)) > MaxFileBytes {
		return "", f, fmt.Errorf("file too large (%d)", len(data))
	}
	var out cleanResp
	err = postJSON(base, "/clean", map[string]any{
		"file": base64.StdEncoding.EncodeToString(data),
		"name": filepath.Base(displayName),
	}, &out)
	if err != nil {
		return "", f, err
	}
	if !out.OK {
		msg := out.Error
		if msg == "" {
			msg = "clean not ok"
		}
		return "", f, fmt.Errorf("%s", msg)
	}
	cleaned, err := base64.StdEncoding.DecodeString(out.Cleaned)
	if err != nil {
		return "", f, fmt.Errorf("decode cleaned: %w", err)
	}
	f.Kind = out.Kind
	f.Report = reportString(out.Report)
	outPath = absPath
	if !inPlace {
		ext := filepath.Ext(absPath)
		stem := strings.TrimSuffix(absPath, ext)
		outPath = stem + ".cleaned" + ext
	}
	if err := os.WriteFile(outPath, cleaned, 0o644); err != nil {
		return "", f, err
	}
	return outPath, f, nil
}

// HygieneExt reports whether the path is a watermarks-remover format we bother with.
func HygieneExt(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".txt", ".html", ".htm", ".csv", ".json", ".xml",
		".png", ".jpg", ".jpeg", ".webp", ".avif", ".heic", ".bmp", ".gif", ".tiff", ".tif",
		".svg", ".pdf", ".docx", ".xlsx", ".pptx", ".epub", ".odt",
		".py", ".go", ".js", ".ts", ".tsx", ".jsx", ".rs", ".rb", ".java",
		".css", ".yml", ".yaml", ".toml", ".rst", ".adoc":
		return true
	default:
		return false
	}
}

// ResolveTargets returns absolute paths under root for the given relative or
// absolute names that exist and match HygieneExt.
func ResolveTargets(root string, rels []string) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, p := range rels {
		abs := p
		if !filepath.IsAbs(p) {
			abs = filepath.Join(root, p)
		}
		st, err := os.Stat(abs)
		if err != nil || st.IsDir() {
			continue
		}
		if !HygieneExt(abs) {
			continue
		}
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		out = append(out, abs)
	}
	return capSlice(out, MaxFiles)
}

func capSlice(in []string, n int) []string {
	if len(in) > n {
		return in[:n]
	}
	return in
}

// InspectFiles runs /inspect on each absolute path.
func InspectFiles(base, root string, absPaths []string) *Report {
	rep := &Report{Root: root, Service: base, Findings: nil}
	ver, err := Health(base)
	if err != nil {
		rep.Healthy = false
		rep.Note = "watermarks-remover not reachable at " + base + " (" + err.Error() + "). Start: python D:\\wakalabs\\watermarks-remover\\service\\scripts\\server.py --host 127.0.0.1 --port 8765"
		return rep
	}
	rep.Healthy = true
	rep.Version = ver
	if caps, err := Capabilities(base); err == nil {
		rep.Caps = caps
	}
	for _, abs := range absPaths {
		rel, _ := filepath.Rel(root, abs)
		if rel == "" || strings.HasPrefix(rel, "..") {
			rel = abs
		}
		f := InspectFile(base, abs, filepath.ToSlash(rel))
		rep.Findings = append(rep.Findings, f)
		rep.Scanned++
		if f.Suspicious {
			rep.Suspicious++
		}
	}
	return rep
}

// FormatReport is the terminal view (verbose, named results).
func FormatReport(r *Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "service:     %s\n", r.Service)
	fmt.Fprintf(&b, "healthy:     %v", r.Healthy)
	if r.Version != "" {
		fmt.Fprintf(&b, "  version=%s", r.Version)
	}
	b.WriteByte('\n')
	if r.Note != "" {
		fmt.Fprintf(&b, "note:        %s\n", r.Note)
	}
	fmt.Fprintf(&b, "scanned:     %d\n", r.Scanned)
	fmt.Fprintf(&b, "suspicious:  %d\n", r.Suspicious)
	fmt.Fprintf(&b, "disposition: %s\n", Disposition(r))
	for _, f := range r.Findings {
		flag := "clean"
		if f.Error != "" {
			flag = "error"
		} else if f.Suspicious {
			flag = "MARKS"
		}
		fmt.Fprintf(&b, "  [%s] %s  kind=%s", flag, f.Path, f.Kind)
		if f.Error != "" {
			fmt.Fprintf(&b, "  %s", f.Error)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// Disposition is advise when marks exist, record when clean/down, never block
// unless FRONTIER_HYGIENE_BLOCK=1 and suspicious > 0.
func Disposition(r *Report) string {
	if r == nil {
		return "record"
	}
	block := os.Getenv("FRONTIER_HYGIENE_BLOCK") == "1" || os.Getenv("FRONTIER_HYGIENE_BLOCK") == "true"
	if block && r.Healthy && r.Suspicious > 0 {
		return "block"
	}
	if r.Suspicious > 0 {
		return "advise"
	}
	return "record"
}

// BlocksGate is true only under FRONTIER_HYGIENE_BLOCK and suspicious findings.
func BlocksGate(r *Report) bool {
	return Disposition(r) == "block"
}
