// Package learn implements L (Learn / Landscape): ingest before V/S change.
package learn

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	MaxTopEntries = 40
	MaxBriefBytes = 12 * 1024
)

// Landscape is a sealed, programmatic understanding of one project root.
// Security concerns (secret surfaces, OWASP) belong in Guard (G), not Learn.
type Landscape struct {
	Root            string         `json:"root"`
	Name            string         `json:"name"`
	Kind            string         `json:"kind"`
	Confidence      string         `json:"confidence"` // high | medium | low
	Reasons         []string       `json:"reasons"`
	HasGit          bool           `json:"has_git"`
	HasCompose      bool           `json:"has_compose"`
	ComposeServices []string       `json:"compose_services,omitempty"`
	LangCounts      map[string]int `json:"lang_counts"`
	Manifests       []string       `json:"manifests"`
	TopEntries      []string       `json:"top_entries"`
	FileCountApprox int            `json:"file_count_approx"`
	Notes           []string       `json:"notes,omitempty"`
	Stamp           string         `json:"stamp"`
}

// Artifacts written by classify.
type Artifacts struct {
	Dir      string
	JSON     string
	Markdown string
	Latest   string
	Stamp    string
}

// Classify inspects root read-only and returns a Landscape.
func Classify(root string) (*Landscape, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", root)
	}

	ls := &Landscape{
		Root:       root,
		Name:       filepath.Base(root),
		LangCounts: map[string]int{},
		Stamp:      time.Now().UTC().Format("20060102T150405Z"),
	}

	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		name := e.Name()
		if len(ls.TopEntries) < MaxTopEntries {
			suffix := ""
			if e.IsDir() {
				suffix = "/"
			}
			ls.TopEntries = append(ls.TopEntries, name+suffix)
		}
		low := strings.ToLower(name)
		if name == ".git" && e.IsDir() {
			ls.HasGit = true
		}
		if low == "docker-compose.yml" || low == "docker-compose.yaml" || low == "compose.yml" || low == "compose.yaml" {
			ls.HasCompose = true
			ls.ComposeServices = parseComposeServices(filepath.Join(root, name))
		}
		if isManifestName(low) || strings.HasSuffix(low, ".tf") {
			ls.Manifests = append(ls.Manifests, name)
		}
	}
	sort.Strings(ls.TopEntries)
	sort.Strings(ls.Manifests)

	langs, moreManifests, files := walkLight(root)
	ls.LangCounts = langs
	ls.Manifests = uniqueSorted(append(ls.Manifests, moreManifests...))
	ls.FileCountApprox = files

	ls.Kind, ls.Confidence, ls.Reasons = inferKind(ls)
	return ls, nil
}

// WriteArtifacts writes .frontier/learn/L-*.{json,md} under root.
func WriteArtifacts(ls *Landscape) (*Artifacts, error) {
	dir := filepath.Join(ls.Root, ".frontier", "learn")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	base := "L-" + ls.Stamp
	jsPath := filepath.Join(dir, base+".json")
	mdPath := filepath.Join(dir, base+".md")
	latest := filepath.Join(dir, "LATEST")

	raw, _ := json.MarshalIndent(ls, "", "  ")
	if err := os.WriteFile(jsPath, raw, 0o644); err != nil {
		return nil, err
	}
	md := renderMarkdown(ls)
	if len(md) > MaxBriefBytes {
		md = md[:MaxBriefBytes] + "\n\n...[truncated for token budget]\n"
	}
	if err := os.WriteFile(mdPath, []byte(md), 0o644); err != nil {
		return nil, err
	}
	_ = os.WriteFile(latest, []byte(base+"\n"), 0o644)
	return &Artifacts{Dir: dir, JSON: jsPath, Markdown: mdPath, Latest: latest, Stamp: ls.Stamp}, nil
}

func renderMarkdown(ls *Landscape) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Frontier L classify - %s\n\n", ls.Name)
	fmt.Fprintf(&b, "- root: `%s`\n", ls.Root)
	fmt.Fprintf(&b, "- kind: **%s** (confidence %s)\n", ls.Kind, ls.Confidence)
	fmt.Fprintf(&b, "- git: %v | compose: %v | approx files: %d\n\n", ls.HasGit, ls.HasCompose, ls.FileCountApprox)
	if len(ls.Reasons) > 0 {
		b.WriteString("## Why\n\n")
		for _, r := range ls.Reasons {
			fmt.Fprintf(&b, "- %s\n", r)
		}
		b.WriteString("\n")
	}
	if len(ls.ComposeServices) > 0 {
		b.WriteString("## Compose services\n\n")
		for _, s := range ls.ComposeServices {
			fmt.Fprintf(&b, "- %s\n", s)
		}
		b.WriteString("\n")
	}
	b.WriteString("## Inventory\n\n")
	if len(ls.LangCounts) == 0 {
		b.WriteString("- langs: _(none counted)_\n")
	} else {
		b.WriteString("- langs:")
		for k, v := range ls.LangCounts {
			fmt.Fprintf(&b, " %s=%d", k, v)
		}
		b.WriteString("\n")
	}
	if len(ls.Manifests) > 0 {
		b.WriteString("- manifests:\n")
		for _, m := range ls.Manifests {
			fmt.Fprintf(&b, "  - %s\n", m)
		}
	}
	b.WriteString("\n## Top entries\n\n")
	for _, e := range ls.TopEntries {
		fmt.Fprintf(&b, "- %s\n", e)
	}
	b.WriteString("\n## Next\n\n")
	b.WriteString("- `frontier guard` (G) for security exam on this tree\n")
	b.WriteString("- `frontier enhance guard` for residual host review\n")
	b.WriteString("- `frontier slim` (S) planned — coverage after a real run profile\n")
	return b.String()
}

