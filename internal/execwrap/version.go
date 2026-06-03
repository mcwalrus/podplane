// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package execwrap

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// VersionCheck verifies if the provided version string is at least requireMajor
func VersionCheck(versionStr string, requireMajor int) (bool, error) {
	// Remove leading 'v' if present
	versionStr = strings.TrimPrefix(versionStr, "v")

	// Use regex to extract version components
	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(versionStr)

	if len(matches) < 4 {
		return false, fmt.Errorf("invalid version format: %s", versionStr)
	}

	// Parse major version
	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return false, fmt.Errorf("failed to parse major version: %v", err)
	}

	// Check if version is at least requireMajor
	return major >= requireMajor, nil
}
