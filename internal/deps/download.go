// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package deps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/podplane/podplane/internal/filecache"
)

const defaultDownloadConcurrency = 4

var registryConcurrencyOverrides = map[string]int{
	"public.ecr.aws": 1,
}

// DownloadEventType describes a dependency download progress event.
type DownloadEventType string

const (
	DownloadEventStatus   DownloadEventType = "status"
	DownloadEventChecking DownloadEventType = "checking"
	DownloadEventCached   DownloadEventType = "cached"
	DownloadEventQueued   DownloadEventType = "queued"
	DownloadEventStarted  DownloadEventType = "started"
	DownloadEventProgress DownloadEventType = "progress"
	DownloadEventDone     DownloadEventType = "done"
	DownloadEventFailed   DownloadEventType = "failed"
)

// DownloadEvent reports progress for a single dependency artifact, or a
// top-level status message when Name is empty.
type DownloadEvent struct {
	Type    DownloadEventType
	Name    string
	Path    string
	Message string
	Current int64
	Total   int64
	Err     error
}

// DownloadOptions controls dependency download behavior.
type DownloadOptions struct {
	DryRun      bool
	Concurrency int
	Client      *http.Client
	Progress    func(DownloadEvent)
	Output      io.Writer
	Archs       []string
	Providers   []string
	Addons      []string

	// SkipComponentImages skips components manifest loading and image mirroring.
	// It is used by callers that are downloading multiple VMConfig archs but only
	// need to populate the shared component mirror once.
	SkipComponentImages bool

	// VMConfigManifestPath, when set, reads the vmconfig manifest from this
	// local JSON file instead of fetching it from the configured deps base URL.
	VMConfigManifestPath string

	// ComponentsManifestPath, when set, reads the components manifest from this
	// local JSON file instead of fetching it from the configured deps base URL.
	ComponentsManifestPath string
}

type pendingDownload struct {
	Item           Item
	Dest           string
	Algo           string
	Checksum       string
	Image          ComponentImage
	ComponentIndex int
	Kind           pendingDownloadKind
}

type pendingDownloadKind string

const (
	pendingDownloadVMConfig       pendingDownloadKind = "vmconfig"
	pendingDownloadComponentImage pendingDownloadKind = "component-image"
)

