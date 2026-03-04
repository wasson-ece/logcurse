package model

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	ymlPath := filepath.Join(dir, "test.yml")

	cf := &CommentFile{
		Version:    1,
		SourceFile: "test.txt",
		Comments: []Comment{
			{
				ID:          "c1",
				Range:       Range{Start: 1, End: 5},
				ContentHash: "sha256:abc123",
				Body:        "This is a comment\nwith multiple lines\n",
				Created:     "2026-03-04T10:00:00Z",
				Updated:     "2026-03-04T10:00:00Z",
			},
		},
	}

	if err := Save(ymlPath, cf); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(ymlPath)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Version != 1 {
		t.Fatalf("expected version 1, got %d", loaded.Version)
	}
	if len(loaded.Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(loaded.Comments))
	}
	c := loaded.Comments[0]
	if c.ID != "c1" || c.Range.Start != 1 || c.Range.End != 5 {
		t.Fatalf("unexpected comment: %+v", c)
	}
	if c.Body != "This is a comment\nwith multiple lines\n" {
		t.Fatalf("unexpected body: %q", c.Body)
	}
}

func TestNextID(t *testing.T) {
	cf := &CommentFile{
		Comments: []Comment{
			{ID: "c1"},
			{ID: "c3"},
		},
	}
	id := NextID(cf)
	if id != "c4" {
		t.Fatalf("expected c4, got %s", id)
	}
}

func TestNextIDEmpty(t *testing.T) {
	cf := &CommentFile{}
	id := NextID(cf)
	if id != "c1" {
		t.Fatalf("expected c1, got %s", id)
	}
}

func TestLoadAndCheckDrift(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "test.txt")
	ymlPath := filepath.Join(dir, "test.txt.yml")

	// Write source file
	os.WriteFile(srcPath, []byte("line1\nline2\nline3\n"), 0644)

	// Create comment with correct hash
	c, err := NewComment(srcPath, 1, 3, "test comment")
	if err != nil {
		t.Fatal(err)
	}
	c.ID = "c1"

	cf := &CommentFile{
		Version:    1,
		SourceFile: "test.txt",
		Comments:   []Comment{*c},
	}
	if err := Save(ymlPath, cf); err != nil {
		t.Fatal(err)
	}

	// Load without modification — no drift
	loaded, err := LoadAndCheckDrift(ymlPath, srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Comments[0].Drifted {
		t.Fatal("expected no drift")
	}

	// Modify source file — should detect drift
	os.WriteFile(srcPath, []byte("CHANGED\nline2\nline3\n"), 0644)
	loaded, err = LoadAndCheckDrift(ymlPath, srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Comments[0].Drifted {
		t.Fatal("expected drift")
	}
}
