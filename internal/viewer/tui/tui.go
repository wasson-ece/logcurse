package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/wasson-ece/logcurse/internal/fileutil"
	"github.com/wasson-ece/logcurse/internal/model"
)

const (
	focusFile    = 0
	focusComment = 1
)

type layout int

const (
	layoutVertical   layout = iota // side by side
	layoutHorizontal               // stacked
)

// Model is the Bubble Tea model for the TUI viewer.
type Model struct {
	sourceFile string
	lines      []string
	comments   *model.CommentFile
	hasYML     bool

	fileViewport    viewport.Model
	commentViewport viewport.Model

	focus         int
	layout        layout
	currentComment int // index into comments, -1 if none
	keys          keyMap
	showHelp      bool

	width  int
	height int
	ready  bool
}

// New creates a new TUI model.
func New(sourceFile string) (*Model, error) {
	lines, err := fileutil.ReadAllLines(sourceFile)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", sourceFile, err)
	}

	m := &Model{
		sourceFile:     sourceFile,
		lines:          lines,
		keys:           defaultKeyMap(),
		currentComment: -1,
	}

	// Try to load sidecar
	ymlPath := model.SidecarPath(sourceFile)
	if _, err := os.Stat(ymlPath); err == nil {
		cf, err := model.LoadAndCheckDrift(ymlPath, sourceFile)
		if err != nil {
			return nil, fmt.Errorf("loading comments: %w", err)
		}
		m.comments = cf
		m.hasYML = true
		if len(cf.Comments) > 0 {
			m.currentComment = 0
		}
	} else {
		m.comments = &model.CommentFile{}
	}

	return m, nil
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.recalcViewports()
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Tab):
			m.focus = (m.focus + 1) % 2
			return m, nil

		case key.Matches(msg, m.keys.NextComment):
			if len(m.comments.Comments) > 0 {
				m.currentComment = (m.currentComment + 1) % len(m.comments.Comments)
				m.scrollToComment()
			}
			return m, nil

		case key.Matches(msg, m.keys.PrevComment):
			if len(m.comments.Comments) > 0 {
				m.currentComment--
				if m.currentComment < 0 {
					m.currentComment = len(m.comments.Comments) - 1
				}
				m.scrollToComment()
			}
			return m, nil

		case key.Matches(msg, m.keys.ToggleLayout):
			if m.layout == layoutVertical {
				m.layout = layoutHorizontal
			} else {
				m.layout = layoutVertical
			}
			m.recalcViewports()
			return m, nil

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			m.recalcViewports()
			return m, nil

		default:
			// Pass to focused viewport
			if m.focus == focusFile {
				var cmd tea.Cmd
				m.fileViewport, cmd = m.fileViewport.Update(msg)
				return m, cmd
			}
			var cmd tea.Cmd
			m.commentViewport, cmd = m.commentViewport.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m *Model) scrollToComment() {
	if m.currentComment < 0 || m.currentComment >= len(m.comments.Comments) {
		return
	}
	c := m.comments.Comments[m.currentComment]
	// Scroll file viewport to the comment's start line
	targetLine := c.Range.Start - 1 // 0-indexed
	m.fileViewport.SetYOffset(targetLine)
	m.commentViewport.SetYOffset(targetLine)
}

var (
	fileBorderActive = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39"))

	fileBorderInactive = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))

	commentBorderActive = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205"))

	commentBorderInactive = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))

	driftStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	commentHighlight = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220"))

	lineNumStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	statusStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("252")).
		Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
)

func (m *Model) recalcViewports() {
	if !m.ready {
		return
	}

	// Reserve 2 lines for status + help bars
	reserved := 2
	if m.showHelp {
		reserved = 4
	}
	contentHeight := m.height - reserved

	var fileW, fileH, commentW, commentH int

	if m.layout == layoutVertical {
		// Side by side, account for border (2 chars each side)
		halfW := m.width/2 - 2
		fileW = halfW
		fileH = contentHeight - 2 // border top+bottom
		commentW = m.width - halfW - 4
		commentH = fileH
	} else {
		// Stacked
		fullW := m.width - 2
		halfH := contentHeight/2 - 2
		fileW = fullW
		fileH = halfH
		commentW = fullW
		commentH = contentHeight - halfH - 4
	}

	m.fileViewport = viewport.New(fileW, fileH)
	m.commentViewport = viewport.New(commentW, commentH)

	m.fileViewport.SetContent(m.renderFileContent(fileW))
	m.commentViewport.SetContent(m.renderCommentContent(commentW))
}

