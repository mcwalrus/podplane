// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// Confirm asks the user to confirm an action, using Bubble Tea in terminals
// and a plain text prompt when interactive terminal UI is unavailable.
func Confirm(message string, autoApprove bool) (bool, error) {
	if autoApprove {
		return true, nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return confirmText(message)
	}
	m, err := tea.NewProgram(confirmModel{message: message}).Run()
	if err != nil {
		return false, fmt.Errorf("error running confirmation prompt: %w", err)
	}
	if m, ok := m.(confirmModel); ok {
		return m.confirmed, nil
	}
	return false, fmt.Errorf("error processing confirmation")
}

// confirmText asks for an explicit "yes" response using standard input and
// output.
func confirmText(message string) (bool, error) {
	fmt.Printf("%s Type yes to continue: ", message)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(line), "yes"), nil
}

type confirmModel struct {
	message   string
	selected  int
	confirmed bool
	done      bool
}

// Init starts the confirmation prompt without any startup command.
func (m confirmModel) Init() tea.Cmd {
	return nil
}

// Update handles key presses for the confirmation prompt.
func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			m.done = true
			return m, tea.Quit
		case "left", "h", "right", "l", "tab":
			if m.selected == 0 {
				m.selected = 1
			} else {
				m.selected = 0
			}
			return m, nil
		case "y":
			m.confirmed = true
			m.done = true
			return m, tea.Quit
		case "n":
			m.confirmed = false
			m.done = true
			return m, tea.Quit
		case "enter":
			m.confirmed = m.selected == 0
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the confirmation prompt.
func (m confirmModel) View() string {
	if m.done {
		return "\n"
	}
	titleStyle := lipgloss.NewStyle().Background(ColorPrimary).Foreground(ColorWhite).Padding(0, 1)
	selectedStyle := lipgloss.NewStyle().Foreground(ColorSecondary)
	yes := "Yes"
	no := "No"
	if m.selected == 0 {
		yes = selectedStyle.Render("> " + yes)
		no = "  " + no
	} else {
		yes = "  " + yes
		no = selectedStyle.Render("> " + no)
	}
	return fmt.Sprintf("\n%s\n\n%s    %s\n\n", titleStyle.Render(m.message), yes, no)
}
