package main

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
)

func NewServer(store *Store, collector *Collector, staticFS fs.FS, iconsDir string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/config", onlyGET(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, store.Get())
	}))
	mux.HandleFunc("/api/stats", onlyGET(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, collector.Current())
	}))
	mux.Handle("/icons/", iconsHandler(iconsDir))
	mux.Handle("/", staticHandler(staticFS))
	return mux
}

func onlyGET(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h(w, r)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

// iconsHandler 提供用户自管的图标文件（config 中 icon 相对路径），目录不存在时一律 404。
func iconsHandler(dir string) http.Handler {
	if dir == "" {
		return http.NotFoundHandler()
	}
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return http.NotFoundHandler()
	}
	fileServer := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r2 := r.Clone(r.Context())
		r2.URL.Path = strings.TrimPrefix(r.URL.Path, "/icons")
		if r2.URL.Path == "" || r2.URL.Path == "/" {
			http.NotFound(w, r)
			return
		}
		fileServer.ServeHTTP(w, r2)
	})
}

// staticHandler 提供嵌入的静态文件，未命中的路径回退到 index.html（SPA fallback）。
func staticHandler(staticFS fs.FS) http.Handler {
	dist, err := fs.Sub(staticFS, "frontend/dist")
	if err != nil {
		log.Fatalf("embed fs: %v", err)
	}
	fileServer := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p != "" {
			if _, err := fs.Stat(dist, p); err != nil {
				r.URL.Path = "/"
			}
		}
		fileServer.ServeHTTP(w, r)
	})
}
