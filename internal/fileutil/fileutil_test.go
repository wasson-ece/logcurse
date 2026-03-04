package fileutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadLines(t *testing.T) {
	path := writeTempFile(t, "line1\nline2\nline3\nline4\nline5\n")

	lines, err := ReadLines(path, 2, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0] != "line2" || lines[1] != "line3" || lines[2] != "line4" {
		t.Fatalf("unexpected lines: %v", lines)
	}
}

func TestReadAllLines(t *testing.T) {
	path := writeTempFile(t, "a\nb\nc\n")
	lines, err := ReadAllLines(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
}

func TestCountLines(t *testing.T) {
	path := writeTempFile(t, "a\nb\nc\nd\ne\n")
	count, err := CountLines(path)
	if err != nil {
		t.Fatal(err)
	}
	if count != 5 {
		t.Fatalf("expected 5, got %d", count)
	}
}

func TestContentHash(t *testing.T) {
	path := writeTempFile(t, "line1\nline2\nline3\n")

	h1, err := ContentHash(path, 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h1, "sha256:") {
		t.Fatalf("expected sha256 prefix, got %s", h1)
	}

	// Same content should produce same hash
	h2, err := ContentHash(path, 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Fatal("same content produced different hashes")
	}

	// Different range should produce different hash
	h3, err := ContentHash(path, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if h1 == h3 {
		t.Fatal("different ranges produced same hash")
	}
}

func TestBuildLineIndexAndRead(t *testing.T) {
	// Build a file with 50 lines
	var sb strings.Builder
	for i := 1; i <= 50; i++ {
		sb.WriteString("line ")
		sb.WriteString(strings.Repeat("x", i))
		sb.WriteString("\n")
	}
	path := writeTempFile(t, sb.String())

	idx, err := BuildLineIndex(path, 10)
	if err != nil {
		t.Fatal(err)
	}
	if idx.TotalLines != 50 {
		t.Fatalf("expected 50 lines, got %d", idx.TotalLines)
	}

	// Read using index
	lines, err := ReadLinesFromIndex(path, idx, 5, 15)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 11 {
		t.Fatalf("expected 11 lines, got %d", len(lines))
	}

	// Compare with direct read
	direct, err := ReadLines(path, 5, 15)
	if err != nil {
		t.Fatal(err)
	}
	for i := range lines {
		if lines[i] != direct[i] {
			t.Fatalf("mismatch at line %d: %q vs %q", i, lines[i], direct[i])
		}
	}
}

func TestLineIndexWriteRead(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 25; i++ {
		sb.WriteString("test line\n")
	}
	path := writeTempFile(t, sb.String())

	idx, err := BuildLineIndex(path, 5)
	if err != nil {
		t.Fatal(err)
	}

	idxPath := filepath.Join(t.TempDir(), "test.idx")
	if err := WriteLineIndex(idxPath, idx); err != nil {
		t.Fatal(err)
	}

	idx2, err := ReadLineIndex(idxPath)
	if err != nil {
		t.Fatal(err)
	}

	if idx.TotalLines != idx2.TotalLines || idx.Interval != idx2.Interval {
		t.Fatal("index mismatch after round-trip")
	}
	if len(idx.Offsets) != len(idx2.Offsets) {
		t.Fatal("offset count mismatch")
	}
	for i := range idx.Offsets {
		if idx.Offsets[i] != idx2.Offsets[i] {
			t.Fatalf("offset %d mismatch: %d vs %d", i, idx.Offsets[i], idx2.Offsets[i])
		}
	}
}
