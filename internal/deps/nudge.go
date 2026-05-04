// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package deps

import (
	"context"
	"fmt"
	"os"
	"time"
)

// IsStale reports whether the cached manifest is older than threshold (or
// missing entirely). It is a cheap, synchronous check used to gate whether
// CheckUpdateNudge is worth running.
func (m *Manager) IsStale(kind, arch string, threshold time.Duration) bool {
	info, err := os.Stat(m.VMConfigManifestCachePath(kind, arch))
	if err != nil {
		// No cached manifest (or unreadable) — treat as stale.
		return true
	}
	return time.Since(info.ModTime()) > threshold
}

// CheckUpdateNudge attempts to determine whether a newer manifest is
// available, returning a one-line nudge message for the user. It is designed
// to be called from a background goroutine and never returns an error: any
// network/parse failure is folded into a friendly message.
//
// Behaviour:
//   - Fetches the latest manifest (subject to ctx).
//   - On success with a different version: returns an "update available" note
//     and refreshes the cached manifest's mtime so we don't re-nag every
//     start.
//   - On success with the same version: refreshes mtime; returns "".
//   - On failure (timeout, network error, parse error): returns a "couldn't
//     check, manifest is N days old" note.
func (m *Manager) CheckUpdateNudge(ctx context.Context, kind, arch string) string {
	cachedPath := m.VMConfigManifestCachePath(kind, arch)
	cachedInfo, statErr := os.Stat(cachedPath)

	cached, _, err := m.ReadCachedManifest(kind, arch)
	if err != nil || cached == nil {
		// No cached manifest at all — Verify/Download paths handle this case;
		// the nudge is silent.
		return ""
	}

	latest, fetchErr := m.fetchManifest(ctx, kind, arch, nil)
	if fetchErr != nil {
		days := 0
		if statErr == nil {
			days = int(time.Since(cachedInfo.ModTime()).Hours() / 24)
		}
		return fmt.Sprintf(
			"ℹ️ Manifest is %d days old and we couldn't check for updates. "+
				"Run `podplane deps download` when online.",
			days,
		)
	}

	// Touch the manifest so the 7-day staleness clock resets, regardless of
	// whether the version changed.
	now := time.Now()
	_ = os.Chtimes(cachedPath, now, now)

	if latest.VMConfig.Version != cached.VMConfig.Version {
		return fmt.Sprintf(
			"ℹ️ Newer dependencies available (%s → %s). Run `podplane deps download`.",
			cached.VMConfig.Version, latest.VMConfig.Version,
		)
	}
	return ""
}
