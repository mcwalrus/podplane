// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package clusterdelete

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/podplane/podplane/internal/clusterconfig"
)

// Executor provides Terraform outputs and destroy behavior for cluster delete.
type Executor interface {
	OutputJSON(ctx context.Context, dir string) ([]byte, error)
	Destroy(ctx context.Context, dir string, autoApprove bool) error
}

// Provider performs provider-specific cleanup before and after destroy.
type Provider interface {
	Prepare(ctx context.Context, cfg *clusterconfig.ClusterConfig, outputs []byte) error
	ScaleManagedGroupsToZero(ctx context.Context) error
	WaitForManagedInstancesGone(ctx context.Context) error
	RemainingManagedInstances(ctx context.Context) ([]Instance, error)
	TerminateInstances(ctx context.Context, instances []Instance) error
	ListTaggedInstances(ctx context.Context, managedOnly bool) ([]Instance, error)
	ClusterID() string
}

// Instance describes a provider VM instance that may need cleanup.
type Instance struct {
	ID     string
	Name   string
	Shard  string
	Group  string
	State  string
	Reason string
}

// ConfirmFunc asks the user to confirm a destructive action.
type ConfirmFunc func(message string) (bool, error)

// StatusFunc reports delete workflow status to the caller.
type StatusFunc func(message string)

// DeleteOptions controls the cluster delete workflow.
type DeleteOptions struct {
	ClusterConfig  *clusterconfig.ClusterConfig
	StackDir       string
	Executor       Executor
	Provider       Provider
	AutoApprove    bool
	Confirm        ConfirmFunc
	Status         StatusFunc
	OutputTimeout  time.Duration
	DestroyTimeout time.Duration
}

// Delete runs the provider cleanup workflow and then destroys managed
// OpenTofu/Terraform resources.
func Delete(ctx context.Context, opts DeleteOptions) error {
	if opts.ClusterConfig == nil {
		return fmt.Errorf("cluster config is required")
	}
	if opts.Executor == nil {
		return fmt.Errorf("OpenTofu/Terraform executor is required")
	}
	if opts.Provider == nil {
		return fmt.Errorf("cluster delete provider is required")
	}
	confirm := opts.Confirm
	if confirm == nil {
		confirm = func(string) (bool, error) {
			return false, fmt.Errorf("confirmation callback is required")
		}
	}
	status := opts.Status
	if status == nil {
		status = func(string) {}
	}

	outputTimeout := opts.OutputTimeout
	if outputTimeout == 0 {
		outputTimeout = 30 * time.Minute
	}
	outputCtx, outputCancel := context.WithTimeout(ctx, outputTimeout)
	defer outputCancel()
	outputs, err := opts.Executor.OutputJSON(outputCtx, opts.StackDir)
	if err != nil {
		return err
	}
	if err := opts.Provider.Prepare(ctx, opts.ClusterConfig, outputs); err != nil {
		return err
	}

	status("Scaling Nstance-managed groups to zero in shard configs.")
	if err := opts.Provider.ScaleManagedGroupsToZero(ctx); err != nil {
		return err
	}
	status("Waiting for Nstance-managed EC2 instances to terminate.")
	if err := opts.Provider.WaitForManagedInstancesGone(ctx); err != nil {
		remaining, listErr := opts.Provider.RemainingManagedInstances(ctx)
		if listErr != nil {
			return fmt.Errorf("%w; additionally failed to list remaining managed instances: %v", err, listErr)
		}
		if len(remaining) == 0 {
			return err
		}
		status(fmt.Sprintf("Nstance-managed EC2 instances are still present after scale-down: %s", FormatInstances(remaining)))
		ok, confirmErr := confirm("Terminate remaining Nstance-managed EC2 instances directly before destroy?")
		if confirmErr != nil {
			return confirmErr
		}
		if !ok {
			return err
		}
		if terminateErr := opts.Provider.TerminateInstances(ctx, remaining); terminateErr != nil {
			return terminateErr
		}
		if waitErr := opts.Provider.WaitForManagedInstancesGone(ctx); waitErr != nil {
			return waitErr
		}
	}

	ok, err := confirm("Destroy OpenTofu/Terraform-managed resources?")
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("destroy cancelled")
	}
	destroyTimeout := opts.DestroyTimeout
	if destroyTimeout == 0 {
		destroyTimeout = 2 * time.Hour
	}
	destroyCtx, destroyCancel := context.WithTimeout(ctx, destroyTimeout)
	defer destroyCancel()
	if err := opts.Executor.Destroy(destroyCtx, opts.StackDir, opts.AutoApprove); err != nil {
		return err
	}

	dangling, err := opts.Provider.ListTaggedInstances(ctx, false)
	if err != nil {
		return err
	}
	if len(dangling) > 0 {
		status(fmt.Sprintf("Found dangling instances tagged for cluster %q: %s", opts.Provider.ClusterID(), FormatInstances(dangling)))
		ok, err := confirm("Terminate dangling EC2 instances?")
		if err != nil {
			return err
		}
		if ok {
			if err := opts.Provider.TerminateInstances(ctx, dangling); err != nil {
				return err
			}
		}
	}
	return nil
}

// FormatInstances formats provider instances for human-readable status and
// confirmation messages.
func FormatInstances(instances []Instance) string {
	parts := make([]string, 0, len(instances))
	for _, inst := range instances {
		label := inst.ID
		if inst.Name != "" {
			label += " (" + inst.Name + ")"
		}
		if inst.Shard != "" {
			label += " shard=" + inst.Shard
		}
		if inst.Group != "" {
			label += " group=" + inst.Group
		}
		if inst.State != "" {
			label += " state=" + inst.State
		}
		parts = append(parts, label)
	}
	return strings.Join(parts, ", ")
}
