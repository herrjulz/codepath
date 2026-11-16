package server

import (
	"io"
	"io/fs"
	"net/http"
	"strings"
)

// DistFS returns the SPA filesystem rooted at the embedded dist directory.
func DistFS(dist fs.FS) (http.FileSystem, error) {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		return nil, err
	}
	return http.FS(sub), nil
}

func assetsHandler(assets http.FileSystem) http.Handler {
	files := http.FileServer(assets)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !allowRequest(w, r) {
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/")
		f, err := assets.Open(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		info, err := f.Stat()
		_ = f.Close()
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		files.ServeHTTP(w, r)
	})
}

func serveIndex(w http.ResponseWriter, _ *http.Request, assets http.FileSystem) {
	f, err := assets.Open("index.html")
	if err != nil {
		http.Error(w, "CodePath UI is not built", http.StatusNotFound)
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, "CodePath UI is not built", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
