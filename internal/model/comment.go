package model

import (
	"fmt"
	"os"
	"time"

	"github.com/wasson-ece/logcurse/internal/fileutil"
	"gopkg.in/yaml.v3"
)

// Range represents a 1-indexed inclusive line range.
type Range struct {
	Start int `yaml:"start"`
	End   int `yaml:"end"`
}

// Comment represents a single annotation on a line range.
type Comment struct {
	ID          string `yaml:"id"`
	Range       Range  `yaml:"range"`
	ContentHash string `yaml:"content_hash"`
	Author      string `yaml:"author"`
	Created     string `yaml:"created"`
	Updated     string `yaml:"updated"`
	Body        string `yaml:"body"`
	Drifted     bool   `yaml:"-"` // set at load time, not persisted
}

// CommentFile is the top-level YAML structure for a sidecar file.
type CommentFile struct {
	Version    int       `yaml:"version"`
	SourceFile string    `yaml:"source_file"`
	Comments   []Comment `yaml:"comments"`
}

// Load reads and parses a YAML comment file.
func Load(path string) (*CommentFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cf CommentFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &cf, nil
}

// Save writes the comment file to disk as YAML.
func Save(path string, cf *CommentFile) error {
	data, err := yaml.Marshal(cf)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadAndCheckDrift loads a comment file and checks each comment's hash against the source.
func LoadAndCheckDrift(ymlPath, sourceFilePath string) (*CommentFile, error) {
	cf, err := Load(ymlPath)
	if err != nil {
		return nil, err
	}

	for i := range cf.Comments {
		c := &cf.Comments[i]
		currentHash, err := fileutil.ContentHash(sourceFilePath, c.Range.Start, c.Range.End)
		if err != nil {
			// If we can't compute hash (e.g. file shorter), mark as drifted
			c.Drifted = true
			continue
		}
		if currentHash != c.ContentHash {
			c.Drifted = true
		}
	}
	return cf, nil
}

// SidecarPath returns the expected YAML sidecar path for a source file.
func SidecarPath(sourceFile string) string {
	return sourceFile + ".yml"
}

// NextID returns a simple sequential ID like "c1", "c2", etc.
func NextID(cf *CommentFile) string {
	max := 0
	for _, c := range cf.Comments {
		var n int
		fmt.Sscanf(c.ID, "c%d", &n)
		if n > max {
			max = n
		}
	}
	return fmt.Sprintf("c%d", max+1)
}

// NewComment creates a comment with timestamps and hash computed from the source file.
func NewComment(sourceFile string, start, end int, body string) (*Comment, error) {
	hash, err := fileutil.ContentHash(sourceFile, start, end)
	if err != nil {
		return nil, fmt.Errorf("computing hash: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	return &Comment{
		Range:       Range{Start: start, End: end},
		ContentHash: hash,
		Created:     now,
		Updated:     now,
		Body:        body,
	}, nil
}
