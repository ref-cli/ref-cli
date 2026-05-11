package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	bkey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"

	"github.com/ref-cli/ref-cli/internal/clipboard"
	"github.com/ref-cli/ref-cli/internal/example"
)

// -------- styles --------

var (
	colorSubtle    = lipgloss.AdaptiveColor{Light: "#999999", Dark: "#666666"}
	colorHighlight = lipgloss.AdaptiveColor{Light: "#0077CC", Dark: "#5FAFFF"}
	colorDim       = lipgloss.AdaptiveColor{Light: "#555555", Dark: "#888888"}
	colorTag       = lipgloss.AdaptiveColor{Light: "#CC7700", Dark: "#FFAF5F"}

	selectedStyle  = lipgloss.NewStyle().Foreground(colorHighlight).Bold(true)
	commentStyle   = lipgloss.NewStyle().Foreground(colorDim)
	tagStyle       = lipgloss.NewStyle().Foreground(colorTag)
	statusBarStyle = lipgloss.NewStyle().Foreground(colorSubtle)
	divStyle       = lipgloss.NewStyle().Foreground(colorSubtle)
)

// -------- message types --------

type clearStatusMsg struct{}
type editorErrMsg struct{ err error }

func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return clearStatusMsg{} })
}

// -------- model --------

type Model struct {
	allExamples []*example.Example
	filtered    []*example.Example
	cursor      int
	listOffset  int
	examplesDir string

	search  textinput.Model
	preview viewport.Model

	width  int
	height int
	ready  bool

	fullscreen bool
	statusMsg  string
}

func New(examples []*example.Example, examplesDir string) Model {
	ti := textinput.New()
	ti.Placeholder = "search commands…"
	ti.Focus()
	ti.CharLimit = 64

	return Model{
		allExamples: examples,
		filtered:    examples,
		examplesDir: examplesDir,
		search:      ti,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// -------- update --------

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.syncViewport()
		return m, nil

	case tea.KeyMsg:
		if m.fullscreen {
			return m.updateFullscreen(msg)
		}
		return m.updateNormal(msg)

	case clearStatusMsg:
		m.statusMsg = ""
		return m, nil

	case editorErrMsg:
		m.statusMsg = "edit failed: " + msg.err.Error()
		return m, clearStatusAfter(4 * time.Second)
	}

	var cmds []tea.Cmd
	var c tea.Cmd
	m.search, c = m.search.Update(msg)
	cmds = append(cmds, c)
	m.preview, c = m.preview.Update(msg)
	cmds = append(cmds, c)
	return m, tea.Batch(cmds...)
}

func (m Model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case bkey.Matches(msg, keys.Quit):
		return m, tea.Quit

	case bkey.Matches(msg, keys.Up):
		if m.cursor > 0 {
			m.cursor--
			m.clampOffset()
			m.refreshPreview()
		}
		return m, nil

	case bkey.Matches(msg, keys.Down):
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			m.clampOffset()
			m.refreshPreview()
		}
		return m, nil

	case bkey.Matches(msg, keys.Copy):
		if ex := m.selected(); ex != nil {
			if clipboard.Write(ex.Raw) == nil {
				m.statusMsg = "Copied!"
				return m, clearStatusAfter(2 * time.Second)
			}
		}
		return m, nil

	case bkey.Matches(msg, keys.Edit):
		return m, m.editCmd()

	case bkey.Matches(msg, keys.Fullscreen):
		if len(m.filtered) > 0 {
			m.fullscreen = true
			m.syncViewport()
			m.preview.GotoTop()
		}
		return m, nil
	}

	// Typing → update search
	var c tea.Cmd
	m.search, c = m.search.Update(msg)
	m.refilter()
	m.cursor = 0
	m.listOffset = 0
	m.refreshPreview()
	return m, c
}

func (m Model) updateFullscreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case bkey.Matches(msg, keys.Back):
		m.fullscreen = false
		m.syncViewport()
		return m, nil

	case bkey.Matches(msg, keys.Copy):
		if ex := m.selected(); ex != nil {
			if clipboard.Write(ex.Raw) == nil {
				m.statusMsg = "Copied!"
				return m, clearStatusAfter(2 * time.Second)
			}
		}
		return m, nil
	}
	var c tea.Cmd
	m.preview, c = m.preview.Update(msg)
	return m, c
}

// -------- view --------

func (m Model) View() string {
	if !m.ready {
		return "Loading…\n"
	}
	if m.fullscreen {
		return m.viewFullscreen()
	}
	return m.viewNormal()
}