func inferKind(ls *Landscape) (kind, conf string, reasons []string) {
	name := strings.ToLower(ls.Name)
	top := strings.Join(ls.TopEntries, " ")

	switch {
	case name == "frontier" || (containsEntry(ls.TopEntries, "bin/") && containsEntry(ls.TopEntries, "ledgers/")):
		return "tooling", "high", []string{"Frontier Ship tooling (bin/ledgers/releases)"}
	case name == "ollama-data" || (name == "ollama" && !ls.HasCompose):
		return "data_volume", "high", []string{"model/weight data volume, not an app repo"}
	case name == "immich" && ls.HasCompose:
		return "media_stack", "high", []string{"Immich compose project", "external photos stay outside this tree"}
	case containsEntry(ls.TopEntries, "skills/") && !ls.HasCompose:
		return "skills_pack", "medium", []string{"skill registry style tree"}
	case name == "waka-net" || (name != "immich" && (containsEntry(ls.TopEntries, "gateway/") || containsEntry(ls.TopEntries, "vigil/"))):
		reasons = []string{"network edge indicators (gateway/vigil/waka-net)"}
		if ls.HasCompose {
			reasons = append(reasons, "has docker-compose")
		}
		return "network_edge", "high", reasons
	case ls.HasCompose && (containsEntry(ls.TopEntries, "ui/") || containsEntry(ls.TopEntries, "data/") || strings.Contains(top, "server.py") || containsEntry(ls.TopEntries, "library/")):
		return "app_compose", "high", []string{"docker-compose + app layout (ui/data/server)"}
	case ls.HasCompose:
		return "compose_project", "medium", []string{"docker-compose present"}
	case ls.HasGit && len(ls.LangCounts) > 0:
		return "app_repo", "medium", []string{"git repo with source files"}
	default:
		return "unknown", "low", []string{"insufficient signals - inspect manually"}
	}
}

func walkLight(root string) (langs map[string]int, manifests []string, files int) {
	langs = map[string]int{}
	skipDir := map[string]bool{
		".git": true, "node_modules": true, "vendor": true, ".frontier": true,
		"__pycache__": true, "dist": true, "build": true, "library": true,
		"covers": true, "postgres": true, "models": true, "blobs": true,
		"venv": true, ".venv": true,
	}
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if info.IsDir() {
			base := info.Name()
			if skipDir[base] {
				return filepath.SkipDir
			}
			// skip deep ollama/immich weight trees
			if strings.Count(rel, "/") > 6 {
				return filepath.SkipDir
			}
			return nil
		}
		files++
		if files > 8000 {
			return filepath.SkipAll
		}
		base := strings.ToLower(filepath.Base(path))
		ext := strings.ToLower(filepath.Ext(base))
		switch ext {
		case ".go", ".py", ".js", ".ts", ".tsx", ".jsx", ".rb", ".php", ".java", ".cs",
			".tf", ".yml", ".yaml", ".json", ".sh", ".ps1", ".html", ".css":
			langs[ext]++
		}
		if isManifestName(base) {
			manifests = append(manifests, rel)
		}
		return nil
	})
	return langs, manifests, files
}

func parseComposeServices(path string) []string {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	// Minimal YAML-ish: lines that look like service keys under services:
	lines := strings.Split(string(b), "\n")
	inServices := false
	var out []string
	for _, line := range lines {
		trim := strings.TrimRight(line, "\r")
		if strings.HasPrefix(strings.TrimSpace(trim), "#") {
			continue
		}
		if strings.TrimSpace(trim) == "services:" {
			inServices = true
			continue
		}
		if !inServices {
			continue
		}
		// left services block on next top-level key
		if len(trim) > 0 && trim[0] != ' ' && trim[0] != '\t' && strings.HasSuffix(strings.TrimSpace(trim), ":") {
			break
		}
		// one indent level key: "  name:"
		if strings.HasPrefix(trim, "  ") && !strings.HasPrefix(trim, "   ") && strings.HasSuffix(strings.TrimSpace(trim), ":") {
			name := strings.TrimSuffix(strings.TrimSpace(trim), ":")
			if name != "" && !strings.Contains(name, " ") {
				out = append(out, name)
			}
		}
	}
	return out
}

func isManifestName(base string) bool {
	switch base {
	case "go.mod", "package.json", "requirements.txt", "pipfile", "cargo.toml",
		"composer.json", "gemfile", "pom.xml", "build.gradle", "dockerfile",
		"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml":
		return true
	default:
		return strings.HasPrefix(base, "dockerfile.")
	}
}

func containsEntry(entries []string, want string) bool {
	for _, e := range entries {
		if e == want || strings.EqualFold(e, want) {
			return true
		}
	}
	return false
}

func uniqueSorted(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
