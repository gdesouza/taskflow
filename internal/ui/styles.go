package ui

import (
	lipgloss "github.com/charmbracelet/lipgloss"
)

var (
	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Background(lipgloss.Color("236")).PaddingLeft(1).PaddingRight(1)
	cursorStyle = lipgloss.NewStyle().Reverse(true)
)
