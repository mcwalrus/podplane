// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package deploy

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// envNamePattern defines valid characters for an environment variable name
// passed via -e/--env.
var envNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// parseEnv parses -e/--env values of the form KEY=VALUE or KEY (the latter
// inherits the value from the current process environment).
func parseEnv(values []string) (map[string]string, error) {
	env := map[string]string{}
	for _, value := range values {
		key, envValue, found := strings.Cut(value, "=")
		key = strings.TrimSpace(key)
		if !envNamePattern.MatchString(key) {
			return nil, fmt.Errorf("invalid environment variable name %q", key)
		}
		if !found {
			var ok bool
			envValue, ok = os.LookupEnv(key)
			if !ok {
				return nil, fmt.Errorf("environment variable %s is not set", key)
			}
		}
		env[key] = envValue
	}
	return env, nil
}
