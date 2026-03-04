package editor

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/wasson-ece/logcurse/internal/fileutil"
	"github.com/wasson-ece/logcurse/internal/model"
)

// ParseRange parses a sed-style range like "140,160p" or a single line like "42" into start and end line numbers.
func ParseRange(rangeStr string) (int, int, error) {
	// Try range format first: "140,160" or "140,160p"
	re := regexp.MustCompile(`^(\d+),(\d+)p?$`)
	if m := re.FindStringSubmatch(rangeStr); m != nil {
		start, err := strconv.Atoi(m[1])
		if err != nil {
			return 0, 0, err
		}
		end, err := strconv.Atoi(m[2])
		if err != nil {
			return 0, 0, err
		}
		if start < 1 || end < start {
			return 0, 0, fmt.Errorf("invalid range: start=%d end=%d", start, end)
		}
		return start, end, nil
	}

	// Try single line: "42" or "42p"
	reSingle := regexp.MustCompile(`^(\d+)p?$`)
	if m := reSingle.FindStringSubmatch(rangeStr); m != nil {
		line, err := strconv.Atoi(m[1])
		if err != nil {
			return 0, 0, err
		}
		if line < 1 {
			return 0, 0, fmt.Errorf("invalid line number: %d", line)
		}
		return line, line, nil
	}

	return 0, 0, fmt.Errorf("invalid range %q: expected format like '42', '42p', or '140,160p'", rangeStr)
}

// defaultEditor returns a platform-appropriate default editor.
func defaultEditor() string {
	if e := os.Getenv("VISUAL"); e != "" {
		return e
	}
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	switch runtime.GOOS {
	case "windows":
		return "notepad"
	case "darwin":
		return "nano"
	default:
		return "nano"
	}
}

// buildTempFile creates a temp file with context lines prefixed by # and instructions.
func buildTempFile(sourceFile string, start, end int) (string, error) {
	lines, err := fileutil.ReadLines(sourceFile, start, end)
	if err != nil {
		return "", fmt.Errorf("reading source lines: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "logcurse-*.txt")
	if err != nil {
		return "", err
	}

	fmt.Fprintf(tmpFile, "# logcurse comment for %s lines %d-%d\n", sourceFile, start, end)
	fmt.Fprintf(tmpFile, "# Lines starting with # are ignored. Write your comment below.\n")
	fmt.Fprintf(tmpFile, "# --- Context (lines %d-%d) ---\n", start, end)
	for i, line := range lines {
		fmt.Fprintf(tmpFile, "# %d: %s\n", start+i, line)
	}
	fmt.Fprintf(tmpFile, "# --- End context ---\n")
	fmt.Fprintf(tmpFile, "\n")

	if err := tmpFile.Close(); err != nil {
		return "", err
	}
	return tmpFile.Name(), nil
}

// extractComment reads the temp file and returns non-comment, non-empty content.
func extractComment(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var commentLines []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "#") {
			continue
		}
		commentLines = append(commentLines, line)
	}

	body := strings.TrimSpace(strings.Join(commentLines, "\n"))
	return body, nil
}

// LaunchEditor opens the user's editor for a comment on the given range, and returns the comment body.
func LaunchEditor(sourceFile string, start, end int) (string, error) {
	tmpPath, err := buildTempFile(sourceFile, start, end)
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpPath)

	editor := defaultEditor()
	cmd := exec.Command(editor, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor failed: %w", err)
	}

	return extractComment(tmpPath)
}

// AddComment is the full workflow: parse range, launch editor, save to YAML sidecar.
func AddComment(sourceFile, rangeStr string) error {
	start, end, err := ParseRange(rangeStr)
	if err != nil {
		return err
	}

	// Validate range against file
	lineCount, err := fileutil.CountLines(sourceFile)
	if err != nil {
		return fmt.Errorf("reading %s: %w", sourceFile, err)
	}
	if end > lineCount {
		return fmt.Errorf("range end %d exceeds file length %d", end, lineCount)
	}

	body, err := LaunchEditor(sourceFile, start, end)
	if err != nil {
		return err
	}
	if body == "" {
		fmt.Println("Empty comment, nothing saved.")
		return nil
	}

	// Load or create sidecar
	sidecarPath := model.SidecarPath(sourceFile)
	var cf *model.CommentFile
	if _, err := os.Stat(sidecarPath); err == nil {
		cf, err = model.Load(sidecarPath)
		if err != nil {
			return fmt.Errorf("loading sidecar: %w", err)
		}
	} else {
		cf = &model.CommentFile{
			Version:    1,
			SourceFile: sourceFile,
		}
	}

	comment, err := model.NewComment(sourceFile, start, end, body)
	if err != nil {
		return err
	}
	comment.ID = model.NextID(cf)
	cf.Comments = append(cf.Comments, *comment)

	if err := model.Save(sidecarPath, cf); err != nil {
		return fmt.Errorf("saving sidecar: %w", err)
	}

	fmt.Printf("Comment %s saved to %s (lines %d-%d)\n", comment.ID, sidecarPath, start, end)
	return nil
}
