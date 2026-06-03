// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package execwrap

import (
	"fmt"
	"log/slog"
	"os/exec"
)

// Installed checks if a dependency is installed
func Installed(dependencies []string) error {
	for _, dependency := range dependencies {
		slog.Debug("checking dependency", "dependency", dependency)

		// get dependency metadata
		depMetadata, ok := Dependencies[dependency]
		if !ok {
			panic("Unexpected invalid dependency key")
		}

		// check binary is on path
		path, err := exec.LookPath(depMetadata.Binary)
		if err != nil {
			slog.Error("dependency binary not found on PATH",
				"dependency", dependency,
				"binary", depMetadata.Binary,
				"err", err,
			)
			return fmt.Errorf("%s is not installed or not found on your shell path", dependency)
		}
		slog.Debug("found dependency binary",
			"dependency", dependency,
			"binary", depMetadata.Binary,
			"path", path,
		)

		// check version
		cmd := Command(depMetadata.Binary, (depMetadata.VersionArgs)...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			slog.Error("failed to run version command",
				"dependency", dependency,
				"binary", depMetadata.Binary,
				"err", err,
			)
			return fmt.Errorf("failed to get %s version", depMetadata.Binary)
		}
		version, err := depMetadata.VersionParse(string(output))
		if err != nil {
			slog.Error("failed to parse version output",
				"dependency", dependency,
				"binary", depMetadata.Binary,
				"output", string(output),
				"err", err,
			)
			return fmt.Errorf("failed to parse %s version", depMetadata.Binary)
		}
		ok, err = VersionCheck(version, depMetadata.MinVersion)
		if err != nil {
			slog.Error("failed to check version",
				"dependency", dependency,
				"binary", depMetadata.Binary,
				"version", version,
				"err", err,
			)
			return fmt.Errorf("failed to check %s version", depMetadata.Binary)
		}
		if !ok {
			slog.Error("dependency version too low",
				"dependency", dependency,
				"binary", depMetadata.Binary,
				"version", version,
				"min_major", depMetadata.MinVersion,
			)
			return fmt.Errorf("%s version %s is too low, please upgrade to %d or higher", dependency, version, depMetadata.MinVersion)
		}
		slog.Debug("dependency version OK",
			"dependency", dependency,
			"binary", depMetadata.Binary,
			"version", version,
			"min_major", depMetadata.MinVersion,
		)
	}
	return nil
}
