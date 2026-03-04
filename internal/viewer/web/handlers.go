package web

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

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
