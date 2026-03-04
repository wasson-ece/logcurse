package editor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseRange(t *testing.T) {
	tests := []struct {
		input     string
		start     int
		end       int
		expectErr bool
	}{
		{"1,10p", 1, 10, false},
		{"140,160p", 140, 160, false},
		{"1,10", 1, 10, false},
		{"5,5p", 5, 5, false},
		{"42", 42, 42, false},
		{"42p", 42, 42, false},
		{"1", 1, 1, false},
		{"abc", 0, 0, true},
		{"0", 0, 0, true},
		{"10,5p", 0, 0, true},
		{"0,5p", 0, 0, true},
	}

	for _, tt := range tests {
		s, e, err := ParseRange(tt.input)
		if tt.expectErr {
			if err == nil {
				t.Errorf("ParseRange(%q) expected error", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseRange(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if s != tt.start || e != tt.end {
			t.Errorf("ParseRange(%q) = (%d,%d), want (%d,%d)", tt.input, s, e, tt.start, tt.end)
		}
	}
}

func TestExtractComment(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "comment.txt")

	content := `# logcurse comment for test.txt lines 1-5
# Lines starting with # are ignored.
# 1: first line
# 2: second line

This is the actual comment.
It has two lines.
`
	os.WriteFile(path, []byte(content), 0644)

	body, err := extractComment(path)
	if err != nil {
		t.Fatal(err)
	}
	expected := "This is the actual comment.\nIt has two lines."
	if body != expected {
		t.Fatalf("expected %q, got %q", expected, body)
	}
}
