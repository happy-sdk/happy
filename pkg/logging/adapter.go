// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

package logging

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/happy-sdk/happy/pkg/logging/internal"
)

// Adapter implements slog.Handler
type Adapter interface {
	slog.Handler
}

// DisposableAdapter is an Adapter that can be closed to release resources.
type DisposableAdapter interface {
	Adapter
	// Dispose closes the adapter and releases resources. Idempotent.
	Dispose() error
}

// FlushableAdapter is an Adapter that can flush buffered log entries.
type FlushableAdapter interface {
	Adapter
	// Flush writes any buffered log entries to the underlying writer.
	Flush() error
}

// AdapterReady is an Adapter that needs signaling when it is initialized and attached.
type AdapterReady interface {
	// Logger calls Ready to signal that the adapter is initialized and attached to a logger.
	Ready()
}

// AdapterWithBatchHandle processes bursts of log records efficiently.
type AdapterWithBatchHandle interface {
	// BatchHandle processes a batch of log records.
	BatchHandle(records []Record) error
}

// AdapterWithHTTPHandle defines adapters handling HTTP records.
type AdapterWithHTTPHandle interface {
	// HTTP handles an HTTP-specific log record.
	HTTP(ctx context.Context, method string, statusCode int, path string, r slog.Record) error
}

// AdapterWithHTTPBatchHandle defines adapters batch-handling HTTP records.
type AdapterWithHTTPBatchHandle interface {
	// HTTPBatchHandle processes a batch of HTTP records.
	HTTPBatchHandle(records []HttpRecord) error
}

// AdapterDisposeFunc closes an Adapter and releases its resources.
type AdapterDisposeFunc func() error

// AdapterComposeFunc creates an Adapter from a writer and configuration.
type AdapterComposeFunc[A Adapter] func(writer *Writer, config Config) A

// AdapterComposeHandlerFunc creates a slog.Handler from a writer and options.
type AdapterComposeHandlerFunc[H slog.Handler] func(writer io.Writer, opts *slog.HandlerOptions) H

// ComposableAdapter is an Adapter that must be composed with logger config.
type ComposableAdapter interface {
	// Compose creates an Adapter using the provided configuration.
	Compose(Config) Adapter
}

// AdapterError wraps errors for storage in atomic.Value by Adapters.
type AdapterError struct {
	err error
}

// Err returns the wrapped error.
func (e *AdapterError) Err() error {
	return e.err
}

// Error returns the string representation of the wrapped error.
func (e *AdapterError) Error() string {
	return e.err.Error()
}

// NewAdapterError wraps an error for use in an Adapter, avoiding double-wrapping.
func NewAdapterError(err error) *AdapterError {
	if aerr, ok := err.(*AdapterError); ok {
		return aerr
	} else if errors.Is(err, ErrAdapter) {
		return &AdapterError{err}
	}
	return &AdapterError{fmt.Errorf("%w: %w", ErrAdapter, err)}
}

// ReplaceAdaptersStdout replaces stdout writers for adapters using os.Stdout.
//
// This must be called BEFORE any system-level file descriptor duplication (e.g. unix.Dup2(fd, 1)),
// as the method checks the original stdout to determine which adapters need updating.
//
// Returns ErrAdapterSwappingOutput if no adapters using os.Stdout are found,
// which typically indicates the method was called after system-level FD duplication.
// This method never fails due to writer close errors since os.Stdout is never
// actually closed during the swap operation.
//
// Designed for hot-swapping stdout in daemonized applications (e.g., log rotation).
// The operation is thread-safe as it locks the logger's shared handler.
func ReplaceAdaptersStdout(l *Logger, w io.Writer) error {
	l.handler.mu.Lock()
	defer l.handler.mu.Unlock()

	var found bool
	state := l.handler.state.Load()
	for _, a := range state.adapters {
		if a.w == nil || a.w.Get() != os.Stdout {
			continue
		}
		_ = a.replaceWriter(w) // Can't fail for os.Stdout
		found = true
	}

	if !found {
		return fmt.Errorf("%w: no adapters found using os.Stdout", ErrAdapterSwappingOutput)
	}
	return nil
}

