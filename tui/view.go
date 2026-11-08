// Views. Minimal: no borders, no chrome — text rows + composer.
package main

import (
	"fmt"
	"strings"
)

func (m Model) View() string {
	if m.width == 0 {
		return "Loading…"
	}
	var b strings.Builder

	b.WriteString(m.headerView())
	b.WriteString("\n\n")

	if m.mode == modeDetail {
		b.WriteString(m.detail.View())
		b.WriteString("\n")
	} else {
		b.WriteString(m.listView())
	}

	b.WriteString(m.statusView())
	b.WriteString(m.composerView())
	b.WriteString(m.helpView())

	return b.String()
}

func (m Model) headerView() string {
	title := styleHeader.Render(fmt.Sprintf("Farcaster — following (fid %s)", m.fid))
	right := styleDim.Render(fmt.Sprintf("%d casts", len(m.casts)))
	if m.loading {
		right = styleDim.Render("refreshing…")
	}
	gap := m.width - lipglossWidth(title) - lipglossWidth(right) - 1
	if gap < 1 {
		gap = 1
	}
	return title + strings.Repeat(" ", gap) + right
}

func (m Model) listView() string {
	if len(m.casts) == 0 {
		if m.loading {
			return styleDim.Render("Loading feed…") + "\n"
		}
		return styleDim.Render("No casts — press r to refresh") + "\n"
	}
	vis := m.rowsVisible()
	end := m.winStart + vis
	if end > len(m.casts) {
		end = len(m.casts)
	}
	var b strings.Builder
	for i := m.winStart; i < end; i++ {
		c := m.casts[i]
		marker := "  "
		if i == m.cursor {
			marker = styleCursor.Render("▸ ")
		}
		head := marker + styleUser.Render("@"+displayName(c)) + styleDim.Render(" · "+timeAgo(c.Timestamp))
		preview := "  " + styleDim.Render(oneLine(c.Text, m.width-6))
		b.WriteString(head + "\n" + preview + "\n\n")
	}
	if len(m.casts) > vis {
		b.WriteString(styleDim.Render(fmt.Sprintf("— %d–%d of %d —", m.winStart+1, end, len(m.casts))) + "\n")
	}
	return b.String()
}

func (m Model) statusView() string {
	if m.fetchError != "" {
		return styleErr.Render("Refresh failed: "+truncateRunes(m.fetchError, m.width-18)) + "\n"
	}
	if m.postStatus != "" {
		if m.postIsErr {
			return styleErr.Render(truncateRunes(m.postStatus, m.width-2)) + "\n"
		}
		return styleOK.Render(m.postStatus) + "\n"
	}
	return ""
}

func (m Model) composerView() string {
	input := m.composer.View()
	if !m.composerFocused {
		input = styleDim.Render("> " + composerHint(m.composer.Value()))
	}
	return input + "\n"
}

func composerHint(v string) string {
	if strings.TrimSpace(v) != "" {
		return v
	}
	return "Quick cast… (i to write)"
}

func (m Model) helpView() string {
	var keys string
	switch {
	case m.composerFocused:
		keys = "enter post · esc back · ctrl+c quit"
	case m.mode == modeDetail:
		keys = "↑↓ scroll · esc back · q quit"
	default:
		keys = "j/k move · enter detail · i write · r refresh · q quit"
	}
	return styleDim.Render(keys)
}

// lipglossWidth is the visible width (lipgloss strips ANSI).
func lipglossWidth(s string) int {
	return len([]rune(stripANSI(s)))
}

func stripANSI(s string) string {
	var out []rune
	inEsc := false
	for _, r := range s {
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		if r == 0x1b {
			inEsc = true
			continue
		}
		out = append(out, r)
	}
	return string(out)
}
