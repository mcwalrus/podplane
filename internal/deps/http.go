// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package deps

import "net/http"

func newDownloadHTTPClient(concurrency int) *http.Client {
	if concurrency <= 0 {
		concurrency = defaultDownloadConcurrency
	}
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return http.DefaultClient
	}
	clone := transport.Clone()
	clone.ForceAttemptHTTP2 = true
	clone.MaxIdleConnsPerHost = concurrency
	clone.MaxConnsPerHost = concurrency
	return &http.Client{Transport: clone}
}