// ReplaceAdaptersStderr replaces stderr writers for adapters using os.Stderr.
//
// This must be called BEFORE any system-level file descriptor duplication (e.g. unix.Dup2(fd, 2)),
// as the method checks the original stderr to determine which adapters need updating.
//
// Returns ErrAdapterSwappingOutput if no adapters using os.Stderr are found,
// which typically indicates the method was called after system-level FD duplication.
// This method never fails due to writer close errors since os.Stderr is never
// actually closed during the swap operation.
//
// Designed for hot-swapping stderr in daemonized applications (e.g., log rotation).
// The operation is thread-safe as it locks the logger's handler.
func ReplaceAdaptersStderr(l *Logger, w io.Writer) error {
	l.handler.mu.Lock()
	defer l.handler.mu.Unlock()

	var found bool
	state := l.handler.state.Load()
	for _, a := range state.adapters {
		if a.w == nil || a.w.Get() != os.Stderr {
			continue
		}
		_ = a.replaceWriter(w) // Can't fail for os.Stderr
		found = true
	}

	if !found {
		return fmt.Errorf("%w: no adapters found using os.Stderr", ErrAdapterSwappingOutput)
	}
	return nil
}

// GetAdaptersFromHandler retrieves Adapters of type T from a slog.Handler.
func GetAdaptersFromHandler[T Adapter](h slog.Handler) []T {
	if h == nil {
		return nil
	}
	var adapters []T

	if h, ok := h.(*handler); ok {
		h.mu.Lock()
		defer h.mu.Unlock()
		state := h.state.Load()
		for _, a := range state.adapters {
			if bufAdapter, ok := a.handler.(*BufferedAdapter[T]); ok {
				adapters = append(adapters, bufAdapter.adapter.(T))
			} else if adapter, ok := a.handler.(T); ok {
				adapters = append(adapters, adapter)
			}
		}
	}

	return adapters
}

// AdapterComposer composes an Adapter from a handler and logger configuration.
type AdapterComposer[A Adapter] struct {
	w       *Writer
	f       AdapterComposeFunc[A]
	dispose AdapterDisposeFunc
}

// NewAdapter creates an AdapterComposer for the specified Adapter type.
func NewAdapter[A Adapter](w io.Writer, f AdapterComposeFunc[A]) *AdapterComposer[A] {
	var writer *Writer
	if ww, ok := w.(*Writer); ok {
		writer = ww
	} else {
		writer = NewWriter(w)
	}
	a := &AdapterComposer[A]{
		f: f,
		w: writer,
	}

	a.dispose = func() error {
		return writer.Close()
	}

	return a
}

// Enabled always returns false until the adapter is composed.
func (c *AdapterComposer[A]) Enabled(ctx context.Context, l slog.Level) bool {
	return false
}

// Handle returns always ErrAdapterNotComposed.
func (c *AdapterComposer[A]) Handle(ctx context.Context, record slog.Record) error {
	return ErrAdapterNotComposed
}

// WithAttrs returns the composer unchanged.
func (c *AdapterComposer[A]) WithAttrs(attrs []slog.Attr) slog.Handler {
	return c
}

// WithGroup returns the composer unchanged.
func (c *AdapterComposer[A]) WithGroup(name string) slog.Handler {
	return c
}

// Err returns always ErrAdapterNotComposed.
func (c *AdapterComposer[A]) Err() error {
	return ErrAdapterNotComposed
}

// Compose creates an Adapter using the provided configuration.
func (c *AdapterComposer[A]) Compose(config Config) Adapter {
	if c == nil || c.f == nil {
		return nil
	}
	var handler slog.Handler = c.f(c.w, config)
	if handler == nil || reflect.ValueOf(handler).IsNil() {
		return nil
	}
	// Check if handler itself is disposable
	if hd, ok := handler.(DisposableAdapter); ok {
		dispose := func() error {
			var errs []error
			if err := hd.Dispose(); err != nil {
				errs = append(errs, err)
			}
			if c.dispose != nil {
				if err := c.dispose(); err != nil {
					errs = append(errs, err)
				}
			}
			return errors.Join(errs...)
		}
		return newDisposableAdapter(c.w, handler, dispose)
	} else if c.dispose != nil {
		return newDisposableAdapter(c.w, handler, c.dispose)
	}

	return newAdapter(c.w, handler)
}

// adapter wraps a slog.Handler with additional state for concurrency and disposal.
type adapter struct {
	mu          *sync.RWMutex
	w           *Writer
	handler     slog.Handler
	adapterName string

	err atomic.Value

	disposed    atomic.Bool
	disposable  bool
	disposeFunc AdapterDisposeFunc

	acceptHttpRecords bool
}

