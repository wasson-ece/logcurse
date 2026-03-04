package web

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/wasson-ece/logcurse/internal/fileutil"
)

//go:embed static
var staticFS embed.FS

// Serve starts the web viewer on the given port.
func Serve(sourceFile string, port int, version string) error {
	idx, err := fileutil.BuildLineIndex(sourceFile, fileutil.DefaultIndexInterval)
	if err != nil {
		return fmt.Errorf("building line index: %w", err)
	}

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/lines", linesHandler(sourceFile, idx))
	mux.HandleFunc("/api/comments", commentsHandler(sourceFile))
	mux.HandleFunc("/api/download", downloadHandler(sourceFile))
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(version))
	})

	// Static files
	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return err
	}
	mux.Handle("/", http.FileServer(http.FS(staticSub)))

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("logcurse web viewer: http://localhost%s\n", addr)
	fmt.Printf("Serving %s (%d lines)\n", sourceFile, idx.TotalLines)
	return http.ListenAndServe(addr, mux)
}

// indexCache provides lazy, thread-safe caching of LineIndex instances.
type indexCache struct {
	mu      sync.Mutex
	entries map[string]*fileutil.LineIndex
}

func newIndexCache() *indexCache {
	return &indexCache{entries: make(map[string]*fileutil.LineIndex)}
}

func (c *indexCache) Get(path string) (*fileutil.LineIndex, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if idx, ok := c.entries[path]; ok {
		return idx, nil
	}

	idx, err := fileutil.BuildLineIndex(path, fileutil.DefaultIndexInterval)
	if err != nil {
		return nil, err
	}
	c.entries[path] = idx
	return idx, nil
}

// ServeDirectory starts the web viewer in directory mode.
func ServeDirectory(dir string, port int, version string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolving directory: %w", err)
	}

	cache := newIndexCache()
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/files", filesHandler(absDir))
	mux.HandleFunc("/api/lines", dirLinesHandler(absDir, cache))
	mux.HandleFunc("/api/comments", dirCommentsHandler(absDir))
	mux.HandleFunc("/api/download", dirDownloadHandler(absDir))
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(version))
	})

	// Static viewer files under /view/
	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return err
	}
	mux.Handle("/view/", http.StripPrefix("/view/", http.FileServer(http.FS(staticSub))))

	// Directory listing page at /
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		content, err := staticFS.ReadFile("static/directory.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write(content)
	})

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("logcurse web viewer: http://localhost%s\n", addr)
	fmt.Printf("Serving directory %s\n", absDir)
	return http.ListenAndServe(addr, mux)
}
