package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/wasson-ece/logcurse/internal/fileutil"
	"github.com/wasson-ece/logcurse/internal/model"
)

func createTestFile(t *testing.T, lines int) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= lines; i++ {
		fmt.Fprintf(f, "line %d content\n", i)
	}
	f.Close()
	return path
}

func TestLinesHandler(t *testing.T) {
	path := createTestFile(t, 50)
	idx, err := fileutil.BuildLineIndex(path, fileutil.DefaultIndexInterval)
	if err != nil {
		t.Fatal(err)
	}

	handler := linesHandler(path, idx)

	t.Run("basic range", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/lines?start=1&end=5", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var resp linesResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}
		if len(resp.Lines) != 5 {
			t.Fatalf("expected 5 lines, got %d", len(resp.Lines))
		}
		if resp.Lines[0].Number != 1 {
			t.Fatalf("expected first line number 1, got %d", resp.Lines[0].Number)
		}
		if resp.Lines[4].Number != 5 {
			t.Fatalf("expected last line number 5, got %d", resp.Lines[4].Number)
		}
		if resp.TotalLines != 50 {
			t.Fatalf("expected total 50, got %d", resp.TotalLines)
		}
	})

	t.Run("missing params defaults", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/lines", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var resp linesResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if resp.Start != 1 {
			t.Fatalf("expected start=1, got %d", resp.Start)
		}
	})

	t.Run("end clamped to total", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/lines?start=45&end=999", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		var resp linesResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if resp.End != 50 {
			t.Fatalf("expected end clamped to 50, got %d", resp.End)
		}
		if len(resp.Lines) != 6 {
			t.Fatalf("expected 6 lines (45-50), got %d", len(resp.Lines))
		}
	})
}

