package hygiene

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHygieneExt(t *testing.T) {
	if !HygieneExt("notes.md") || !HygieneExt("shot.PNG") || HygieneExt("app.exe") {
		t.Fatal("ext filter")
	}
}

func TestInspectAndClean(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"version":"test"}`))
	})
	mux.HandleFunc("/capabilities", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"version":"test"}`))
	})
	mux.HandleFunc("/inspect", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true, "kind": "text", "suspicious": true,
			"report": "zero-width: 1",
		})
	})
	mux.HandleFunc("/clean", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true, "kind": "text",
			"cleaned": base64.StdEncoding.EncodeToString([]byte("clean")),
			"report":  "stripped",
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	dir := t.TempDir()
	p := filepath.Join(dir, "draft.md")
	if err := os.WriteFile(p, []byte("hello\u200b"), 0o644); err != nil {
		t.Fatal(err)
	}

	rep := InspectFiles(srv.URL, dir, []string{p})
	if !rep.Healthy || rep.Suspicious != 1 {
		t.Fatalf("inspect: %+v", rep)
	}
	if Disposition(rep) != "advise" {
		t.Fatalf("disposition %s", Disposition(rep))
	}

	out, f, err := CleanFile(srv.URL, p, "draft.md", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(out, ".cleaned.md") {
		t.Fatalf("out path %s", out)
	}
	got, _ := os.ReadFile(out)
	if string(got) != "clean" {
		t.Fatalf("cleaned bytes %q kind %s", got, f.Kind)
	}
}

func TestHealthDown(t *testing.T) {
	rep := InspectFiles("http://127.0.0.1:1", ".", nil)
	if rep.Healthy {
		t.Fatal("expected down")
	}
	if Disposition(rep) != "record" {
		t.Fatalf("down should not advise, got %s", Disposition(rep))
	}
}

func TestResolveTargetsExplicit(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.md")
	b := filepath.Join(dir, "b.exe")
	_ = os.WriteFile(a, []byte("x"), 0o644)
	_ = os.WriteFile(b, []byte("x"), 0o644)
	got := ResolveTargets(dir, []string{"a.md", "b.exe"})
	if len(got) != 1 || !strings.HasSuffix(got[0], "a.md") {
		t.Fatalf("%v", got)
	}
}
