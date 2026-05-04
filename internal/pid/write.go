// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package pid

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Write marshal's the PIDFile struct and writes it to the PID file
func (p *PIDFile) Write() error {
	pidFile := p.FilePath()
	// ensure all parent directories exist
	dir := filepath.Dir(pidFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	// marshall to json
	data, err := json.Marshal(PIDFileData{
		PID:  p.pid,
		Data: p.data,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal PID file: %w", err)
	}

	// write to file
	return os.WriteFile(pidFile, data, 0644)
}
