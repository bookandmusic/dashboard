package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	cfg, err := parseConfig([]byte(validYAML))
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(cfg)
	collector := NewCollector("/proc", "")
	if err := collector.Sample(); err != nil {
		t.Skipf("no /proc on this platform: %v", err)
	}
	fsys := fstest.MapFS{
		"frontend/dist/index.html":       &fstest.MapFile{Data: []byte("<html>index</html>")},
		"frontend/dist/assets/app.js":    &fstest.MapFile{Data: []byte("console.log(1)")},
		"frontend/dist/assets/style.css": &fstest.MapFile{Data: []byte("body{}")},
	}
	return NewServer(store, collector, fsys, t.TempDir())
}

func doGet(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAPIConfig(t *testing.T) {
	h := newTestServer(t)
	rec := doGet(t, h, "/api/config")
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	var cfg Config
	if err := json.Unmarshal(rec.Body.Bytes(), &cfg); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if cfg.Title != "内部工具导航" {
		t.Errorf("Title = %q", cfg.Title)
	}
}

func TestAPIStats(t *testing.T) {
	h := newTestServer(t)
	rec := doGet(t, h, "/api/stats")
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	var s Stats
	if err := json.Unmarshal(rec.Body.Bytes(), &s); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if s.Memory.Total == 0 {
		t.Error("Memory.Total = 0, expected real value from /proc")
	}
}

func TestStaticIndex(t *testing.T) {
	h := newTestServer(t)
	rec := doGet(t, h, "/")
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	if body, _ := io.ReadAll(rec.Body); string(body) != "<html>index</html>" {
		t.Errorf("body = %q", body)
	}
}

func TestStaticAsset(t *testing.T) {
	h := newTestServer(t)
	rec := doGet(t, h, "/assets/app.js")
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	if body, _ := io.ReadAll(rec.Body); string(body) != "console.log(1)" {
		t.Errorf("body = %q", body)
	}
}

func TestSPAFallback(t *testing.T) {
	h := newTestServer(t)
	rec := doGet(t, h, "/some/deep/route")
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	if body, _ := io.ReadAll(rec.Body); string(body) != "<html>index</html>" {
		t.Errorf("SPA fallback body = %q, want index.html", body)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	h := newTestServer(t)
	req := httptest.NewRequest("POST", "/api/config", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /api/config status = %d, want 405", rec.Code)
	}
}

func TestIconsServed(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "gitea.svg"), []byte("<svg>g</svg>"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := parseConfig([]byte(validYAML))
	if err != nil {
		t.Fatal(err)
	}
	collector := NewCollector("/proc", "")
	if err := collector.Sample(); err != nil {
		t.Skipf("no /proc on this platform: %v", err)
	}
	h := NewServer(NewStore(cfg), collector, fstest.MapFS{"frontend/dist/index.html": &fstest.MapFile{Data: []byte("i")}}, dir)
	rec := doGet(t, h, "/icons/gitea.svg")
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	if body, _ := io.ReadAll(rec.Body); string(body) != "<svg>g</svg>" {
		t.Errorf("body = %q", body)
	}
}

func TestIconsMissingDir(t *testing.T) {
	cfg, err := parseConfig([]byte(validYAML))
	if err != nil {
		t.Fatal(err)
	}
	collector := NewCollector("/proc", "")
	if err := collector.Sample(); err != nil {
		t.Skipf("no /proc on this platform: %v", err)
	}
	h := NewServer(NewStore(cfg), collector, fstest.MapFS{"frontend/dist/index.html": &fstest.MapFile{Data: []byte("i")}}, filepath.Join(t.TempDir(), "nope"))
	rec := doGet(t, h, "/icons/gitea.svg")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
