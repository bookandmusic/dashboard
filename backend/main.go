package main

import (
	"embed"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

//go:embed frontend/dist
var staticFS embed.FS

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	cfgPath := envOr("CONFIG_PATH", "config.yaml")
	cfg, err := loadConfig(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	store := NewStore(cfg)
	log.Printf("config loaded: %s (%d groups)", cfg.Title, len(cfg.Groups))

	watcher := NewWatcher(cfgPath, store)
	go watcher.Start()

	collector := NewCollector(envOr("HOST_PROC", "/proc"), envOr("HOST_ROOT", ""))
	go collector.Run(10 * time.Second)

	addr := envOr("LISTEN_ADDR", ":8080")
	iconsDir := filepath.Join(filepath.Dir(cfgPath), "icons")
	server := &http.Server{
		Addr:              addr,
		Handler:           NewServer(store, collector, staticFS, iconsDir),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("dashboard listening on %s", addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
