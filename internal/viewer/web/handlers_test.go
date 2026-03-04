package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
