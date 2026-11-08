// Model: state + update loop. List <-> detail modes, composer focus.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type mode int

const (
	modeList mode = iota
	modeDetail
)

const (
	refreshEvery = 2 * time.Minute
	rowHeight    = 3 // lines per cast row in list view
	chromeLines  = 6 // header(2) + status(1) + composer(2) + help(1)
)

type feedLoadedMsg struct {
	casts []Cast
	fresh bool // from network (vs initial file load)
}

type feedErrMsg struct{ err error }
type postedMsg struct{}
type postErrMsg struct{ err error }
type autoRefreshMsg struct{}

type Model struct {
	apiKey   string
	fid      string
	feedPath string

	casts    []Cast
	cursor   int
	winStart int
	width    int
	height   int

	mode    mode
	detail  viewport.Model
	detailW int
	detailH int

	composer         textinput.Model
	composerFocused  bool
	postStatus       string
	postIsErr        bool
	fetchError       string
	loading          bool
	seen             map[string]bool
	suppressNotify   bool // true until first network refresh lands
}

func NewModel(apiKey, fid, feedPath string, initial []Cast) Model {
	ti := textinput.New()
	ti.Placeholder = "Quick cast…"
	ti.CharLimit = 320
	ti.Prompt = "> "

	seen := map[string]bool{}
	for _, c := range initial {
		seen[c.Hash] = true
	}
	return Model{
		apiKey:         apiKey,
		fid:            fid,
		feedPath:       feedPath,
		casts:          initial,
		composer:       ti,
		seen:           seen,
		suppressNotify: true,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		fetchFeedCmd(m.apiKey, m.fid, m.feedPath),
		scheduleRefresh(),
	)
}

func scheduleRefresh() tea.Cmd {
	return tea.Tick(refreshEvery, func(time.Time) tea.Msg { return autoRefreshMsg{} })
}

func fetchFeedCmd(key, fid, path string) tea.Cmd {
	return func() tea.Msg {
		casts, err := fetchFeed(key, fid)
		if err != nil {
			return feedErrMsg{err}
		}
		_ = saveFeedFile(path, casts)
		return feedLoadedMsg{casts: casts, fresh: true}
	}
}

func postCastCmd(key, signer, text string) tea.Cmd {
	return func() tea.Msg {
		if err := postCast(key, signer, text); err != nil {
			return postErrMsg{err}
		}
		return postedMsg{}
	}
}

func (m Model) rowsVisible() int {
	n := (m.height - chromeLines) / rowHeight
	if n < 1 {
		n = 1
	}
	return n
}

func (m *Model) clampCursor() {
	if len(m.casts) == 0 {
		m.cursor = 0
		m.winStart = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.casts) {
		m.cursor = len(m.casts) - 1
	}
	vis := m.rowsVisible()
	if m.cursor < m.winStart {
		m.winStart = m.cursor
	}
	if m.cursor >= m.winStart+vis {
		m.winStart = m.cursor - vis + 1
	}
	if m.winStart < 0 {
		m.winStart = 0
	}
}

func (m *Model) openDetail() {
	if len(m.casts) == 0 {
		return
	}
	c := m.casts[m.cursor]
	w := m.width - 4
	if w < 20 {
		w = 20
	}
	h := m.height - chromeLines
	if h < 3 {
		h = 3
	}
	vp := viewport.New(w, h)
	vp.SetContent(detailContent(c, w))
	vp.GotoTop()
	m.detail = vp
	m.mode = modeDetail
}

func detailContent(c Cast, width int) string {
	name := displayName(c)
	dn := c.Author.DisplayName
	if dn == "" {
		dn = name
	}
	body := c.Text
	if body == "" {
		body = "(no text)"
	}
	return fmt.Sprintf("%s (@%s)\n%s\n\n%s\n\n%s",
		dn, name,
		styleDim.Render(timeAgo(c.Timestamp)+" · hash "+c.Hash),
		wrapText(body, width),
		styleDim.Render("fid "+fmt.Sprint(c.Author.Fid)),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.composer.Width = msg.Width - 8
		if m.composer.Width < 20 {
			m.composer.Width = 20
		}
		if m.mode == modeDetail {
			m.openDetail() // re-wrap to new size, keeps cursor cast
		}
		m.clampCursor()
		return m, nil

	case autoRefreshMsg:
		return m, tea.Batch(fetchFeedCmd(m.apiKey, m.fid, m.feedPath), scheduleRefresh())

	case feedLoadedMsg:
		m.loading = false
		m.fetchError = ""
		m.casts = msg.casts
		var fresh []Cast
		if msg.fresh {
			for _, c := range msg.casts {
				if !m.seen[c.Hash] {
					fresh = append(fresh, c)
				}
				m.seen[c.Hash] = true
			}
		}
		m.clampCursor()
		if msg.fresh && !m.suppressNotify && len(fresh) > 0 {
			m.suppressNotify = false
			title, body := newCastNotification(fresh)
			return m, notifyCmd(title, body)
		}
		m.suppressNotify = false
		return m, nil

	case feedErrMsg:
		m.loading = false
		m.fetchError = msg.err.Error()
		return m, nil

	case postedMsg:
		m.postStatus = "Posted!"
		m.postIsErr = false
		m.composer.Reset()
		m.composer.Blur()
		m.composerFocused = false
		return m, fetchFeedCmd(m.apiKey, m.fid, m.feedPath)

	case postErrMsg:
		m.postStatus = msg.err.Error()
		m.postIsErr = true
		return m, nil

	case notifyDoneMsg:
		return m, nil

	case tea.KeyMsg:
		return m.updateKey(msg)
	}

	// Route non-key messages to focused component.
	if m.composerFocused {
		var cmd tea.Cmd
		m.composer, cmd = m.composer.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Composer focused: typing goes to the input.
	if m.composerFocused {
		switch key {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.composer.Blur()
			m.composerFocused = false
			return m, nil
		case "enter":
			text := m.composer.Value()
			if err := validatePost(text); err != nil {
				m.postStatus = err.Error()
				m.postIsErr = true
				return m, nil
			}
			m.postStatus = "Posting…"
			m.postIsErr = false
			return m, postCastCmd(m.apiKey, os.Getenv("SIGNER_UUID"), text)
		}
		var cmd tea.Cmd
		m.composer, cmd = m.composer.Update(msg)
		return m, cmd
	}

	// Detail mode.
	if m.mode == modeDetail {
		switch key {
		case "ctrl+c", "q", "esc", "enter", "backspace", "left":
			m.mode = modeList
			return m, nil
		}
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd
	}

	// List mode.
	switch key {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "down", "j":
		m.cursor++
		m.clampCursor()
	case "up", "k":
		m.cursor--
		m.clampCursor()
	case "g":
		m.cursor = 0
		m.clampCursor()
	case "G":
		m.cursor = len(m.casts) - 1
		m.clampCursor()
	case "enter", "right", "l":
		m.openDetail()
	case "i", "/":
		m.composerFocused = true
		m.postStatus = ""
		return m, m.composer.Focus()
	case "r":
		m.loading = true
		m.fetchError = ""
		return m, fetchFeedCmd(m.apiKey, m.fid, m.feedPath)
	}
	return m, nil
}
