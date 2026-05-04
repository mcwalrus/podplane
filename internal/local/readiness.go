// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"

	"github.com/podplane/podplane/internal/vm"
)

const localReadinessTimeout = 10 * time.Minute

// ReadinessOptions configures optional output while waiting for first-boot
// user-data to finish.
type ReadinessOptions struct {
	StreamUserdataLogs bool
}

// WaitForReadiness waits for the local VM's first-boot user-data script to
// complete successfully. It intentionally does not wait for Kubernetes here:
// local vmconfig development images may stop after user-data setup and before
// install.sh/configure.sh have been synced and run.
func (m *Local) WaitForReadiness(ctx context.Context, opts ReadinessOptions) error {
	state, err := readState(m.runtimeDir, m.clusterID)
	if err != nil {
		return err
	}
	sshPort := state.Ports.SSH
	if sshPort == 0 {
		return fmt.Errorf("state is missing ssh port")
	}

	deadline := time.Now().Add(localReadinessTimeout)
	if !opts.StreamUserdataLogs {
		fmt.Println("Waiting for cloud-init user-data to complete...")
	}
	var stopStreaming context.CancelFunc
	var streamDone <-chan struct{}
	if opts.StreamUserdataLogs {
		stopStreaming, streamDone = m.streamUserdataLogs(ctx, sshPort)
	}
	defer func() {
		if stopStreaming != nil {
			stopStreaming()
			<-streamDone
		}
	}()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return fmt.Errorf("timed out waiting for cloud-init user-data after %s; run `podplane local console` to inspect the VM boot and cloud-init logs", localReadinessTimeout)
		}
		commandTimeout := remaining
		if commandTimeout > 30*time.Second {
			commandTimeout = 30 * time.Second
		}
		output, err := m.vm.Shell(ctx, "cloud-init status --wait", sshPort, vm.ShellOptions{Timeout: commandTimeout})
		if err == nil {
			trimmed := strings.TrimSpace(string(output))
			if strings.Contains(trimmed, "status: done") || trimmed == "" {
				color.Green("✅ cloud-init user-data completed successfully")
				return nil
			}
			return fmt.Errorf("cloud-init finished with unexpected status: %s", trimmed)
		}
		if strings.Contains(string(output), "status: error") {
			return fmt.Errorf("cloud-init user-data failed: %s", strings.TrimSpace(string(output)))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

// streamUserdataLogs tails cloud-init output over SSH until the returned cancel
// function is called or the parent context is canceled.
func (m *Local) streamUserdataLogs(ctx context.Context, sshPort int) (context.CancelFunc, <-chan struct{}) {
	streamCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		fmt.Print("Waiting for cloud-init log stream..")
		command := "while [ ! -e /var/log/cloud-init-output.log ]; do printf '.'; sleep 1; done; printf '\\n'; tail -n +1 -F /var/log/cloud-init-output.log"
		for {
			_, err := m.vm.Shell(streamCtx, command, sshPort, vm.ShellOptions{
				Stdout: os.Stdout,
				Stderr: io.Discard,
			})
			if streamCtx.Err() != nil {
				return
			}
			if err != nil {
				fmt.Print(".")
				select {
				case <-streamCtx.Done():
					return
				case <-time.After(2 * time.Second):
				}
				continue
			}
			select {
			case <-streamCtx.Done():
				return
			case <-time.After(2 * time.Second):
			}
		}
	}()
	return cancel, done
}
