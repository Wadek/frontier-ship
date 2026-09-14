package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSkill(t *testing.T, dir, name, content string) {
	t.Helper()
	p := filepath.Join(dir, name, "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestListAndShow(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "zeta-skill", "# Zeta Skill\nbody")
	writeSkill(t, root, "alpha-skill", "---\nname: Alpha Skill\n---\nbody")
	// noise that must be skipped
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("catalog"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".hidden"), 0o755); err != nil {
		t.Fatal(err)
	}

	entries, err := List(root)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2: %+v", len(entries), entries)
	}
	if entries[0].Name != "alpha-skill" || entries[1].Name != "zeta-skill" {
		t.Errorf("want sorted names, got %q, %q", entries[0].Name, entries[1].Name)
	}
	if entries[0].Title != "Alpha Skill" {
		t.Errorf("alpha title = %q", entries[0].Title)
	}
	if entries[1].Title != "Zeta Skill" {
		t.Errorf("zeta title = %q", entries[1].Title)
	}

	path, err := Show(root, "ZETA-SKILL") // case-insensitive
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	// Windows resolves the stat case-insensitively, so compare basenames folded.
	if !strings.EqualFold(filepath.Base(filepath.Dir(path)), "zeta-skill") ||
		!strings.EqualFold(filepath.Base(path), "SKILL.md") {
		t.Errorf("show path = %s", path)
	}
	if _, err := Show(root, "missing"); err == nil {
		t.Errorf("show missing should error")
	}
}

func TestSkillsDirEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FRONTIER_SKILLS_DIR", dir)
	if got := SkillsDir(); got != dir {
		t.Errorf("SkillsDir = %s, want %s", got, dir)
	}
	t.Setenv("FRONTIER_AGENTS_DIR", dir)
	if got := AgentsDir(); got != dir {
		t.Errorf("AgentsDir = %s, want %s", got, dir)
	}
}

func TestNestedGroupsAndUniqueSuffix(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "grok/dogfood", "# dogfood\nbody")
	writeSkill(t, root, "grok/regular-git", "# regular-git\nbody")
	writeSkill(t, root, "wakagym/wakagym-400m", "# wakagym-400m\nbody")

	entries, err := List(root)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("entries = %d, want 3: %+v", len(entries), entries)
	}
	want := []string{"grok/dogfood", "grok/regular-git", "wakagym/wakagym-400m"}
	for i, e := range entries {
		if e.Name != want[i] {
			t.Errorf("entry[%d] = %s, want %s", i, e.Name, want[i])
		}
	}

	// unique basename suffix resolves
	path, err := Show(root, "dogfood")
	if err != nil {
		t.Fatalf("show suffix: %v", err)
	}
	if !strings.HasSuffix(path, filepath.Join("grok", "dogfood", "SKILL.md")) {
		t.Errorf("suffix show path = %s", path)
	}
	// exact rel path resolves too
	path, err = Show(root, "grok/regular-git")
	if err != nil {
		t.Fatalf("show rel: %v", err)
	}
	if !strings.HasSuffix(path, filepath.Join("grok", "regular-git", "SKILL.md")) {
		t.Errorf("rel show path = %s", path)
	}
}