func (m Model) viewNormal() string {
	lw := m.leftW()
	lh := m.listH()
	divH := divStyle.Render(strings.Repeat("─", m.width))
	sep := divStyle.Render("│")

	listLines := m.renderList(lw, lh)

	m.preview.Width = m.rightW()
	m.preview.Height = lh
	previewLines := strings.Split(m.preview.View(), "\n")

	var body strings.Builder
	for i := 0; i < lh; i++ {
		left := ""
		if i < len(listLines) {
			left = listLines[i]
		}
		left = padRight(left, lw)
		right := ""
		if i < len(previewLines) {
			right = previewLines[i]
		}
		fmt.Fprintf(&body, "%s %s %s\n", left, sep, right)
	}

	return m.search.View() + "\n" +
		divH + "\n" +
		body.String() +
		divH + "\n" +
		m.renderFooter()
}

func (m Model) viewFullscreen() string {
	ex := m.selected()
	if ex == nil {
		return ""
	}
	title := lipgloss.NewStyle().Bold(true).Render("ref " + ex.Name)
	hint := statusBarStyle.Render("  [esc/q] back  [y] copy")
	div := divStyle.Render(strings.Repeat("─", m.width))

	m.preview.Width = m.width
	m.preview.Height = m.height - 3
	return title + hint + "\n" + div + "\n" + m.preview.View()
}

func (m Model) renderList(width, height int) []string {
	lines := make([]string, height)
	for i := 0; i < height; i++ {
		idx := m.listOffset + i
		if idx >= len(m.filtered) {
			lines[i] = strings.Repeat(" ", width)
			continue
		}
		name := m.filtered[idx].Name
		if lipgloss.Width(name) > width {
			name = name[:width]
		}
		if idx == m.cursor {
			lines[i] = padRight(selectedStyle.Render(name), width)
		} else {
			lines[i] = padRight(name, width)
		}
	}
	return lines
}

func (m Model) renderFooter() string {
	s := "[↑↓] navigate  [y] copy  [e] edit  [↵] fullscreen  [q] quit"
	if m.statusMsg != "" {
		s = m.statusMsg + "  " + s
	}
	return statusBarStyle.Render(s)
}

// -------- helpers --------

func (m Model) selected() *example.Example {
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return nil
	}
	return m.filtered[m.cursor]
}

func (m *Model) refilter() {
	q := strings.TrimSpace(m.search.Value())
	if q == "" {
		m.filtered = m.allExamples
		return
	}
	srcs := make([]string, len(m.allExamples))
	for i, ex := range m.allExamples {
		srcs[i] = ex.SearchText()
	}
	matches := fuzzy.Find(q, srcs)
	m.filtered = make([]*example.Example, len(matches))
	for i, match := range matches {
		m.filtered[i] = m.allExamples[match.Index]
	}
}

func (m *Model) refreshPreview() {
	ex := m.selected()
	if ex == nil {
		m.preview.SetContent("")
		return
	}
	m.preview.SetContent(renderContent(ex))
	m.preview.GotoTop()
}

func (m *Model) syncViewport() {
	m.preview.Width = m.rightW()
	m.preview.Height = m.listH()
	if m.fullscreen {
		m.preview.Width = m.width
		m.preview.Height = m.height - 3
	}
	m.refreshPreview()
}

func (m *Model) clampOffset() {
	lh := m.listH()
	if m.cursor < m.listOffset {
		m.listOffset = m.cursor
	}
	if m.cursor >= m.listOffset+lh {
		m.listOffset = m.cursor - lh + 1
	}
}

func (m Model) leftW() int {
	w := m.width / 5
	if w < 14 {
		w = 14
	}
	if w > 24 {
		w = 24
	}
	return w
}

func (m Model) rightW() int {
	w := m.width - m.leftW() - 3
	if w < 1 {
		w = 1
	}
	return w
}

func (m Model) listH() int {
	h := m.height - 4
	if h < 1 {
		h = 1
	}
	return h
}

func (m Model) editCmd() tea.Cmd {
	ex := m.selected()
	if ex == nil {
		return nil
	}
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	path := filepath.Join(m.examplesDir, ex.Name+".txt")
	c := exec.Command(editor, path)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return editorErrMsg{err}
		}
		return nil
	})
}

func renderContent(ex *example.Example) string {
	var sb strings.Builder
	for i, e := range ex.Entries {
		if i > 0 {
			sb.WriteByte('\n')
		}
		// comment line with optional tags
		sb.WriteString("# ")
		for _, t := range e.Tags {
			sb.WriteString(tagStyle.Render("["+t+"]") + " ")
		}
		sb.WriteString(commentStyle.Render(e.Comment) + "\n")
		sb.WriteString(e.Command + "\n")
	}
	return sb.String()
}

func padRight(s string, width int) string {
	n := lipgloss.Width(s)
	if n >= width {
		return s
	}
	return s + strings.Repeat(" ", width-n)
}

// Run starts the TUI and blocks until the user quits.
func Run(examples []*example.Example, examplesDir string) error {
	m := New(examples, examplesDir)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