// Download fetches the latest manifests and downloads any referenced vmconfig
// artifacts and component images into the local deps cache. The manifests are
// cached on success so they can be reused later.
//
// If dryRun is true, no network downloads are performed and the manifest is
// not written to the cache; instead, a summary of what would be downloaded
// is printed. Items that are already cached (with matching sha256) are
// skipped in both modes — dry-run only lists files that would actually be
// downloaded.
func (m *Manager) Download(kind, arch string, opts DownloadOptions) error {
	output := opts.Output
	if output == nil {
		output = os.Stdout
	}

	progress := opts.Progress
	var progressMu sync.Mutex
	emit := func(event DownloadEvent) {
		if progress == nil {
			return
		}
		progressMu.Lock()
		defer progressMu.Unlock()
		progress(event)
	}

	client := opts.Client
	if client == nil {
		client = newDownloadHTTPClient(opts.Concurrency)
	}

	ctx := context.Background()
	vmconfigManifestSource := fmt.Sprintf("%s/vmconfig/manifests/%s", m.baseURL, manifestFilename(kind, arch))
	var manifest *Manifest
	var err error
	if opts.VMConfigManifestPath != "" {
		vmconfigManifestSource = opts.VMConfigManifestPath
		emit(DownloadEvent{Type: DownloadEventStatus, Message: fmt.Sprintf("Reading vmconfig manifest from %s...", opts.VMConfigManifestPath)})
		manifest, err = readVMConfigManifestFile(opts.VMConfigManifestPath)
	} else {
		emit(DownloadEvent{Type: DownloadEventStatus, Message: "Fetching vmconfig manifest..."})
		manifest, err = m.fetchManifest(ctx, kind, arch, client)
	}
	if err != nil {
		return fmt.Errorf("failed to load vmconfig manifest: %w", err)
	}
	if opts.VMConfigManifestPath == "" {
		for _, it := range manifest.Items() {
			if it.IsUnreleasedVMConfigStub() {
				return fmt.Errorf("published vmconfig manifest contains unreleased vmconfig stub")
			}
		}
	}
	manifest.ResetCached()
	items := manifest.DownloadItems(ItemFilter{Providers: opts.Providers})
	for _, it := range items {
		if _, _, err := it.Dep.ParseDigest(); err != nil {
			return fmt.Errorf("invalid digest for %s: %w", it.Name, err)
		}
		if it.Dep.URL == "" {
			return fmt.Errorf("missing URL for %s", it.Name)
		}
		if it.Dep.Size <= 0 {
			return fmt.Errorf("missing size for %s", it.Name)
		}
	}

	componentsManifest := &ComponentsManifest{}
	componentsSource := ""
	componentIndexes := []int{}
	if !opts.SkipComponentImages {
		componentsManifest, componentsSource, err = m.loadComponentsManifest(ctx, client, opts, emit)
		if err != nil {
			return err
		}
		componentsManifest.ResetCached()
		componentArchs, err := normalizeArchs(opts.Archs, arch)
		if err != nil {
			return err
		}
		componentIndexes = componentsManifest.DownloadImageIndexes(ComponentImageFilter{Archs: componentArchs, Providers: opts.Providers, Addons: opts.Addons})
	}
	for _, index := range componentIndexes {
		image := componentsManifest.Components.Images[index]
		if image.Image == "" {
			return fmt.Errorf("component %s has empty image", image.Component)
		}
		if image.Digest == "" {
			return fmt.Errorf("component %s image %s has empty digest", image.Component, image.Image)
		}
		if image.Size <= 0 {
			return fmt.Errorf("component %s image %s has missing size", image.Component, image.Image)
		}
	}

	if opts.DryRun {
		fmt.Fprintf(output, "VMConfig manifest: %s\n", vmconfigManifestSource)
		fmt.Fprintf(output, "VMConfig cache directory: %s\n", m.VMConfigCacheDir())
		fmt.Fprintln(output, "VMConfig artifacts that would be downloaded:")
	} else {
		totalItems := len(items) + len(componentIndexes)
		emit(DownloadEvent{Type: DownloadEventStatus, Message: fmt.Sprintf("Checking %d vmconfig artifacts and component images in cache...", totalItems)})
		for _, it := range items {
			emit(DownloadEvent{Type: DownloadEventChecking, Name: it.Name, Total: it.Dep.Size})
		}
		for _, index := range componentIndexes {
			image := componentsManifest.Components.Images[index]
			emit(DownloadEvent{Type: DownloadEventChecking, Name: image.Image, Total: image.Size})
		}
	}

	pending := make([]pendingDownload, 0, len(items)+len(componentIndexes))
	missing := 0
	componentMissing := 0
	vmconfigPending := 0
	componentPending := 0
	for _, it := range items {
		algo, hex, err := it.Dep.ParseDigest()
		if err != nil {
			return fmt.Errorf("invalid digest for %s: %w", it.Name, err)
		}
		if it.Dep.URL == "" {
			return fmt.Errorf("missing URL for %s", it.Name)
		}
		if it.Dep.Size <= 0 {
			return fmt.Errorf("missing size for %s", it.Name)
		}

		dest := m.VMConfigArtifactCachePath(it.Name, it.Dep)
		exists, err := filecache.Exists(dest, algo, hex)
		if err != nil {
			return fmt.Errorf("failed to check cache for %s: %w", it.Name, err)
		}
		if exists {
			manifest.MarkCached(it.Name)
			emit(DownloadEvent{Type: DownloadEventCached, Name: it.Name, Path: dest, Current: it.Dep.Size, Total: it.Dep.Size})
			continue
		}
		missing++

		if opts.DryRun {
			rel := strings.TrimPrefix(
				strings.TrimPrefix(dest, m.depsCacheDir),
				string(filepath.Separator),
			)
			fmt.Fprintf(output, "  • %s\n", rel)
			continue
		}

		pending = append(pending, pendingDownload{Kind: pendingDownloadVMConfig, Item: it, Dest: dest, Algo: algo, Checksum: hex})
		vmconfigPending++
		emit(DownloadEvent{Type: DownloadEventQueued, Name: it.Name, Path: dest})
	}
	for _, index := range componentIndexes {
		image := componentsManifest.Components.Images[index]
		cached, err := componentImageCached(m.ComponentsImagesCacheDir(), image)
		if err != nil {
			return err
		}
		if cached {
			componentsManifest.MarkCached(index)
			emit(DownloadEvent{Type: DownloadEventCached, Name: image.Image, Current: image.Size, Total: image.Size})
			continue
		}
		componentMissing++
		if !opts.DryRun {
			pending = append(pending, pendingDownload{Kind: pendingDownloadComponentImage, Image: image, ComponentIndex: index})
			componentPending++
			emit(DownloadEvent{Type: DownloadEventQueued, Name: image.Image, Total: image.Size})
		}
	}

	if opts.DryRun {
		if missing == 0 && componentMissing == 0 {
			fmt.Fprintln(output, "All dependencies already cached, nothing to download.")
		}
		if !opts.SkipComponentImages {
			fmt.Fprintf(output, "Components manifest: %s\n", componentsSource)
			fmt.Fprintf(output, "Component image cache directory: %s\n", m.ComponentsImagesCacheDir())
			fmt.Fprintf(output, "Component images: %d\n", len(componentIndexes))
		}
		return nil
	}

	if len(pending) == 0 {
		emit(DownloadEvent{Type: DownloadEventStatus, Message: "All dependencies already cached, nothing to download."})
	} else {
		emit(DownloadEvent{Type: DownloadEventStatus, Message: downloadPendingMessage(vmconfigPending, componentPending)})
		if err := m.downloadPending(pending, opts.Concurrency, client, emit); err != nil {
			return err
		}
		for _, job := range pending {
			switch job.Kind {
			case pendingDownloadComponentImage:
				componentsManifest.MarkCached(job.ComponentIndex)
			default:
				manifest.MarkCached(job.Item.Name)
			}
		}
	}
	filteredRaw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode cached vmconfig manifest: %w", err)
	}
	raw := append(filteredRaw, '\n')

	if err := m.WriteCachedManifest(kind, arch, raw); err != nil {
		return fmt.Errorf("failed to write cached manifest: %w", err)
	}

	if !opts.SkipComponentImages {
		filteredRaw, err := json.MarshalIndent(componentsManifest, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to encode cached components manifest: %w", err)
		}
		componentsRaw := append(filteredRaw, '\n')
		if err := m.WriteCachedComponentsManifest(componentsRaw); err != nil {
			return fmt.Errorf("failed to write cached components manifest: %w", err)
		}
	}
	emit(DownloadEvent{Type: DownloadEventStatus, Message: fmt.Sprintf("All dependencies for %s_%s_%s are up to date.", kind, OS, arch)})
	return nil
}

