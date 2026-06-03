// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package pid

import (
	"encoding/json"
	"fmt"
	"os"
)

// Init checks if a PID file exists on disk, and if so unmarshals the
// PID file into the PIDFile struct, or otherwises returns an empty PIDFile
func Init(opts PIDOptions) (PIDFile, error) {
	p := PIDFile{
		options: opts,
	}
	pidFile := p.FilePath()
	// read the file
	data, err := os.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			// if the file does not exist, return an empty PIDFile struct
			return p, nil
		}
		return p, fmt.Errorf("failed to read PID file: %w", err)
	}
	// unmarshal the json
	var pData PIDFileData
	if err := json.Unmarshal(data, &pData); err != nil {
		return p, fmt.Errorf("failed to unmarshal PID file: %w", err)
	}
	p.pid = pData.PID
	p.data = pData.Data
	return p, nil
}
