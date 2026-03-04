package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wasson-ece/logcurse/internal/fileutil"
	"github.com/wasson-ece/logcurse/internal/model"
)

type linesResponse struct {
	Lines      []numberedLine `json:"lines"`
	TotalLines int            `json:"total_lines"`
	Start      int            `json:"start"`
	End        int            `json:"end"`
}

type numberedLine struct {
	Number int    `json:"number"`
	Text   string `json:"text"`
}

type commentResponse struct {
	ID          string `json:"id"`
	RangeStart  int    `json:"range_start"`
	RangeEnd    int    `json:"range_end"`
	Body        string `json:"body"`
	Author      string `json:"author"`
	Created     string `json:"created"`
	Updated     string `json:"updated"`
	ContentHash string `json:"content_hash"`
	Drifted     bool   `json:"drifted"`
}

type commentsResponse struct {
	Comments   []commentResponse `json:"comments"`
	SourceFile string            `json:"source_file"`
}

func linesHandler(sourceFile string, idx *fileutil.LineIndex) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start, _ := strconv.Atoi(r.URL.Query().Get("start"))
		end, _ := strconv.Atoi(r.URL.Query().Get("end"))

		if start < 1 {
			start = 1
		}
		if end < start {
			end = start + 99
		}
		if end > idx.TotalLines {
			end = idx.TotalLines
		}

		lines, err := fileutil.ReadLinesFromIndex(sourceFile, idx, start, end)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		numbered := make([]numberedLine, len(lines))
		for i, l := range lines {
			numbered[i] = numberedLine{Number: start + i, Text: l}
		}

		resp := linesResponse{
			Lines:      numbered,
			TotalLines: idx.TotalLines,
			Start:      start,
			End:        end,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func commentsHandler(sourceFile string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ymlPath := model.SidecarPath(sourceFile)

		resp := commentsResponse{
			SourceFile: sourceFile,
			Comments:   []commentResponse{},
		}

		if _, err := os.Stat(ymlPath); err == nil {
			cf, err := model.LoadAndCheckDrift(ymlPath, sourceFile)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			for _, c := range cf.Comments {
				resp.Comments = append(resp.Comments, commentResponse{
					ID:          c.ID,
					RangeStart:  c.Range.Start,
					RangeEnd:    c.Range.End,
					Body:        c.Body,
					Author:      c.Author,
					Created:     c.Created,
					Updated:     c.Updated,
					ContentHash: c.ContentHash,
					Drifted:     c.Drifted,
				})
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// resolveAndValidate ensures filename is a bare name (no path separators)
// and resolves it within baseDir, verifying the result stays inside baseDir.
func resolveAndValidate(baseDir, filename string) (string, error) {
	if filename == "" {
		return "", fmt.Errorf("filename is required")
	}
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return "", fmt.Errorf("invalid filename: must not contain path separators")
	}
	resolved := filepath.Join(baseDir, filename)
	abs, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("invalid base path: %w", err)
	}
	if !strings.HasPrefix(abs, absBase+string(filepath.Separator)) {
		return "", fmt.Errorf("path traversal detected")
	}
	return abs, nil
}

type fileEntry struct {
	Name         string `json:"name"`
	Size         int64  `json:"size"`
	Annotated    bool   `json:"annotated"`
	CommentCount int    `json:"comment_count"`
}

func filesHandler(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var files []fileEntry
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if strings.HasSuffix(name, ".yml") {
				continue
			}

			info, err := e.Info()
			if err != nil {
				continue
			}

			fe := fileEntry{
				Name: name,
				Size: info.Size(),
			}

			sidecar := filepath.Join(dir, model.SidecarPath(name))
			if _, serr := os.Stat(sidecar); serr == nil {
				fe.Annotated = true
				if cf, lerr := model.Load(sidecar); lerr == nil {
					fe.CommentCount = len(cf.Comments)
				}
			}

			files = append(files, fe)
		}

		if files == nil {
			files = []fileEntry{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(files)
	}
}

func dirLinesHandler(dir string, cache *indexCache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := r.URL.Query().Get("file")
		fullPath, err := resolveAndValidate(dir, filename)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		idx, err := cache.Get(fullPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		linesHandler(fullPath, idx)(w, r)
	}
}

func dirCommentsHandler(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := r.URL.Query().Get("file")
		fullPath, err := resolveAndValidate(dir, filename)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		commentsHandler(fullPath)(w, r)
	}
}

func downloadHandler(sourceFile string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := filepath.Base(sourceFile)
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, name))
		http.ServeFile(w, r, sourceFile)
	}
}

func dirDownloadHandler(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := r.URL.Query().Get("file")
		fullPath, err := resolveAndValidate(dir, filename)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, filename))
		http.ServeFile(w, r, fullPath)
	}
}
