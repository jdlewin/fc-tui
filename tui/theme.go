// All styling lives here. Terminal defaults for now (no explicit colors —
// just bold/dim), so any terminal theme applies. Omarchy theme switching
// later = add a Theme struct + setter in this file; views must only use
// these vars, never raw lipgloss styles.
package main

import "github.com/charmbracelet/lipgloss"

var (
	styleHeader = lipgloss.NewStyle().Bold(true)
	styleUser   = lipgloss.NewStyle().Bold(true)
	styleDim    = lipgloss.NewStyle().Faint(true)
	styleCursor = lipgloss.NewStyle().Bold(true)
	styleErr    = lipgloss.NewStyle().Bold(true)
	styleOK     = lipgloss.NewStyle().Bold(true)
)
