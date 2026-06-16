// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"os"
	"strings"

	"golang.org/x/term"
)

const (
	// MinTUIWidth is the minimum terminal width supportd by the Podplane TUI
	MinTUIWidth = 80

	// MinTUIHeight is the minimum terminal height supported by the Podplane TUI
	MinTUIHeight = 24
)

// Capability describes whether Podplane should use a Bubble Tea terminal UI.
type Capability struct {
	OK     bool
	Width  int
	Height int
	Reason string
}

// CanUseTUI reports whether stdout supports an interactive TUI at minWidth and
// minHeight. A zero minimum skips that dimension check.
func CanUseTUI(minWidth, minHeight int) Capability {
	if strings.EqualFold(os.Getenv("CI"), "true") {
		return Capability{Reason: "CI=true"}
	}
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return Capability{Reason: "stdout is not a terminal"}
	}
	if strings.EqualFold(os.Getenv("TERM"), "dumb") {
		return Capability{Reason: "TERM=dumb"}
	}
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return Capability{Reason: "terminal size unavailable"}
	}
	if minWidth > 0 && width < minWidth {
		return Capability{Width: width, Height: height, Reason: "terminal too narrow"}
	}
	if minHeight > 0 && height < minHeight {
		return Capability{Width: width, Height: height, Reason: "terminal too short"}
	}
	return Capability{OK: true, Width: width, Height: height}
}