// loadComponentsManifest loads the components manifest from either a local path or the configured deps base URL.
func (m *Manager) loadComponentsManifest(ctx context.Context, client *http.Client, opts DownloadOptions, emit func(DownloadEvent)) (*ComponentsManifest, string, error) {
	componentsSource := fmt.Sprintf("%s/%s", m.baseURL, componentsManifestPath)
	var componentsManifest *ComponentsManifest
	var err error
	if opts.ComponentsManifestPath != "" {
		componentsSource = opts.ComponentsManifestPath
		emit(DownloadEvent{Type: DownloadEventStatus, Message: fmt.Sprintf("Reading components manifest from %s...", opts.ComponentsManifestPath)})
		componentsManifest, err = readComponentsManifestFile(opts.ComponentsManifestPath)
	} else {
		emit(DownloadEvent{Type: DownloadEventStatus, Message: "Fetching components manifest..."})
		componentsManifest, err = m.fetchComponentsManifest(ctx, client)
	}
	if err != nil {
		return nil, componentsSource, fmt.Errorf("failed to load components manifest: %w", err)
	}
	return componentsManifest, componentsSource, nil
}

func downloadPendingMessage(vmconfigPending, componentPending int) string {
	if vmconfigPending == 0 {
		return fmt.Sprintf("Populating component image mirror store with %d images...", componentPending)
	}
	if componentPending == 0 {
		return fmt.Sprintf("Downloading %d vmconfig artifacts...", vmconfigPending)
	}
	return fmt.Sprintf("Downloading %d vmconfig artifacts and populating component image mirror store with %d images...", vmconfigPending, componentPending)
}

