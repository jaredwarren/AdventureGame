package editorweb

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestPlayRequiresRepoMakefile(t *testing.T) {
	_, ts := newTestServer(t)
	resp := do(t, ts, "POST", "/api/play", nil)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500 when there is no Makefile above the maps dir", resp.StatusCode)
	}
	var body apiError
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Kind != "play_failed" {
		t.Fatalf("kind %q, want play_failed", body.Kind)
	}
}

func TestPlayRunsMake(t *testing.T) {
	root := t.TempDir()
	maps := filepath.Join(root, "assets", "maps")
	if err := os.MkdirAll(maps, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module "+gameModule+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Makefile"), []byte("run:\n\t@true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeFixture(maps, "F-5", validMap()); err != nil {
		t.Fatal(err)
	}

	s, err := New(Options{MapsDir: maps, Token: testToken})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.stopPlay)
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)

	resp := do(t, ts, "POST", "/api/play", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", resp.StatusCode)
	}
	got := decode[map[string]any](t, resp)
	if got["ok"] != true {
		t.Fatalf("ok=%v", got["ok"])
	}
	gotDir, err := filepath.EvalSymlinks(got["dir"].(string))
	if err != nil {
		t.Fatal(err)
	}
	wantDir, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if gotDir != wantDir {
		t.Fatalf("dir=%s, want %s", gotDir, wantDir)
	}
}
