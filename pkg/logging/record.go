// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

package logging

import (
	"context"
	"log/slog"
	"time"

	"github.com/happy-sdk/happy/pkg/logging/internal"
)

type Record struct {
	isHTTP bool
	http   *internal.HttpRecord
	Ctx    context.Context
	Record slog.Record
}

// SetHTTP sets the HTTP record from a slog.Attr.
func (r *Record) setHTTP(a slog.Attr) {
	if val, ok := a.Value.Any().(internal.HttpRecord); ok {
		r.http = &val
		r.isHTTP = true
	}
}

// HttpRecord is special record type passed to Adapters implementing
// either AdapterWithHTTPHandle or AdapterWithHTTPBatchHandle
type HttpRecord struct {
	Ctx    context.Context
	Record slog.Record
	Method string
	Code   int
	Path   string
}

// NewHttpRecord returns special slog.Record
// to log http logs. This record is passed ONLY to Adapters acepting it
// otherwise record is dropped without Handlerer or Adapter error
func NewHttpRecord(t time.Time, method string, statusCode int, path string, args ...any) slog.Record {
	return internal.NewHttpRecord(t, method, statusCode, path, args...)
}
