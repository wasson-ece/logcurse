package fileutil

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// maxScanBuf is the maximum line length the scanner will handle (1 MB).
const maxScanBuf = 1024 * 1024

// ReadLines reads lines [start, end] (1-indexed, inclusive) from a file.
func ReadLines(path string, start, end int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, maxScanBuf), maxScanBuf)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum > end {
			break
		}
		if lineNum >= start {
			lines = append(lines, scanner.Text())
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

// ReadAllLines reads every line from a file.
func ReadAllLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, maxScanBuf), maxScanBuf)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

// CountLines counts the total lines in a file.
func CountLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, maxScanBuf), maxScanBuf)
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}

// ContentHash returns "sha256:<hex>" of the raw bytes of lines [start, end] (1-indexed, inclusive).
func ContentHash(path string, start, end int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, maxScanBuf), maxScanBuf)
	lineNum := 0
	first := true
	for scanner.Scan() {
		lineNum++
		if lineNum > end {
			break
		}
		if lineNum >= start {
			if !first {
				h.Write([]byte("\n"))
			}
			h.Write(scanner.Bytes())
			first = false
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}

// LineIndex stores byte offsets at regular intervals for O(1) seeking into large files.
type LineIndex struct {
	TotalLines int
	Interval   int      // every N lines
	Offsets    []int64  // byte offset at line 1, 1+interval, 1+2*interval, ...
}

const DefaultIndexInterval = 1000

// BuildLineIndex scans a file and records byte offsets every `interval` lines.
func BuildLineIndex(path string, interval int) (*LineIndex, error) {
	if interval <= 0 {
		interval = DefaultIndexInterval
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	idx := &LineIndex{Interval: interval}
	idx.Offsets = append(idx.Offsets, 0) // line 1 starts at offset 0

	reader := bufio.NewReader(f)
	var offset int64
	lineNum := 0

	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			lineNum++
			offset += int64(len(line))
			if lineNum%interval == 0 {
				idx.Offsets = append(idx.Offsets, offset)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	idx.TotalLines = lineNum
	return idx, nil
}

// ReadLinesFromIndex reads lines [start, end] using a prebuilt index for fast seeking.
func ReadLinesFromIndex(path string, idx *LineIndex, start, end int) ([]string, error) {
	if start < 1 {
		start = 1
	}
	if end > idx.TotalLines {
		end = idx.TotalLines
	}
	if start > end {
		return nil, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Find the nearest indexed position before start
	chunkIdx := (start - 1) / idx.Interval
	if chunkIdx >= len(idx.Offsets) {
		chunkIdx = len(idx.Offsets) - 1
	}
	seekOffset := idx.Offsets[chunkIdx]
	startLine := chunkIdx*idx.Interval + 1

	if _, err := f.Seek(seekOffset, io.SeekStart); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, maxScanBuf), maxScanBuf)
	lineNum := startLine - 1
	var lines []string

	for scanner.Scan() {
		lineNum++
		if lineNum > end {
			break
		}
		if lineNum >= start {
			lines = append(lines, scanner.Text())
		}
	}
	return lines, scanner.Err()
}

// WriteLineIndex writes an index to a binary file for caching.
func WriteLineIndex(path string, idx *LineIndex) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	// Header: totalLines(int64), interval(int64), count(int64)
	binary.Write(w, binary.LittleEndian, int64(idx.TotalLines))
	binary.Write(w, binary.LittleEndian, int64(idx.Interval))
	binary.Write(w, binary.LittleEndian, int64(len(idx.Offsets)))
	for _, off := range idx.Offsets {
		binary.Write(w, binary.LittleEndian, off)
	}
	return w.Flush()
}

// ReadLineIndex reads a previously written index file.
func ReadLineIndex(path string) (*LineIndex, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	var totalLines, interval, count int64
	if err := binary.Read(r, binary.LittleEndian, &totalLines); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &interval); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &count); err != nil {
		return nil, err
	}

	offsets := make([]int64, count)
	for i := range offsets {
		if err := binary.Read(r, binary.LittleEndian, &offsets[i]); err != nil {
			return nil, err
		}
	}

	return &LineIndex{
		TotalLines: int(totalLines),
		Interval:   int(interval),
		Offsets:    offsets,
	}, nil
}
