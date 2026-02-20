package dashboard

import (
	"embed"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

//go:embed dist/*
var distFS embed.FS

// distDir is the subtree of distFS that contains the built SPA (without the "dist" prefix in URL path).
var distDir, _ = fs.Sub(distFS, "dist")

const dashboardPrefix = "/dashboard"

// Handler returns an http.Handler that serves the dashboard SPA.
// If CORTEX_ENV=dev, it proxies to the Vite dev server at http://localhost:5173.
// Otherwise it serves embedded files with SPA fallback (unknown paths -> index.html).
func Handler() http.Handler {
	if os.Getenv("CORTEX_ENV") == "dev" {
		slog.Info("dashboard: dev mode, proxying to Vite at http://localhost:5173")
		return devProxy()
	}
	return http.StripPrefix(dashboardPrefix, &spaFSHandler{fs: http.FS(distDir)})
}

// spaFSHandler serves files from fs and returns index.html for non-file paths (SPA fallback).
type spaFSHandler struct {
	fs http.FileSystem
}

func (h *spaFSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}
	f, err := h.fs.Open(path)
	if err != nil {
		// SPA fallback: serve index.html so client-side routing can handle it
		f, _ = h.fs.Open("index.html")
		if f != nil {
			defer f.Close()
			if stat, _ := f.Stat(); stat != nil && !stat.IsDir() {
				if rs, ok := f.(io.ReadSeeker); ok {
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					http.ServeContent(w, r, "index.html", stat.ModTime(), rs)
					return
				}
			}
		}
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil || stat.IsDir() {
		http.NotFound(w, r)
		return
	}
	rs, ok := f.(io.ReadSeeker)
	if !ok {
		slog.Error("dashboard: file does not implement ReadSeeker", "path", path)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, path, stat.ModTime(), rs)
}

func devProxy() http.Handler {
	target, err := url.Parse("http://localhost:5173")
	if err != nil {
		slog.Error("dashboard dev proxy: invalid target", "error", err)
		return http.NotFoundHandler()
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		// Strip /dashboard prefix so Vite gets /, /memories, etc.
		path := strings.TrimPrefix(req.URL.Path, dashboardPrefix)
		if path == "" {
			path = "/"
		}
		req.URL.Path = path
		if req.URL.RawPath != "" {
			req.URL.RawPath = path
		}
	}
	return http.StripPrefix(dashboardPrefix, proxy)
}
