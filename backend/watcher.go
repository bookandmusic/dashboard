package main

import (
	"log"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher 监听配置文件所在目录（而非文件本身）。
// 编辑器保存通常是「写临时文件 → rename 替换」，inode 会变更，
// 直接监听文件会丢失事件；监听目录后过滤文件名即可覆盖 Write 与 Create 两种路径。
type Watcher struct {
	path     string
	store    *Store
	debounce time.Duration
	done     chan struct{}
}

func NewWatcher(path string, store *Store) *Watcher {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	return &Watcher{
		path:     abs,
		store:    store,
		debounce: 100 * time.Millisecond,
		done:     make(chan struct{}),
	}
}

func (w *Watcher) Stop() { close(w.done) }

// Start 阻塞运行，直到 Stop 被调用或 watcher 出错。
func (w *Watcher) Start() {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("watcher: %v", err)
		return
	}
	defer fw.Close()

	dir := filepath.Dir(w.path)
	if err := fw.Add(dir); err != nil {
		log.Printf("watcher: watch %s: %v", dir, err)
		return
	}
	log.Printf("watcher: watching %s", w.path)

	base := filepath.Base(w.path)
	var timer *time.Timer
	for {
		select {
		case <-w.done:
			return
		case ev, ok := <-fw.Events:
			if !ok {
				return
			}
			if filepath.Base(ev.Name) != base {
				continue
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Chmod) == 0 {
				continue
			}
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(w.debounce, w.reload)
		case err, ok := <-fw.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)
		}
	}
}

func (w *Watcher) reload() {
	cfg, err := loadConfig(w.path)
	if err != nil {
		log.Printf("config reload failed, keeping old config: %v", err)
		return
	}
	w.store.Set(cfg)
	log.Printf("config reloaded: %d groups", len(cfg.Groups))
}