func TestCommentsHandler(t *testing.T) {
	t.Run("no sidecar file", func(t *testing.T) {
		path := createTestFile(t, 10)
		handler := commentsHandler(path)

		req := httptest.NewRequest("GET", "/api/comments", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var resp commentsResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if len(resp.Comments) != 0 {
			t.Fatalf("expected 0 comments, got %d", len(resp.Comments))
		}
		if resp.SourceFile != path {
			t.Fatalf("expected source_file=%q, got %q", path, resp.SourceFile)
		}
	})

	t.Run("with sidecar file", func(t *testing.T) {
		path := createTestFile(t, 10)

		comment, err := model.NewComment(path, 1, 3, "test comment")
		if err != nil {
			t.Fatal(err)
		}
		comment.ID = "c1"
		cf := &model.CommentFile{
			Version:    1,
			SourceFile: path,
			Comments:   []model.Comment{*comment},
		}
		if err := model.Save(model.SidecarPath(path), cf); err != nil {
			t.Fatal(err)
		}

		handler := commentsHandler(path)
		req := httptest.NewRequest("GET", "/api/comments", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var resp commentsResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if len(resp.Comments) != 1 {
			t.Fatalf("expected 1 comment, got %d", len(resp.Comments))
		}
		if resp.Comments[0].ID != "c1" {
			t.Fatalf("expected comment id=c1, got %q", resp.Comments[0].ID)
		}
		if resp.Comments[0].Body != "test comment" {
			t.Fatalf("expected body=%q, got %q", "test comment", resp.Comments[0].Body)
		}
	})
}

func TestResolveAndValidate(t *testing.T) {
	dir := t.TempDir()

	t.Run("normal filename", func(t *testing.T) {
		// Create the file so the path is valid
		os.WriteFile(filepath.Join(dir, "test.log"), []byte("hi"), 0644)
		path, err := resolveAndValidate(dir, "test.log")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := filepath.Join(dir, "test.log")
		absExpected, _ := filepath.Abs(expected)
		if path != absExpected {
			t.Fatalf("expected %q, got %q", absExpected, path)
		}
	})

	t.Run("empty filename", func(t *testing.T) {
		_, err := resolveAndValidate(dir, "")
		if err == nil {
			t.Fatal("expected error for empty filename")
		}
	})

	t.Run("path traversal with slash", func(t *testing.T) {
		_, err := resolveAndValidate(dir, "../etc/passwd")
		if err == nil {
			t.Fatal("expected error for path traversal")
		}
	})

	t.Run("path traversal with backslash", func(t *testing.T) {
		_, err := resolveAndValidate(dir, "..\\etc\\passwd")
		if err == nil {
			t.Fatal("expected error for backslash path traversal")
		}
	})

	t.Run("subdirectory attempt", func(t *testing.T) {
		_, err := resolveAndValidate(dir, "sub/file.log")
		if err == nil {
			t.Fatal("expected error for subdirectory path")
		}
	})
}

func TestFilesHandler(t *testing.T) {
	dir := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(dir, "app.log"), []byte("line1\nline2\n"), 0644)
	os.WriteFile(filepath.Join(dir, "error.log"), []byte("err\n"), 0644)

	// Create an annotated file with sidecar
	os.WriteFile(filepath.Join(dir, "server.log"), []byte("line1\nline2\nline3\n"), 0644)
	serverPath := filepath.Join(dir, "server.log")
	comment, _ := model.NewComment(serverPath, 1, 2, "test note")
	comment.ID = "c1"
	cf := &model.CommentFile{Version: 1, SourceFile: serverPath, Comments: []model.Comment{*comment}}
	model.Save(filepath.Join(dir, "server.log.yml"), cf)

	// Create a subdirectory (should be excluded)
	os.Mkdir(filepath.Join(dir, "subdir"), 0755)

	handler := filesHandler(dir)
	req := httptest.NewRequest("GET", "/api/files", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var files []fileEntry
	if err := json.NewDecoder(w.Body).Decode(&files); err != nil {
		t.Fatal(err)
	}

	// Should have 3 files (app.log, error.log, server.log) -- yml excluded, subdir excluded
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d: %+v", len(files), files)
	}

	// Find server.log and check annotation
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	found := false
	for _, f := range files {
		if f.Name == "server.log" {
			found = true
			if !f.Annotated {
				t.Fatal("expected server.log to be annotated")
			}
			if f.CommentCount != 1 {
				t.Fatalf("expected 1 comment, got %d", f.CommentCount)
			}
		}
	}
	if !found {
		t.Fatal("server.log not in file list")
	}
}

func TestDirLinesHandler(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.log"), []byte("alpha\nbeta\ngamma\n"), 0644)

	cache := newIndexCache()
	handler := dirLinesHandler(dir, cache)

	req := httptest.NewRequest("GET", "/api/lines?file=test.log&start=1&end=2", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp linesResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(resp.Lines))
	}
	if resp.Lines[0].Text != "alpha" {
		t.Fatalf("expected 'alpha', got %q", resp.Lines[0].Text)
	}
}

func TestDirLinesHandler_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	cache := newIndexCache()
	handler := dirLinesHandler(dir, cache)

	tests := []struct {
		name string
		file string
	}{
		{"dot-dot slash", "../etc/passwd"},
		{"backslash", "..\\secret"},
		{"nested slash", "sub/file.log"},
		{"empty", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/lines?file="+tc.file+"&start=1&end=1", nil)
			w := httptest.NewRecorder()
			handler(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 for file=%q, got %d", tc.file, w.Code)
			}
		})
	}
}

func TestDirCommentsHandler(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")
	os.WriteFile(logPath, []byte("line1\nline2\nline3\n"), 0644)

	// Create sidecar
	comment, _ := model.NewComment(logPath, 1, 2, "dir comment")
	comment.ID = "c1"
	cf := &model.CommentFile{Version: 1, SourceFile: logPath, Comments: []model.Comment{*comment}}
	model.Save(filepath.Join(dir, "test.log.yml"), cf)

	handler := dirCommentsHandler(dir)
	req := httptest.NewRequest("GET", "/api/comments?file=test.log", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp commentsResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(resp.Comments))
	}
	if resp.Comments[0].Body != "dir comment" {
		t.Fatalf("expected body=%q, got %q", "dir comment", resp.Comments[0].Body)
	}
}
