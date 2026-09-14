// Package catalog lists the wakalabs skills and agents imported into
// frontier-ship (skills/ and agents/ in the source tree) so they can be used
// through `frontier skills` / `frontier agents`.
//
// Entry semantics (bounded walk):
//   - a directory at depth <= 2 that owns a primary doc (SKILL.md or README.md)
//     is one entry, named by its relative slash path (e.g. "grok/dogfood");
//   - a doc file at depth <= 3 whose parent directory owns no primary doc is
//     one entry named by its relative path minus extension
//     (e.g. "github/git-agent", "grok/rules/local-build-pipeline");
//   - support trees (references/, scripts/, docs/) and dot-dirs are skipped.
//
// Resolution order for the roots (local-first, single habitat):
//
//	FRONTIER_SKILLS_DIR / FRONTIER_AGENTS_DIR  (explicit override)
//	D:\frontier\skills / D:\frontier\agents     (deployed copy)
//	<dir of running binary>\skills / \agents    (next to the binary)
//	C:\Users\waka\src\frontier-ship\skills / \agents  (source checkout)
package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Entry is one skill or agent under the catalog root.
type Entry struct {
	Name  string // relative slash path (minus extension for bare files)
	Path  string // primary file to show
	Title string // first heading or description line
}

// SkillsDir resolves the imported skills root.
func SkillsDir() string {
	if v := strings.TrimSpace(os.Getenv("FRONTIER_SKILLS_DIR")); v != "" {
		return v
	}
	return resolveDir("skills")
}

// AgentsDir resolves the imported agents root.
func AgentsDir() string {
	if v := strings.TrimSpace(os.Getenv("FRONTIER_AGENTS_DIR")); v != "" {
		return v
	}
	return resolveDir("agents")
}

func resolveDir(name string) string {
	exe, _ := os.Executable()
	candidates := []string{
		filepath.Join("D:\\frontier", name),
	}
	if exe != "" {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), name))
	}
	candidates = append(candidates,
		filepath.Join("C:\\Users\\waka", "src", "frontier-ship", name),
	)
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	return candidates[len(candidates)-1] // last chance; List will report missing
}

// List walks one catalog root and returns its entries, sorted by name.
func List(root string) ([]Entry, error) {
	st, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, &os.PathError{Op: "catalog", Path: root, Err: os.ErrInvalid}
	}
	var out []Entry
	walkLevel(root, "", 0, &out)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// walkLevel descends to depth 3 (root itself is depth 0); rel is slash-separated.
func walkLevel(abs, rel string, depth int, out *[]Entry) {
	items, err := os.ReadDir(abs)
	if err != nil {
		return
	}
	primary := primaryFile(abs)
	if depth >= 1 && depth <= 2 && primary != "" {
		*out = append(*out, Entry{Name: rel, Path: primary, Title: titleOf(primary)})
		return // a dir with its own doc is one entry; support files stay inside it
	}
	for _, it := range items {
		name := it.Name()
		if strings.HasPrefix(name, ".") || strings.EqualFold(name, "README.md") || strings.EqualFold(name, "SKILL.md") {
			continue
		}
		childRel := name
		if rel != "" {
			childRel = rel + "/" + name
		}
		if it.IsDir() {
			if skipDirName(name) || depth >= 3 {
				continue
			}
			walkLevel(filepath.Join(abs, name), childRel, depth+1, out)
			continue
		}
		if depth >= 1 && isDoc(name) {
			*out = append(*out, Entry{Name: trimExt(childRel), Path: filepath.Join(abs, name), Title: titleOf(filepath.Join(abs, name))})
		}
	}
}

func skipDirName(name string) bool {
	switch strings.ToLower(name) {
	case "references", "scripts", "docs", "assets":
		return true
	}
	return false
}

// primaryFile is only SKILL.md / README.md — the dir's own defining doc.
// Bare doc files (a profile next to peers) become entries on their own.
func primaryFile(dir string) string {
	for _, name := range []string{"SKILL.md", "README.md", "readme.md"} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func isDoc(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".md", ".txt", ".ps1", ".py", ".json", ".yml", ".yaml":
		return true
	}
	return false
}

func trimExt(name string) string {
	for _, suf := range []string{".instructions.md", ".agent.md", ".md", ".txt", ".ps1", ".py", ".json", ".yml", ".yaml"} {
		if strings.HasSuffix(strings.ToLower(name), suf) {
			return name[:len(name)-len(suf)]
		}
	}
	return strings.TrimSuffix(name, filepath.Ext(name))
}

// titleOf returns the first heading / name line of a file (one line).
func titleOf(path string) string {
	if path == "" {
		return ""
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(b), "\n")
	var fallback string
	for i := 0; i < len(lines) && i < 25; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "---") {
			continue
		}
		if strings.HasPrefix(line, "#") {
			return strings.TrimSpace(strings.TrimLeft(line, "#"))
		}
		if strings.HasPrefix(line, "name:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		}
		low := strings.ToLower(line)
		if strings.HasPrefix(low, "description:") || strings.HasPrefix(low, "tools:") ||
			strings.HasPrefix(low, "user-invocable:") || strings.HasPrefix(low, "argument-hint:") ||
			strings.HasPrefix(low, "when-to-use:") || strings.HasPrefix(low, "applyto:") {
			continue
		}
		if fallback == "" {
			fallback = truncateLine(line, 90)
		}
	}
	if fallback != "" {
		return fallback
	}
	return ""
}

// Show resolves one named entry: exact rel path, or a unique basename suffix
// ("dogfood" finds "grok/dogfood"), case-insensitive.
func Show(root, name string) (string, error) {
	if name == "" {
		return "", &os.PathError{Op: "catalog show", Path: root, Err: os.ErrInvalid}
	}
	entries, err := List(root)
	if err != nil {
		return "", err
	}
	want := strings.ToLower(strings.TrimSpace(name))
	var matches []Entry
	for _, e := range entries {
		if strings.EqualFold(e.Name, want) {
			matches = append(matches, e)
			continue
		}
		base := strings.ToLower(e.Name)
		if i := strings.LastIndex(base, "/"); i >= 0 {
			base = base[i+1:]
		}
		if base == want {
			matches = append(matches, e)
		}
	}
	if len(matches) == 1 {
		return matches[0].Path, nil
	}
	if len(matches) > 1 {
		names := make([]string, 0, len(matches))
		for _, m := range matches {
			names = append(names, m.Name)
		}
		sort.Strings(names)
		return "", fmt.Errorf("catalog show %q is ambiguous: %s", name, strings.Join(names, ", "))
	}
	return "", &os.PathError{Op: "catalog show", Path: filepath.Join(root, name), Err: os.ErrNotExist}
}

func truncateLine(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
