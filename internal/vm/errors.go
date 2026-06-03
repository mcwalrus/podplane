// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package vm

import "errors"

// ErrAlreadyRunning is returned when a VM start is attempted but the VM
// is already running.
var ErrAlreadyRunning = errors.New("VM is already running")

// ErrNotRunning is returned when a VM stop is attempted but the VM is not
// running.
var ErrNotRunning = errors.New("VM is not running")
