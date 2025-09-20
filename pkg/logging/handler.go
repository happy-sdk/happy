// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

package logging

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// handlerState holds the immutable handler configuration
type handlerState struct {
	config    Config
	adapters  []*adapter
	processor *attrProcessor
	disposed  bool
}

// handler manages multiple adapters internally
type handler struct {
	// Atomic pointer for lock-free reads in hot path
	state atomic.Pointer[handlerState]

	// Mutex only for mutations (rarely used)
	mu sync.Mutex

	// Fast path optimization
	level *slog.LevelVar

	// Resource management
	disposed  atomic.Bool
	readyOnce sync.Once
	closeOnce sync.Once
}

func newHandler(config Config, adapters []*adapter) *handler {
	h := &handler{}

	state := &handlerState{
		config:    config,
		adapters:  adapters,
		processor: newAttrProcessor(config),
		disposed:  false,
	}

	h.state.Store(state)
	h.level = config.lvl

	// Notify adapters they're ready
	for _, a := range adapters {
		if ready, ok := a.handler.(AdapterReady); ok {
			ready.Ready()
		}
	}

	return h
}

// Enabled uses atomic operations for zero-allocation fast path
func (h *handler) Enabled(ctx context.Context, level slog.Level) bool {
	if h.disposed.Load() {
		return false
	}
	return level >= h.level.Level()
}

// Handle optimized for the common case (no errors)
func (h *handler) Handle(ctx context.Context, record slog.Record) error {
	if h.disposed.Load() {
		return ErrLoggerDisposed
	}
	state := h.state.Load()

	// Process attributes once
	pr := state.processor.process(ctx, record)
	return h.handle(state, pr)
}

func (h *handler) handle(state *handlerState, pr *Record) error {
	var firstError error
	errorCount := 0

	for _, adapter := range state.adapters {
		if !adapter.Enabled(pr.Ctx, pr.Record.Level) {
			continue
		}
		if err := adapter.handle(*pr); err != nil { // Dereference pr for adapter.handle
			errorCount++
			if firstError == nil {
				firstError = err
			}
		}
	}

	if errorCount == 0 {
		return nil
	}
	if errorCount == 1 {
		return firstError
	}

	return h.collectErrors(state.adapters)
}

func (h *handler) queueHandle(records []Record) (int, error) {
	if h.disposed.Load() {
		return 0, ErrLoggerDisposed
	}

	state := h.state.Load()

	var firstError error
	errorCount := 0
	prCount := 0
	for _, src := range records {
		pr := state.processor.process(src.Ctx, src.Record)
		if err := h.handle(state, pr); err != nil {
			errorCount++
			if firstError == nil {
				firstError = err
			}
		}
		prCount++
	}
	return prCount, firstError
}

func (h *handler) collectErrors(adapters []*adapter) error {
	var errs []error
	for _, adapter := range adapters {
		if err := adapter.Err(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// WithAttrs creates derived handlers without
func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if h.disposed.Load() {
		return DiscardAdapter
	}
	return newDerivedAdapter(h, "", attrs)
}

// WithGroup creates derived handlers
func (h *handler) WithGroup(name string) slog.Handler {
	if h.disposed.Load() {
		return DiscardAdapter
	}
	return newDerivedAdapter(h, name, nil)
}

func (h *handler) Dispose() error {
	if h.disposed.Swap(true) {
		return nil
	}

	var errs []error
	h.closeOnce.Do(func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		state := h.state.Load()

		// Flush first
		if err := h.flush(state); err != nil && !errors.Is(err, ErrAdapterDisposed) {
			errs = append(errs, err)
		}

		// Dispose adapters
		for _, adapter := range state.adapters {
			if err := adapter.Dispose(); err != nil {
				errs = append(errs, err)
			}
		}

		// Update state
		newState := &handlerState{
			config:    state.config,
			adapters:  nil,
			processor: state.processor,
			disposed:  true,
		}
		h.state.Store(newState)
	})

	return errors.Join(errs...)
}

func (h *handler) Flush() error {
	if h.disposed.Load() {
		return ErrLoggerDisposed
	}
	state := h.state.Load()

	h.mu.Lock()
	defer h.mu.Unlock()

	return h.flush(state)
}

func (h *handler) flush(state *handlerState) error {
	var errs []error

	for _, adapter := range state.adapters {
		if err := adapter.Flush(); err != nil {
			errs = append(errs, err)
		}
	}

	// Sleep no longer than second to wait writers
	time.Sleep(min(h.state.Load().config.Adapter.FlushInterval, time.Second))

	return errors.Join(errs...)
}

func (h *handler) Ready() {
	h.readyOnce.Do(func() {
		state := h.state.Load()
		for _, a := range state.adapters {
			if a, ok := a.handler.(AdapterReady); ok {
				a.Ready()
			}
		}
	})
}
