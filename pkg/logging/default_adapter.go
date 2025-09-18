// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

package logging

import (
	"io"
	"log/slog"
)

// NewAdapterWithHandler creates an AdapterComposer from a slog.Handler creator.
func NewAdapterWithHandler[H slog.Handler](
	w io.Writer,
	f AdapterComposeHandlerFunc[H],
) *AdapterComposer[*DefaultAdapter] {
	var writer *Writer
	if ww, ok := w.(*Writer); ok {
		writer = ww
	} else {
		writer = NewWriter(w)
	}
	a := &AdapterComposer[*DefaultAdapter]{
		f: func(writer *Writer, config Config) *DefaultAdapter {
			return &DefaultAdapter{Handler: f(writer, config.HandlerOptions())}
		},
		w: writer,
	}

	a.dispose = func() error {
		return writer.Close()
	}

	return a
}

// NewTextAdapter creates an AdapterComposer for a slog.TextHandler.
func NewTextAdapter(w io.Writer) *AdapterComposer[*DefaultAdapter] {
	return NewAdapterWithHandler(w, slog.NewTextHandler)
}

// NewJSONAdapter creates an AdapterComposer for a slog.JSONHandler.
func NewJSONAdapter(w io.Writer) *AdapterComposer[*DefaultAdapter] {
	return NewAdapterWithHandler(w, slog.NewJSONHandler)
}

// DefaultAdapter wraps a slog.Handler to implement the Adapter interface.
type DefaultAdapter struct {
	slog.Handler
}

// Dispose closes the wrapped handler if it implements DisposableAdapter
func (d *DefaultAdapter) Dispose() error {
	if disposable, ok := d.Handler.(DisposableAdapter); ok {
		return disposable.Dispose()
	}
	return nil
}

// Flush flushes the wrapped handler if it implements FlushableAdapter.
func (d *DefaultAdapter) Flush() error {
	if flushable, ok := d.Handler.(FlushableAdapter); ok {
		return flushable.Flush()
	}
	return nil
}

// Ready signals readiness if the wrapped handler implements AdapterReady.
func (d *DefaultAdapter) Ready() {
	if ready, ok := d.Handler.(AdapterReady); ok {
		ready.Ready()
	}
}

// BatchHandler processes a batch if the wrapped handler implements AdapterWithBatchHandler.
func (d *DefaultAdapter) BatchHandle(records []Record) error {
	if ready, ok := d.Handler.(AdapterWithBatchHandle); ok {
		return ready.BatchHandle(records)
	}
	return nil
}
