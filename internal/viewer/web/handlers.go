package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

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

func downloadCommentsHandler(sourceFile string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ymlPath := model.SidecarPath(sourceFile)
		if _, err := os.Stat(ymlPath); err != nil {
			http.Error(w, "no comments file found", http.StatusNotFound)
			return
		}
		name := filepath.Base(ymlPath)
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, name))
		http.ServeFile(w, r, ymlPath)
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

func dirDownloadCommentsHandler(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := r.URL.Query().Get("file")
		fullPath, err := resolveAndValidate(dir, filename)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ymlPath := model.SidecarPath(fullPath)
		if _, err := os.Stat(ymlPath); err != nil {
			http.Error(w, "no comments file found", http.StatusNotFound)
			return
		}
		ymlName := filepath.Base(ymlPath)
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, ymlName))
		http.ServeFile(w, r, ymlPath)
	}
}

// configHandler returns the server configuration as JSON.
func configHandler(rw bool, version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"rw":      rw,
			"version": version,
		})
	}
}

type createCommentRequest struct {
	RangeStart int    `json:"range_start"`
	RangeEnd   int    `json:"range_end"`
	Body       string `json:"body"`
	Author     string `json:"author"`
}

// generateID creates a comment ID. If author is empty, returns w<unix_timestamp>.
// If author is set, scans existing comments for <authorLower><N> and returns max+1.
func generateID(cf *model.CommentFile, author string) string {
	if author == "" {
		return fmt.Sprintf("w%d", time.Now().Unix())
	}
	prefix := strings.ToLower(author)
	re := regexp.MustCompile(`^` + regexp.QuoteMeta(prefix) + `(\d+)$`)
	max := 0
	for _, c := range cf.Comments {
		if m := re.FindStringSubmatch(c.ID); m != nil {
			n, _ := strconv.Atoi(m[1])
			if n > max {
				max = n
			}
		}
	}
	return fmt.Sprintf("%s%d", prefix, max+1)
}

func createCommentHandler(sourceFile string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req createCommentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		if req.Body == "" {
			http.Error(w, "body is required", http.StatusBadRequest)
			return
		}

		totalLines, err := fileutil.CountLines(sourceFile)
		if err != nil {
			http.Error(w, "counting lines: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if req.RangeStart < 1 || req.RangeEnd < req.RangeStart || req.RangeEnd > totalLines {
			http.Error(w, fmt.Sprintf("invalid range: must be 1-%d", totalLines), http.StatusBadRequest)
			return
		}

		ymlPath := model.SidecarPath(sourceFile)
		var cf *model.CommentFile
		if _, err := os.Stat(ymlPath); err == nil {
			cf, err = model.Load(ymlPath)
			if err != nil {
				http.Error(w, "loading comments: "+err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			cf = &model.CommentFile{
				Version:    1,
				SourceFile: sourceFile,
				Comments:   []model.Comment{},
			}
		}

		comment, err := model.NewComment(sourceFile, req.RangeStart, req.RangeEnd, req.Body)
		if err != nil {
			http.Error(w, "creating comment: "+err.Error(), http.StatusInternalServerError)
			return
		}
		comment.ID = generateID(cf, req.Author)
		comment.Author = req.Author

		// Insert in sorted order by range start
		insertIdx := len(cf.Comments)
		for i, c := range cf.Comments {
			if req.RangeStart < c.Range.Start {
				insertIdx = i
				break
			}
		}
		cf.Comments = append(cf.Comments, model.Comment{})
		copy(cf.Comments[insertIdx+1:], cf.Comments[insertIdx:])
		cf.Comments[insertIdx] = *comment

		if err := model.Save(ymlPath, cf); err != nil {
			http.Error(w, "saving: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(commentResponse{
			ID:          comment.ID,
			RangeStart:  comment.Range.Start,
			RangeEnd:    comment.Range.End,
			Body:        comment.Body,
			Author:      comment.Author,
			Created:     comment.Created,
			Updated:     comment.Updated,
			ContentHash: comment.ContentHash,
		})
	}
}

type updateCommentRequest struct {
	ID   string `json:"id"`
	Body string `json:"body"`
}

func updateCommentHandler(sourceFile string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req updateCommentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		if req.ID == "" || req.Body == "" {
			http.Error(w, "id and body are required", http.StatusBadRequest)
			return
		}

		ymlPath := model.SidecarPath(sourceFile)
		cf, err := model.Load(ymlPath)
		if err != nil {
			http.Error(w, "loading comments: "+err.Error(), http.StatusInternalServerError)
			return
		}

		idx := -1
		for i, c := range cf.Comments {
			if c.ID == req.ID {
				idx = i
				break
			}
		}
		if idx == -1 {
			http.Error(w, "comment not found", http.StatusNotFound)
			return
		}

		cf.Comments[idx].Body = req.Body
		cf.Comments[idx].Updated = time.Now().UTC().Format(time.RFC3339)

		hash, err := fileutil.ContentHash(sourceFile, cf.Comments[idx].Range.Start, cf.Comments[idx].Range.End)
		if err == nil {
			cf.Comments[idx].ContentHash = hash
		}

		if err := model.Save(ymlPath, cf); err != nil {
			http.Error(w, "saving: "+err.Error(), http.StatusInternalServerError)
			return
		}

		c := cf.Comments[idx]
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(commentResponse{
			ID:          c.ID,
			RangeStart:  c.Range.Start,
			RangeEnd:    c.Range.End,
			Body:        c.Body,
			Author:      c.Author,
			Created:     c.Created,
			Updated:     c.Updated,
			ContentHash: c.ContentHash,
		})
	}
}

func deleteCommentHandler(sourceFile string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}

		ymlPath := model.SidecarPath(sourceFile)
		cf, err := model.Load(ymlPath)
		if err != nil {
			http.Error(w, "loading comments: "+err.Error(), http.StatusInternalServerError)
			return
		}

		idx := -1
		for i, c := range cf.Comments {
			if c.ID == id {
				idx = i
				break
			}
		}
		if idx == -1 {
			http.Error(w, "comment not found", http.StatusNotFound)
			return
		}

		cf.Comments = append(cf.Comments[:idx], cf.Comments[idx+1:]...)
		if err := model.Save(ymlPath, cf); err != nil {
			http.Error(w, "saving: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
	}
}

// Directory-mode wrappers for write handlers

func dirCreateCommentHandler(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := r.URL.Query().Get("file")
		fullPath, err := resolveAndValidate(dir, filename)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		createCommentHandler(fullPath)(w, r)
	}
}

func dirUpdateCommentHandler(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := r.URL.Query().Get("file")
		fullPath, err := resolveAndValidate(dir, filename)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		updateCommentHandler(fullPath)(w, r)
	}
}

func dirDeleteCommentHandler(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := r.URL.Query().Get("file")
		fullPath, err := resolveAndValidate(dir, filename)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		deleteCommentHandler(fullPath)(w, r)
	}
}
