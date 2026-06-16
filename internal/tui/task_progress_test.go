// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTaskProgressSmallTerminalShowsResizePrompt(t *testing.T) {
	m := taskProgressModel{width: MinTUIWidth - 1, height: MinTUIHeight, tui: true}

	view := m.View()
	if !strings.Contains(view, "Podplane local start is still running.") {
		t.Fatalf("small terminal view missing running message:\n%s", view)
	}
	if !strings.Contains(view, "The installer view needs at least 80x24.") {
		t.Fatalf("small terminal view missing minimum size message:\n%s", view)
	}
	if !strings.Contains(view, "Press p to disable the TUI and continue with plain output.") {
		t.Fatalf("small terminal view missing plain-output hint:\n%s", view)
	}
}

func TestTaskProgressSmallTerminalPDisablesTUI(t *testing.T) {
	m := taskProgressModel{width: MinTUIWidth - 1, height: MinTUIHeight, tui: true}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	updatedModel, ok := updated.(taskProgressModel)
	if !ok {
		t.Fatalf("Update returned %T, want taskProgressModel", updated)
	}
	if updatedModel.tui {
		t.Fatal("pressing p in a small terminal should disable TUI")
	}
}
