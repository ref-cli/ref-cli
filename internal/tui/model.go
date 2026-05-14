package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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

	selectedStyle   = lipgloss.NewStyle().Foreground(colorHighlight).Bold(true)
	activePaneStyle = lipgloss.NewStyle().Foreground(colorHighlight).Bold(true)
	commentStyle    = lipgloss.NewStyle().Foreground(colorDim)
	tagStyle        = lipgloss.NewStyle().Foreground(colorTag)
	statusBarStyle  = lipgloss.NewStyle().Foreground(colorSubtle)
	divStyle        = lipgloss.NewStyle().Foreground(colorSubtle)
)

// linesPerEntry is the number of rendered lines each entry occupies in the
// preview: one comment line, one command line, and one blank separator.
const linesPerEntry = 3

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
	entryCursor int
	examplesDir string

	search  textinput.Model
	preview viewport.Model

	width  int
	height int
	ready  bool

	fullscreen bool
	activePane int // 0 = list, 1 = preview
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
	// ctrl+c always quits regardless of mode
	if msg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}

	// Arrow keys work in both input and command mode (no conflict with typing).
	switch {
	case bkey.Matches(msg, keys.Up):
		if m.activePane == 1 {
			m.moveEntry(-1)
		} else if m.cursor > 0 {
			m.cursor--
			m.clampOffset()
			m.entryCursor = 0
			m.refreshPreview()
		}
		return m, nil

	case bkey.Matches(msg, keys.Down):
		if m.activePane == 1 {
			m.moveEntry(+1)
		} else if m.cursor < len(m.filtered)-1 {
			m.cursor++
			m.clampOffset()
			m.entryCursor = 0
			m.refreshPreview()
		}
		return m, nil

	case bkey.Matches(msg, keys.PaneLeft):
		m.activePane = 0
		return m, nil

	case bkey.Matches(msg, keys.PaneRight):
		m.activePane = 1
		return m, nil
	}

	if m.search.Focused() {
		// Search mode: Esc exits to command mode, all other keys go to the input
		if msg.Type == tea.KeyEsc {
			m.search.Blur()
			return m, nil
		}
		var c tea.Cmd
		m.search, c = m.search.Update(msg)
		m.refilter()
		m.cursor = 0
		m.listOffset = 0
		m.entryCursor = bestEntryForQuery(strings.TrimSpace(m.search.Value()), m.selected())
		m.refreshPreview()
		return m, c
	}

	// Command mode: single-letter shortcuts are safe since input is blurred
	switch {
	case bkey.Matches(msg, keys.Quit):
		return m, tea.Quit

	case bkey.Matches(msg, keys.Copy):
		if ex := m.selected(); ex != nil && m.entryCursor < len(ex.Entries) {
			if clipboard.Write(ex.Entries[m.entryCursor].Command) == nil {
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

	// "i" enters search mode (vim-style insert)
	if msg.Type == tea.KeyRunes && msg.String() == "i" {
		m.search.Focus()
		return m, nil
	}
	return m, nil
}

func (m Model) updateFullscreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}
	switch {
	case bkey.Matches(msg, keys.Back):
		m.fullscreen = false
		m.syncViewport()
		return m, nil

	case bkey.Matches(msg, keys.Up):
		m.moveEntry(-1)
		return m, nil

	case bkey.Matches(msg, keys.Down):
		m.moveEntry(+1)
		return m, nil

	case bkey.Matches(msg, keys.Copy):
		if ex := m.selected(); ex != nil && m.entryCursor < len(ex.Entries) {
			if clipboard.Write(ex.Entries[m.entryCursor].Command) == nil {
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

	// Vertical separator: highlight when right pane is active
	var sep string
	if m.activePane == 1 {
		sep = activePaneStyle.Render("│")
	} else {
		sep = divStyle.Render("│")
	}

	listLines := m.renderList(lw, lh)

	m.preview.Width = m.rightW()
	m.preview.Height = lh
	previewLines := strings.Split(m.preview.View(), "\n")

	var body strings.Builder
	for i := range lh {
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
		m.renderPaneDiv() + "\n" +
		body.String() +
		divH + "\n" +
		m.renderFooter()
}

// renderPaneDiv renders the top divider with "list" / "preview" labels,
// highlighting the label for the active pane.
func (m Model) renderPaneDiv() string {
	lw := m.leftW()
	rw := m.rightW()

	rightName := "─"
	if ex := m.selected(); ex != nil {
		rightName = ex.Name
	}
	leftLabel := " command "
	rightLabel := " " + rightName + " "

	var lStyle, rStyle lipgloss.Style
	if m.activePane == 0 {
		lStyle = activePaneStyle
		rStyle = statusBarStyle
	} else {
		lStyle = statusBarStyle
		rStyle = activePaneStyle
	}

	leftFill := max(0, lw-lipgloss.Width(leftLabel))
	rightFill := max(0, rw-lipgloss.Width(rightLabel))

	left := lStyle.Render(leftLabel) + divStyle.Render(strings.Repeat("─", leftFill))
	mid := divStyle.Render("───") // aligns with " │ " in body lines
	right := rStyle.Render(rightLabel) + divStyle.Render(strings.Repeat("─", rightFill))

	return left + mid + right
}

func (m Model) viewFullscreen() string {
	ex := m.selected()
	if ex == nil {
		return ""
	}
	title := lipgloss.NewStyle().Bold(true).Render(ex.Name)
	hint := statusBarStyle.Render("  [↑↓] scroll  [esc] back  [y] copy  [ctrl+c] quit")
	div := divStyle.Render(strings.Repeat("─", m.width))

	m.preview.Width = m.width
	m.preview.Height = m.height - 3
	return title + hint + "\n" + div + "\n" + m.preview.View()
}

func (m Model) renderList(width, height int) []string {
	lines := make([]string, height)
	for i := range height {
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
	const nav = "[↑↓] scroll  [←→] switch"
	var s string
	if m.search.Focused() {
		s = nav + "  [esc] command mode  [ctrl+c] quit"
	} else {
		s = nav + "  [i] input mode  [y] copy  [e] edit  [↵] fullscreen  [ctrl+c] quit"
	}
	if m.statusMsg != "" {
		s = m.statusMsg + "  " + s
	}
	return statusBarStyle.Render(s)
}

// -------- helpers --------

func (m *Model) moveEntry(delta int) {
	ex := m.selected()
	if ex == nil {
		return
	}
	next := m.entryCursor + delta
	if next < 0 || next >= len(ex.Entries) {
		return
	}
	m.entryCursor = next
	m.preview.SetContent(m.renderContent(ex))
	m.preview.SetYOffset(m.entryCursor * linesPerEntry)
}

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
	ql := strings.ToLower(q)

	type scored struct {
		ex    *example.Example
		score int
	}
	var results []scored
	for _, ex := range m.allExamples {
		if s := scoreExample(ex, ql); s > 0 {
			results = append(results, scored{ex, s})
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})
	m.filtered = make([]*example.Example, len(results))
	for i, r := range results {
		m.filtered[i] = r.ex
	}
}

// scoreExample returns a priority score for the example against a lowercased query.
// Higher score = ranked earlier. Returns 0 if no match.
func scoreExample(ex *example.Example, ql string) int {
	name := strings.ToLower(ex.Name)
	// Tier 1: name match
	if name == ql {
		return 1000
	}
	if strings.HasPrefix(name, ql) {
		return 900
	}
	if strings.Contains(name, ql) {
		return 800
	}
	// Tier 2: command text match
	for _, e := range ex.Entries {
		cmd := strings.ToLower(e.Command)
		if strings.HasPrefix(cmd, ql) {
			return 700
		}
	}
	for _, e := range ex.Entries {
		if strings.Contains(strings.ToLower(e.Command), ql) {
			return 600
		}
	}
	// Tier 3: frontmatter tag match (e.g. "compression" → tar)
	for _, t := range ex.Frontmatter.Tags {
		if strings.Contains(strings.ToLower(t), ql) {
			return 550
		}
	}
	// Tier 4: inline tag match (e.g. "gnu" → entries tagged [GNU])
	for _, e := range ex.Entries {
		for _, t := range e.Tags {
			if strings.Contains(strings.ToLower(t), ql) {
				return 450
			}
		}
	}
	// Tier 5: comment/description match
	for _, e := range ex.Entries {
		if strings.Contains(strings.ToLower(e.Comment), ql) {
			return 400
		}
	}
	// Tier 4: fuzzy match on name
	if len(fuzzy.Find(ql, []string{name})) > 0 {
		return 300
	}
	// Tier 5: fuzzy match on full search text
	if len(fuzzy.Find(ql, []string{strings.ToLower(ex.SearchText())})) > 0 {
		return 100
	}
	return 0
}

func (m *Model) refreshPreview() {
	ex := m.selected()
	if ex == nil {
		m.preview.SetContent("")
		return
	}
	m.preview.SetContent(m.renderContent(ex))
	m.preview.SetYOffset(m.entryCursor * linesPerEntry)
}

// bestEntryForQuery returns the index of the entry whose command best matches q.
// Priority: command prefix > command contains > word overlap in command+comment.
func bestEntryForQuery(q string, ex *example.Example) int {
	if q == "" || ex == nil {
		return 0
	}
	ql := strings.ToLower(q)
	for i, e := range ex.Entries {
		if strings.HasPrefix(strings.ToLower(e.Command), ql) {
			return i
		}
	}
	for i, e := range ex.Entries {
		if strings.Contains(strings.ToLower(e.Command), ql) {
			return i
		}
	}
	words := strings.Fields(ql)
	bestIdx, bestScore := 0, 0
	for i, e := range ex.Entries {
		text := strings.ToLower(e.Command + " " + e.Comment)
		score := 0
		for _, w := range words {
			if strings.Contains(text, w) {
				score++
			}
		}
		if score > bestScore {
			bestScore = score
			bestIdx = i
		}
	}
	return bestIdx
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
	return max(14, min(24, m.width/5))
}

func (m Model) rightW() int {
	return max(1, m.width-m.leftW()-3)
}

func (m Model) listH() int {
	return max(1, m.height-4)
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

func (m Model) renderContent(ex *example.Example) string {
	var sb strings.Builder
	for i, e := range ex.Entries {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString("# ")
		for _, t := range e.Tags {
			sb.WriteString(tagStyle.Render("["+t+"]") + " ")
		}
		sb.WriteString(commentStyle.Render(e.Comment) + "\n")
		if i == m.entryCursor {
			sb.WriteString(selectedStyle.Render(padRight(e.Command, m.rightW())) + "\n")
		} else {
			sb.WriteString(e.Command + "\n")
		}
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
