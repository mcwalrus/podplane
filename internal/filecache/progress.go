// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package filecache

import (
	"io"
	"time"
)

// Progress is a wrapper for an io.Writer that reports progress for
// a file download
type Progress struct {
	Writer       io.Writer
	Total        int64 // response "Content-Length" header
	Current      int64
	LastTime     time.Time
	Interval     time.Duration
	ProgressFunc func(current, total int64)
}

// Write implements the io.Writer interface
func (dp *Progress) Write(p []byte) (n int, err error) {
	// Write if writer specified
	if dp.Writer != nil {
		n, err = dp.Writer.Write(p)
	} else {
		n = len(p)
		err = nil
	}

	// Increment counter
	dp.Current += int64(n)

	// Report progress if function specified
	if dp.ProgressFunc != nil {
		// Only update progress every interval to avoid flooding the terminal
		if time.Since(dp.LastTime) > dp.Interval {
			dp.ProgressFunc(dp.Current, dp.Total)
			dp.LastTime = time.Now()
		}
	}

	return n, err
}

// NewProgress creates a new download progress writer
func NewProgress(writer io.Writer, total int64, interval time.Duration, progressFunc func(current, total int64)) *Progress {
	if interval == 0 {
		interval = 100 * time.Millisecond
	}
	return &Progress{
		Writer:       writer,
		Total:        total,
		Current:      0,
		Interval:     interval,
		ProgressFunc: progressFunc,
	}
}