// newAdapter creates an adapter wrapping a slog.Handler.
func newAdapter(w *Writer, h slog.Handler) *adapter {
	a := &adapter{
		mu:      &sync.RWMutex{},
		handler: h,
		w:       w,
	}

	switch hh := h.(type) {
	case *DefaultAdapter:
		a.adapterName = fmt.Sprintf("%T", hh.Handler)
		_, a.acceptHttpRecords = hh.Handler.(AdapterWithHTTPHandle)
		if !a.acceptHttpRecords {
			_, a.acceptHttpRecords = hh.Handler.(AdapterWithHTTPBatchHandle)
		}
	case internal.BufferedAdapter[Record]:
		a.adapterName = hh.GetBufferedAdapterName()
		a.acceptHttpRecords = hh.AcceptsHTTP()
	default:
		a.adapterName = fmt.Sprintf("%T", hh)
		_, a.acceptHttpRecords = hh.(AdapterWithHTTPHandle)
		if !a.acceptHttpRecords {
			_, a.acceptHttpRecords = hh.(AdapterWithHTTPBatchHandle)
		}
	}

	return a
}

// newDisposableAdapter creates an adapter with a dispose function.
func newDisposableAdapter(w *Writer, h slog.Handler, disposeFunc AdapterDisposeFunc) *adapter {
	adapter := newAdapter(w, h)
	adapter.disposable = true
	adapter.disposeFunc = disposeFunc
	return adapter
}

// Enabled checks if the adapter handles records at the given level.
func (a *adapter) Enabled(ctx context.Context, level slog.Level) bool {
	if a.disposed.Load() {
		return false
	}
	return a.handler.Enabled(ctx, level)
}

