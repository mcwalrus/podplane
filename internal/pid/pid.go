// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package pid

import (
	"fmt"
	"path/filepath"
)

// PIDOptions holds config
type PIDOptions struct {
	Directory string
	Filename  string
}

// PIDFile represents a process PID file
type PIDFile struct {
	options PIDOptions
	pid     int
	data    map[string]string
}

// PIDFileData represents the data structure of the PID file
type PIDFileData struct {
	PID  int               `json:"pid"`
	Data map[string]string `json:"data"`
}

// FilePath returns the path to the PID file
func (p *PIDFile) FilePath() string {
	return filepath.Join(p.options.Directory, p.options.Filename)
}

// PID returns the process id
func (p *PIDFile) PID() int {
	return p.pid
}

// SetPID sets the process id
func (p *PIDFile) SetPID(pid int) error {
	if p.pid != 0 {
		return fmt.Errorf("PID already set to %d", p.pid)
	}
	p.pid = pid
	return nil
}

// GetData returns the value of the specified key from the data map
func (p *PIDFile) GetData(key string) string {
	if p.data == nil {
		return ""
	}
	return p.data[key]
}

// SetData sets the value of the specified key in the data map
func (p *PIDFile) SetData(key, value string) {
	if p.data == nil {
		p.data = make(map[string]string)
	}
	p.data[key] = value
}

// ClearData clears the data map
func (p *PIDFile) ClearData() {
	p.data = nil
}
