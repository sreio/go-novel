package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	fepub "github.com/sreio/go-novel/internal/format/epub"
	fpdf "github.com/sreio/go-novel/internal/format/pdf"
	ftxt "github.com/sreio/go-novel/internal/format/txt"
	"github.com/sreio/go-novel/internal/sources"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	router      *chi.Mux
	sourcesDir  string
	concurrency int
	sources     []*sources.ConfigSource
}

func main() {
	srv := &Server{
		router:      chi.NewRouter(),
		sourcesDir:  getEnv("SOURCES_DIR", "./configs/sources"),
		concurrency: atoi(getEnv("CONCURRENCY", "8"), 8),
	}

	srv.router.Use(middleware.RealIP)
	srv.router.Use(middleware.Logger)
	srv.router.Use(middleware.Recoverer)
	srv.router.Use(cors)

	srv.router.Get("/api/health", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	srv.router.Get("/api/search", srv.handleSearch)
	srv.router.Get("/api/books/chapters", srv.handleChapters)
	srv.router.Get("/api/chapter", srv.handleChapter)
	srv.router.Get("/api/download", srv.handleDownload)

	fs := http.FileServer(http.Dir("./web/dist"))
	srv.router.Handle("/*", fs)

	if err := srv.reloadSources(); err != nil {
		log.Fatalf("load sources: %v", err)
	}

	log.Println("listen :8080")
	http.ListenAndServe(":8080", srv.router)
}

func (s *Server) reloadSources() error {
	cfgs, err := sources.LoadFromDir(s.sourcesDir)
	if err != nil {
		return err
	}
	var out []*sources.ConfigSource
	for _, c := range cfgs {
		sc, err := sources.NewFromConfig(c)
		if err != nil {
			return err
		}
		out = append(out, sc)
	}
	s.sources = out
	return nil
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing q"})
		return
	}

	type row struct{ Title, Author, ID, Source string }
	var result []row
	ctx := r.Context()
	for _, src := range s.sources {
		items, _ := src.Search(ctx, q, 1)
		for _, it := range items {
			result = append(result, row{Title: it.Title, Author: it.Author, ID: it.ID, Source: src.Name()})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": result})
}

func (s *Server) handleChapters(w http.ResponseWriter, r *http.Request) {
	u := strings.TrimSpace(r.URL.Query().Get("url"))
	if u == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing url"})
		return
	}
	src := chooseSourceByURL(s.sources, u)
	if src == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no source"})
		return
	}

	ctx := r.Context()
	chs, err := src.Chapters(ctx, u, u)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	type row struct {
		Title, URL string
		Index      int
	}
	out := make([]row, len(chs))
	for i, c := range chs {
		out[i] = row{Title: c.Title, URL: c.URL, Index: c.Index}
	}
	writeJSON(w, http.StatusOK, map[string]any{"chapters": out})
}

func (s *Server) handleChapter(w http.ResponseWriter, r *http.Request) {
	u := strings.TrimSpace(r.URL.Query().Get("url"))
	if u == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing url"})
		return
	}
	limit := atoi(r.URL.Query().Get("limit"), 1000)
	full := r.URL.Query().Get("full") == "1"

	src := chooseSourceByURL(s.sources, u)
	if src == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no source"})
		return
	}

	ctx := r.Context()
	ch := sources.Chapter{Title: "", URL: u, Index: 0, ID: u}
	content, err := src.Content(ctx, ch)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	body := content
	if !full {
		body = truncateRunes(content, limit)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"url":     u,
		"title":   ch.Title,
		"content": body,
		"full":    full,
		"limit":   limit,
	})
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	u := strings.TrimSpace(r.URL.Query().Get("url"))
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if u == "" || (format != "txt" && format != "epub" && format != "pdf") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing or invalid url/format"})
		return
	}

	src := chooseSourceByURL(s.sources, u)
	if src == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no source"})
		return
	}

	ctx := r.Context()
	chs, err := src.Chapters(ctx, u, u)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if len(chs) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no chapters"})
		return
	}

	type text struct{ Title, Content string }
	out := make([]text, len(chs))

	sem := make(chan struct{}, s.concurrency)
	g, gctx := errgroup.WithContext(ctx)
	for i := range chs {
		i := i
		g.Go(func() error {
			select {
			case sem <- struct{}{}:
			case <-gctx.Done():
				return gctx.Err()
			}
			defer func() { <-sem }()
			content, err := src.Content(gctx, chs[i])
			if err != nil {
				return err
			}
			out[i] = text{Title: chs[i].Title, Content: content}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if err := os.MkdirAll("./outputs", 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	name := safeNameFromURL(u) + "." + format
	dst := filepath.Join("./outputs", name)

	switch format {
	case "txt":
		conv := make([]ftxt.Chapter, len(out))
		for i, c := range out {
			conv[i] = struct{ Title, Content string }{Title: c.Title, Content: c.Content}
		}
		if err := ftxt.Save(dst, conv); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	case "epub":
		chapters := make([]fepub.Chapter, len(out))
		for i, c := range out {
			chapters[i] = fepub.Chapter{Title: c.Title, Content: c.Content}
		}
		if err := fepub.Save(dst, fepub.Meta{Title: name, Author: ""}, chapters); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	case "pdf":
		chapters := make([]fpdf.Chapter, len(out))
		for i, c := range out {
			chapters[i] = fpdf.Chapter{Title: c.Title, Content: c.Content}
		}
		if err := fpdf.Save(dst, fpdf.Meta{Title: name, Author: ""}, chapters); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+name)
	http.ServeFile(w, r, dst)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
func atoi(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

func chooseSourceByURL(all []*sources.ConfigSource, u string) *sources.ConfigSource {
	for _, s := range all {
		if s == nil || u == "" {
			continue
		}
		if strings.Contains(u, s.ID()) || strings.Contains(u, s.Name()) {
			return s
		}
	}
	if len(all) > 0 {
		return all[0]
	}
	return nil
}

func safeNameFromURL(u string) string {
	n := filepath.Base(u)
	if n == "/" || n == "." || n == "" {
		n = "book"
	}
	n = strings.ReplaceAll(n, "?", "_")
	n = strings.ReplaceAll(n, "&", "_")
	return n
}

func truncateRunes(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= limit {
		return s
	}
	i := 0
	for idx := range s {
		if i == limit {
			return s[:idx] + "…"
		}
		i++
	}
	return s
}

// 允许基础 CORS 以便前端本地开发
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
