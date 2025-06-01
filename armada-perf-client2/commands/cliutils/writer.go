
package cliutils

import (
	"os"
	"sync"
	"text/tabwriter"
)

// LockedTabWriter is a wrapper around tabwriter to make it thread safe
type LockedTabWriter struct {
	out *tabwriter.Writer
	mu  sync.Mutex
}

// NewWriter creates a new tabw aligned Writer
func NewWriter() *LockedTabWriter {
	return &LockedTabWriter{out: tabwriter.NewWriter(os.Stdout, 0, 8, 3, ' ', tabwriter.StripEscape)}
}

// Write writes buf to the writer w.
func (w *LockedTabWriter) Write(buf []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.out.Write(buf)
}

// Flush should be called after the last call to Write to ensure
// that any data buffered in the Writer is written to the output.
func (w *LockedTabWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.out.Flush()
}