func (m *Model) renderFileContent(width int) string {
	totalLines := len(m.lines)
	lineNumW := len(fmt.Sprintf("%d", totalLines))

	// Build a set of highlighted line ranges
	highlightLines := make(map[int]bool)
	if m.currentComment >= 0 && m.currentComment < len(m.comments.Comments) {
		c := m.comments.Comments[m.currentComment]
		for i := c.Range.Start; i <= c.Range.End; i++ {
			highlightLines[i] = true
		}
	}

	var sb strings.Builder
	for i, line := range m.lines {
		lineNum := i + 1
		numStr := lineNumStyle.Render(fmt.Sprintf("%*d", lineNumW, lineNum))

		displayLine := line
		if len(displayLine) > width-lineNumW-2 {
			displayLine = displayLine[:width-lineNumW-2]
		}

		if highlightLines[lineNum] {
			displayLine = commentHighlight.Render(displayLine)
		}

		sb.WriteString(fmt.Sprintf("%s  %s", numStr, displayLine))
		if i < len(m.lines)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func (m *Model) renderCommentContent(width int) string {
	if len(m.comments.Comments) == 0 {
		return "No comments found."
	}

	// Build comment content aligned to line positions.
	// Each comment is placed at its start line offset.
	totalLines := len(m.lines)
	commentLines := make([]string, totalLines)

	for i, c := range m.comments.Comments {
		if c.Range.Start < 1 || c.Range.Start > totalLines {
			continue
		}

		prefix := fmt.Sprintf("[%s] L%d-%d", c.ID, c.Range.Start, c.Range.End)
		if c.Drifted {
			prefix = driftStyle.Render("[DRIFT] ") + prefix
		}
		if i == m.currentComment {
			prefix = ">>> " + prefix
		}

		bodyLines := strings.Split(strings.TrimRight(c.Body, "\n"), "\n")
		idx := c.Range.Start - 1
		if idx < totalLines {
			commentLines[idx] = prefix
		}
		for j, bl := range bodyLines {
			lineIdx := idx + 1 + j
			if lineIdx < totalLines {
				commentLines[lineIdx] = "  " + bl
			}
		}
	}

	return strings.Join(commentLines, "\n")
}

func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	var fileStyle, cStyle lipgloss.Style
	if m.focus == focusFile {
		fileStyle = fileBorderActive
		cStyle = commentBorderInactive
	} else {
		fileStyle = fileBorderInactive
		cStyle = commentBorderActive
	}

	filePane := fileStyle.Render(m.fileViewport.View())
	commentPane := cStyle.Render(m.commentViewport.View())

	var content string
	if m.layout == layoutVertical {
		content = lipgloss.JoinHorizontal(lipgloss.Top, filePane, commentPane)
	} else {
		content = lipgloss.JoinVertical(lipgloss.Left, filePane, commentPane)
	}

	// Status bar
	commentInfo := "No comments"
	if len(m.comments.Comments) > 0 {
		commentInfo = fmt.Sprintf("Comment %d/%d", m.currentComment+1, len(m.comments.Comments))
	}
	focusLabel := "FILE"
	if m.focus == focusComment {
		focusLabel = "COMMENTS"
	}
	status := statusStyle.Width(m.width).Render(
		fmt.Sprintf(" %s | %s | %d lines | ? help", m.sourceFile, focusLabel, len(m.lines)) +
			strings.Repeat(" ", max(0, m.width-len(m.sourceFile)-len(focusLabel)-len(commentInfo)-30)) +
			commentInfo,
	)

	help := ""
	if m.showHelp {
		var parts []string
		for _, k := range m.keys.helpKeys() {
			parts = append(parts, fmt.Sprintf("%s: %s", k.Help().Key, k.Help().Desc))
		}
		help = helpStyle.Render(strings.Join(parts, "  ")) + "\n"
	}

	return content + "\n" + status + "\n" + help
}

// Run starts the Bubble Tea program.
func Run(sourceFile string) error {
	m, err := New(sourceFile)
	if err != nil {
		return err
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
