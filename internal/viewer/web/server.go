package web

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"

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
