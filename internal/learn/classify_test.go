package learn

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClassifyAppCompose(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("services:\n  web:\n    image: x\n"), 0o644)
	_ = os.Mkdir(filepath.Join(dir, "ui"), 0o755)
	_ = os.Mkdir(filepath.Join(dir, "data"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "ui", "server.py"), []byte("print(1)\n"), 0o644)

	ls, err := Classify(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ls.Kind != "app_compose" {
		t.Fatalf("kind=%s want app_compose", ls.Kind)
	}
	art, err := WriteArtifacts(ls)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(art.Markdown); err != nil {
		t.Fatal(err)
	}
}

func TestClassifySkipsVenv(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("services:\n  app:\n    image: x\n"), 0o644)
	_ = os.Mkdir(filepath.Join(dir, "data"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "main.py"), []byte("print(1)\n"), 0o644)
	_ = os.MkdirAll(filepath.Join(dir, "venv", "Lib"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "venv", "Lib", "noise.py"), []byte("x=1\n"), 0o644)

	ls, err := Classify(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ls.LangCounts[".py"] != 1 {
		t.Fatalf("py count=%d want 1 (venv must not be walked); langs=%v", ls.LangCounts[".py"], ls.LangCounts)
	}
}

func TestClassifyTooling(t *testing.T) {
	dir := t.TempDir()
	// Name the temp dir frontier by nesting
	root := filepath.Join(dir, "frontier")
	_ = os.MkdirAll(filepath.Join(root, "bin"), 0o755)
	_ = os.MkdirAll(filepath.Join(root, "ledgers"), 0o755)
	ls, err := Classify(root)
	if err != nil {
		t.Fatal(err)
	}
	if ls.Kind != "tooling" {
		t.Fatalf("kind=%s want tooling", ls.Kind)
	}
}