func (m *Manager) downloadPending(pending []pendingDownload, concurrency int, client *http.Client, emit func(DownloadEvent)) error {
	if concurrency <= 0 {
		concurrency = defaultDownloadConcurrency
	}
	if concurrency > len(pending) {
		concurrency = len(pending)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	var errMu sync.Mutex
	var firstErr error
	var repoLocksMu sync.Mutex
	repoLocks := map[string]*sync.Mutex{}
	globalSem := make(chan struct{}, concurrency)
	hostSemaphores := registryConcurrencySemaphores(registryConcurrencyOverrides)

	setErr := func(err error) bool {
		errMu.Lock()
		defer errMu.Unlock()
		if firstErr != nil {
			return false
		}
		firstErr = err
		cancel()
		return true
	}
	componentRepoLock := func(image ComponentImage) *sync.Mutex {
		repo := mirrorRepoFromChartImage(image.Image)
		repoLocksMu.Lock()
		defer repoLocksMu.Unlock()
		lock := repoLocks[repo]
		if lock == nil {
			lock = &sync.Mutex{}
			repoLocks[repo] = lock
		}
		return lock
	}
	acquire := func(sem chan struct{}) bool {
		select {
		case sem <- struct{}{}:
			return true
		case <-ctx.Done():
			return false
		}
	}
	release := func(sem chan struct{}) {
		<-sem
	}
	hostSemaphore := func(job pendingDownload) chan struct{} {
		if job.Kind != pendingDownloadComponentImage {
			return nil
		}
		return hostSemaphores[registryHostFromImage(job.Image.Image)]
	}

	for _, job := range pending {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		go func(job pendingDownload) {
			defer wg.Done()

			hostSem := hostSemaphore(job)
			if hostSem != nil {
				if !acquire(hostSem) {
					return
				}
				defer release(hostSem)
			}
			if !acquire(globalSem) {
				return
			}
			defer release(globalSem)

			switch job.Kind {
			case pendingDownloadComponentImage:
				lock := componentRepoLock(job.Image)
				lock.Lock()
				defer lock.Unlock()

				emit(DownloadEvent{Type: DownloadEventStarted, Name: job.Image.Image, Total: job.Image.Size})
				imageProgress := &componentImageProgress{name: job.Image.Image, total: job.Image.Size, emit: emit}
				imageProgress.report()
				err := writeComponentImage(ctx, m.ComponentsImagesCacheDir(), job.Image, imageProgress)
				if err != nil {
					wrapped := fmt.Errorf("failed to download %s: %w", job.Image.Image, err)
					if setErr(wrapped) {
						emit(DownloadEvent{Type: DownloadEventFailed, Name: job.Image.Image, Err: err})
					}
					return
				}
				emit(DownloadEvent{Type: DownloadEventDone, Name: job.Image.Image, Current: imageProgress.current, Total: imageProgress.total})
			default:
				emit(DownloadEvent{Type: DownloadEventStarted, Name: job.Item.Name, Path: job.Dest})
				_, err := filecache.Download(ctx, job.Item.Dep.URL, job.Dest, job.Algo, job.Checksum, filecache.DownloadOptions{
					Client: client,
					Total:  job.Item.Dep.Size,
					Progress: func(current, total int64) {
						emit(DownloadEvent{Type: DownloadEventProgress, Name: job.Item.Name, Path: job.Dest, Current: current, Total: total})
					},
				})
				if err != nil {
					wrapped := fmt.Errorf("failed to download %s: %w", job.Item.Name, err)
					if setErr(wrapped) {
						emit(DownloadEvent{Type: DownloadEventFailed, Name: job.Item.Name, Path: job.Dest, Err: err})
					}
					return
				}
				emit(DownloadEvent{Type: DownloadEventDone, Name: job.Item.Name, Path: job.Dest, Current: job.Item.Dep.Size, Total: job.Item.Dep.Size})
			}
		}(job)
	}
	wg.Wait()

	if firstErr != nil {
		return firstErr
	}
	return nil
}

func registryHostFromImage(image string) string {
	repo, _, _ := splitImageRef(image)
	first := repo
	if slash := strings.Index(repo, "/"); slash >= 0 {
		first = repo[:slash]
	}
	if strings.Contains(first, ".") || strings.Contains(first, ":") || first == "localhost" {
		return first
	}
	return "docker.io"
}

func registryConcurrencySemaphores(overrides map[string]int) map[string]chan struct{} {
	semaphores := make(map[string]chan struct{}, len(overrides))
	for registry, limit := range overrides {
		if limit <= 0 {
			continue
		}
		semaphores[registry] = make(chan struct{}, limit)
	}
	return semaphores
}

// normalizeArchs returns a de-duplicated list of supported target architectures.
func normalizeArchs(values []string, defaultArch string) ([]string, error) {
	if len(values) == 0 {
		values = []string{defaultArch}
	}
	seen := map[string]bool{}
	archs := []string{}
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			arch := strings.TrimSpace(part)
			if arch == "" {
				continue
			}
			switch arch {
			case "amd64", "arm64":
			default:
				return nil, fmt.Errorf("unsupported arch %q", arch)
			}
			if !seen[arch] {
				seen[arch] = true
				archs = append(archs, arch)
			}
		}
	}
	if len(archs) == 0 {
		return nil, fmt.Errorf("at least one arch is required")
	}
	return archs, nil
}
