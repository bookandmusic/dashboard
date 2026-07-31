package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func startWatcher(t *testing.T, path string, store *Store) {
	t.Helper()
	w := NewWatcher(path, store)
	w.debounce = 20 * time.Millisecond
	go w.Start()
	t.Cleanup(func() { w.Stop() })
	time.Sleep(200 * time.Millisecond) // 等 watcher 就绪
}

func waitForTitle(t *testing.T, store *Store, want string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if store.Get().Title == want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timeout: config title never became %q (got %q)", want, store.Get().Title)
}

func TestWatcherHotReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	write := func(title string) {
		content := "title: " + title + "\ngroups:\n  - name: g\n    sites:\n      - name: s\n        addresses:\n          - {net: intranet, label: l, url: u}\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("旧标题")

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(cfg)
	startWatcher(t, path, store)

	write("新标题")
	waitForTitle(t, store, "新标题")
}

// 编辑器常见行为：写临时文件再 rename 替换（inode 变更），watcher 必须仍能感知。
func TestWatcherReloadOnRename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	base := "title: %s\ngroups:\n  - name: g\n    sites:\n      - name: s\n        addresses:\n          - {net: intranet, label: l, url: u}\n"
	if err := os.WriteFile(path, []byte("title: 甲\ngroups:\n  - name: g\n    sites:\n      - name: s\n        addresses:\n          - {net: intranet, label: l, url: u}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(cfg)
	startWatcher(t, path, store)

	tmp := filepath.Join(dir, "config.yaml.tmp")
	if err := os.WriteFile(tmp, []byte("title: 乙\ngroups:\n  - name: g\n    sites:\n      - name: s\n        addresses:\n          - {net: intranet, label: l, url: u}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, path); err != nil {
		t.Fatal(err)
	}
	waitForTitle(t, store, "乙")
	_ = base
}

// 新配置非法时保留旧配置，服务不中断。
func TestWatcherInvalidKeepsOld(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := "title: 正常\ngroups:\n  - name: g\n    sites:\n      - name: s\n        addresses:\n          - {net: intranet, label: l, url: u}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(cfg)
	startWatcher(t, path, store)

	if err := os.WriteFile(path, []byte("title: \ngroups: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	if got := store.Get().Title; got != "正常" {
		t.Errorf("invalid reload must keep old config, got title %q", got)
	}
}