// Dispose closes the adapter and releases resources. Idempotent.
func (a *adapter) Dispose() error {
	if a.disposed.Swap(true) {
		return a.Err()
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	var err error
	if e := a.Err(); e != nil {
		err = e
	}
	if !a.disposable || a.disposeFunc == nil {
		if err != nil {
			return fmt.Errorf("%w(%T): %w", ErrAdapter, a.handler, a.Err())
		}
		return nil
	}
	if e := a.disposeFunc(); e != nil {
		err = errors.Join(err, e)
	}
	if err != nil {
		return fmt.Errorf("%w(%T): %w", ErrAdapter, a.handler, err)
	}
	return nil
}

// Err returns the last error encountered by the adapter, or nil if none.
func (a *adapter) Err() error {
	if err, ok := a.err.Load().(error); ok {
		return err
	}
	return nil
}

// Flush writes any buffered log entries to the underlying writer.
func (a *adapter) Flush() (err error) {
	if a.disposed.Load() {
		return ErrAdapterDisposed
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	if _, ok := a.handler.(discardAdapter); ok {
		return nil
	}

	if a, ok := a.handler.(FlushableAdapter); ok {
		err = a.Flush()
	}

	if a.w != nil {
		if syncable, ok := a.w.Get().(interface{ Sync() error }); ok && (syncable != os.Stdout && syncable != os.Stderr) {
			err = errors.Join(err, syncable.Sync())
		}
	}

	return err
}

// Handle processes a log record, dispatching it to the wrapped handler.
func (a *adapter) Handle(ctx context.Context, record slog.Record) error {
	if a.disposed.Load() {
		return nil
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	if err := a.handler.Handle(ctx, record); err != nil {
		e := fmt.Errorf("[%s] %w", a.adapterName, err)
		ee := NewAdapterError(e)
		a.err.Store(ee)
		return e
	}
	return nil
}

func (a *adapter) handle(record Record) error {
	if record.isHTTP {
		return a.http(record)
	}
	return a.Handle(record.Ctx, record.Record)
}

func (a *adapter) http(record Record) error {
	switch aWithHTTP := a.handler.(type) {
	case AdapterWithHTTPHandle:
		return aWithHTTP.HTTP(
			record.Ctx,
			record.http.Method,
			record.http.Code,
			record.http.Path,
			record.Record,
		)
	case internal.BufferedAdapter[Record]:
		if aWithHTTP.AcceptsHTTP() {
			return aWithHTTP.RecordHandle(record)
		}
		return nil
	default:
		return nil
	}
}

// WithAttrs returns a new handler with additional attributes.
func (a *adapter) WithAttrs(attrs []slog.Attr) slog.Handler {
	if a.disposed.Load() {
		return DiscardAdapter
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.handler.WithAttrs(attrs)
}

// WithGroup returns a new handler with the specified group name.
func (a *adapter) WithGroup(name string) slog.Handler {
	if a.disposed.Load() {
		return DiscardAdapter
	}

	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.handler.WithGroup(name)
}

// replaceWriter swaps the adapter's writer with a new one.
func (a *adapter) replaceWriter(neww io.Writer) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.w.Swap(neww)
}

// DiscardAdapter discards all log output and always returns false for Enabled.
var DiscardAdapter Adapter = discardAdapter{}

// discardAdapter efficiently discards all log output
type discardAdapter struct{}

func (discardAdapter) Enabled(context.Context, slog.Level) bool  { return false }
func (discardAdapter) Handle(context.Context, slog.Record) error { return nil }
func (d discardAdapter) WithAttrs([]slog.Attr) slog.Handler      { return d }
func (d discardAdapter) WithGroup(string) slog.Handler           { return d }

// derivedAdapter efficiently handles WithAttrs/WithGroup without new adapters.
type derivedAdapter struct {
	root  *handler
	attrs []slog.Attr
	group string

	// Cached derived adapters (created lazily)
	derivedAdapters []slog.Handler
}

// newDerivedAdapter creates a derived adapter with group and attributes.
func newDerivedAdapter(root *handler, group string, attrs []slog.Attr) Adapter {
	state := root.state.Load()
	d := &derivedAdapter{
		root:  root,
		group: group,
		attrs: attrs,
	}
	d.derivedAdapters = make([]slog.Handler, len(state.adapters))
	for i, adapter := range state.adapters {
		hasGroup, hasAttr := d.group != "", len(d.attrs) > 0
		handler := adapter.handler
		if hasGroup && hasAttr {
			handler = adapter.WithGroup(d.group).WithAttrs(d.attrs)
		} else if hasGroup {
			handler = adapter.WithGroup(d.group)
		} else if hasAttr {
			handler = adapter.WithAttrs(d.attrs)
		}
		d.derivedAdapters[i] = handler
	}

	return d
}

// Enabled checks if the root handler enables the given level.
func (d *derivedAdapter) Enabled(ctx context.Context, level slog.Level) bool {
	return d.root.Enabled(ctx, level)
}

// Handle processes a log record with added attributes and group.
func (d *derivedAdapter) Handle(ctx context.Context, record slog.Record) error {
	state := d.root.state.Load()
	if state.disposed {
		return ErrAdapterDisposed
	}

	// Process record
	pr := state.processor.process(ctx, record)
	pr.Record.AddAttrs(d.attrs...)

	// Handle with derived adapters
	var errs []error

	for _, handler := range d.derivedAdapters {
		if pr.isHTTP {
			if httpAdapter, ok := handler.(AdapterWithHTTPHandle); ok {
				if err := httpAdapter.HTTP(ctx, pr.http.Method, pr.http.Code, pr.http.Path, pr.Record); err != nil {
					errs = append(errs, err)
				}
			} else if httpAdapter, ok := handler.(AdapterWithHTTPBatchHandle); ok {
				if err := httpAdapter.HTTPBatchHandle([]HttpRecord{
					{Ctx: ctx, Record: pr.Record, Method: pr.http.Method, Code: pr.http.Code, Path: pr.http.Path},
				}); err != nil {
					errs = append(errs, err)
				}
			}
			continue
		}
		if err := handler.Handle(ctx, pr.Record); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// WithAttrs returns a new derived adapter with additional attributes.
func (d *derivedAdapter) WithAttrs(attrs []slog.Attr) slog.Handler {
	state := d.root.state.Load()
	if state.disposed {
		return DiscardAdapter
	}
	if len(attrs) == 0 {
		return d
	}
	attrs = state.processor.processAttrs(attrs)

	// Chain attributes
	combinedAttrs := make([]slog.Attr, 0, len(d.attrs)+len(attrs))
	combinedAttrs = append(combinedAttrs, d.attrs...)
	combinedAttrs = append(combinedAttrs, attrs...)

	return newDerivedAdapter(
		d.root,
		d.group,
		combinedAttrs,
	)
}

// WithGroup returns a new derived adapter with the specified group name.
func (d *derivedAdapter) WithGroup(name string) slog.Handler {
	if name == "" {
		return d
	}
	state := d.root.state.Load()
	if state.disposed {
		return DiscardAdapter
	}

	return newDerivedAdapter(
		d.root,
		d.group+"."+name,
		d.attrs,
	)
}
