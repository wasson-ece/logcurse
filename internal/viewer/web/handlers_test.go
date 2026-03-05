package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

func TestConfigHandler(t *testing.T) {
	t.Run("rw true", func(t *testing.T) {
		handler := configHandler(true, "v1.0.0")
		req := httptest.NewRequest("GET", "/api/config", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var resp map[string]interface{}
		json.NewDecoder(w.Body).Decode(&resp)
		if resp["rw"] != true {
			t.Fatalf("expected rw=true, got %v", resp["rw"])
		}
		if resp["version"] != "v1.0.0" {
			t.Fatalf("expected version=v1.0.0, got %v", resp["version"])
		}
	})

	t.Run("rw false", func(t *testing.T) {
		handler := configHandler(false, "dev")
		req := httptest.NewRequest("GET", "/api/config", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		var resp map[string]interface{}
		json.NewDecoder(w.Body).Decode(&resp)
		if resp["rw"] != false {
			t.Fatalf("expected rw=false, got %v", resp["rw"])
		}
	})
}

func TestCreateCommentHandler(t *testing.T) {
	t.Run("valid create without author", func(t *testing.T) {
		path := createTestFile(t, 10)
		handler := createCommentHandler(path)

		body := `{"range_start":1,"range_end":3,"body":"hello world"}`
		req := httptest.NewRequest("POST", "/api/comments/create", strings.NewReader(body))
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var resp commentResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if resp.Body != "hello world" {
			t.Fatalf("expected body='hello world', got %q", resp.Body)
		}
		if !strings.HasPrefix(resp.ID, "w") {
			t.Fatalf("expected ID starting with 'w', got %q", resp.ID)
		}

		// Verify sidecar file
		cf, err := model.Load(model.SidecarPath(path))
		if err != nil {
			t.Fatal(err)
		}
		if len(cf.Comments) != 1 {
			t.Fatalf("expected 1 comment in sidecar, got %d", len(cf.Comments))
		}
	})

	t.Run("valid create with author", func(t *testing.T) {
		path := createTestFile(t, 10)
		handler := createCommentHandler(path)

		body := `{"range_start":1,"range_end":2,"body":"first","author":"chip"}`
		req := httptest.NewRequest("POST", "/api/comments/create", strings.NewReader(body))
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var resp commentResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if resp.ID != "chip1" {
			t.Fatalf("expected ID='chip1', got %q", resp.ID)
		}

		// Add a second comment by same author
		body = `{"range_start":3,"range_end":4,"body":"second","author":"chip"}`
		req = httptest.NewRequest("POST", "/api/comments/create", strings.NewReader(body))
		w = httptest.NewRecorder()
		handler(w, req)

		json.NewDecoder(w.Body).Decode(&resp)
		if resp.ID != "chip2" {
			t.Fatalf("expected ID='chip2', got %q", resp.ID)
		}
	})

	t.Run("inserts in sorted order by range_start", func(t *testing.T) {
		path := createTestFile(t, 10)
		handler := createCommentHandler(path)

		// Create comment on lines 5-6 first
		body := `{"range_start":5,"range_end":6,"body":"middle","author":"a"}`
		req := httptest.NewRequest("POST", "/api/comments/create", strings.NewReader(body))
		w := httptest.NewRecorder()
		handler(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", w.Code)
		}

		// Create comment on lines 1-2 (should be inserted before)
		body = `{"range_start":1,"range_end":2,"body":"first","author":"a"}`
		req = httptest.NewRequest("POST", "/api/comments/create", strings.NewReader(body))
		w = httptest.NewRecorder()
		handler(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", w.Code)
		}

		// Create comment on lines 8-9 (should be appended)
		body = `{"range_start":8,"range_end":9,"body":"last","author":"a"}`
		req = httptest.NewRequest("POST", "/api/comments/create", strings.NewReader(body))
		w = httptest.NewRecorder()
		handler(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", w.Code)
		}

		cf, err := model.Load(model.SidecarPath(path))
		if err != nil {
			t.Fatal(err)
		}
		if len(cf.Comments) != 3 {
			t.Fatalf("expected 3 comments, got %d", len(cf.Comments))
		}
		if cf.Comments[0].Range.Start != 1 {
			t.Fatalf("expected first comment at line 1, got %d", cf.Comments[0].Range.Start)
		}
		if cf.Comments[1].Range.Start != 5 {
			t.Fatalf("expected second comment at line 5, got %d", cf.Comments[1].Range.Start)
		}
		if cf.Comments[2].Range.Start != 8 {
			t.Fatalf("expected third comment at line 8, got %d", cf.Comments[2].Range.Start)
		}
	})

	t.Run("invalid range", func(t *testing.T) {
		path := createTestFile(t, 10)
		handler := createCommentHandler(path)

		body := `{"range_start":0,"range_end":3,"body":"bad"}`
		req := httptest.NewRequest("POST", "/api/comments/create", strings.NewReader(body))
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("range exceeds file", func(t *testing.T) {
		path := createTestFile(t, 10)
		handler := createCommentHandler(path)

		body := `{"range_start":1,"range_end":100,"body":"bad"}`
		req := httptest.NewRequest("POST", "/api/comments/create", strings.NewReader(body))
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("empty body", func(t *testing.T) {
		path := createTestFile(t, 10)
		handler := createCommentHandler(path)

		body := `{"range_start":1,"range_end":3,"body":""}`
		req := httptest.NewRequest("POST", "/api/comments/create", strings.NewReader(body))
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})
}

func TestUpdateCommentHandler(t *testing.T) {
	t.Run("valid update", func(t *testing.T) {
		path := createTestFile(t, 10)

		// Create a comment first
		comment, _ := model.NewComment(path, 1, 3, "original")
		comment.ID = "c1"
		cf := &model.CommentFile{Version: 1, SourceFile: path, Comments: []model.Comment{*comment}}
		model.Save(model.SidecarPath(path), cf)

		handler := updateCommentHandler(path)
		body := `{"id":"c1","body":"updated body"}`
		req := httptest.NewRequest("PUT", "/api/comments/update", strings.NewReader(body))
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp commentResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if resp.Body != "updated body" {
			t.Fatalf("expected body='updated body', got %q", resp.Body)
		}

		// Verify sidecar
		cf, _ = model.Load(model.SidecarPath(path))
		if cf.Comments[0].Body != "updated body" {
			t.Fatalf("sidecar not updated: %q", cf.Comments[0].Body)
		}
	})

	t.Run("not found", func(t *testing.T) {
		path := createTestFile(t, 10)
		comment, _ := model.NewComment(path, 1, 3, "original")
		comment.ID = "c1"
		cf := &model.CommentFile{Version: 1, SourceFile: path, Comments: []model.Comment{*comment}}
		model.Save(model.SidecarPath(path), cf)

		handler := updateCommentHandler(path)
		body := `{"id":"nonexistent","body":"updated"}`
		req := httptest.NewRequest("PUT", "/api/comments/update", strings.NewReader(body))
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestDeleteCommentHandler(t *testing.T) {
	t.Run("valid delete", func(t *testing.T) {
		path := createTestFile(t, 10)

		comment, _ := model.NewComment(path, 1, 3, "to delete")
		comment.ID = "c1"
		cf := &model.CommentFile{Version: 1, SourceFile: path, Comments: []model.Comment{*comment}}
		model.Save(model.SidecarPath(path), cf)

		handler := deleteCommentHandler(path)
		req := httptest.NewRequest("DELETE", "/api/comments/delete?id=c1", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		// Verify removed
		cf, _ = model.Load(model.SidecarPath(path))
		if len(cf.Comments) != 0 {
			t.Fatalf("expected 0 comments after delete, got %d", len(cf.Comments))
		}
	})

	t.Run("not found", func(t *testing.T) {
		path := createTestFile(t, 10)
		comment, _ := model.NewComment(path, 1, 3, "keep")
		comment.ID = "c1"
		cf := &model.CommentFile{Version: 1, SourceFile: path, Comments: []model.Comment{*comment}}
		model.Save(model.SidecarPath(path), cf)

		handler := deleteCommentHandler(path)
		req := httptest.NewRequest("DELETE", "/api/comments/delete?id=nonexistent", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestDirCreateCommentHandler(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")
	os.WriteFile(logPath, []byte("line1\nline2\nline3\n"), 0644)

	handler := dirCreateCommentHandler(dir)
	body := `{"range_start":1,"range_end":2,"body":"dir create","author":"web"}`
	req := httptest.NewRequest("POST", "/api/comments/create?file=test.log", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp commentResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Body != "dir create" {
		t.Fatalf("expected body='dir create', got %q", resp.Body)
	}
}

func TestGenerateID(t *testing.T) {
	t.Run("without author", func(t *testing.T) {
		cf := &model.CommentFile{}
		id := generateID(cf, "")
		if !strings.HasPrefix(id, "w") {
			t.Fatalf("expected ID starting with 'w', got %q", id)
		}
	})

	t.Run("with author first comment", func(t *testing.T) {
		cf := &model.CommentFile{}
		id := generateID(cf, "alice")
		if id != "alice1" {
			t.Fatalf("expected 'alice1', got %q", id)
		}
	})

	t.Run("with author increments", func(t *testing.T) {
		cf := &model.CommentFile{
			Comments: []model.Comment{
				{ID: "alice1"},
				{ID: "alice2"},
				{ID: "bob1"},
			},
		}
		id := generateID(cf, "alice")
		if id != "alice3" {
			t.Fatalf("expected 'alice3', got %q", id)
		}
	})
}
