// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

package logging

import (
	"context"
	"log/slog"
	"sync"

	"github.com/happy-sdk/happy/pkg/logging/internal"
)

// attrProcessor caches attribute processing logic
type attrProcessor struct {
	config Config
	pool   sync.Pool
}

func newAttrProcessor(config Config) *attrProcessor {
	return &attrProcessor{
		config: config,
		pool: sync.Pool{
			New: func() any {
				return make([]slog.Attr, 0, config.AttrProcessorPoolSize)
			},
		},
	}
}

type attrFrame struct {
	groups    []string
	attrs     []slog.Attr
	processed []any
	key       string
	index     int
}

func applyReplaceIter(
	initGroups []string,
	initAttr slog.Attr,
	replace func([]string, slog.Attr) slog.Attr,
) slog.Attr {
	if replace == nil {
		return initAttr
	}

	a := replace(initGroups, initAttr)
	if a.Key == "" || a.Value.Kind() != slog.KindGroup {
		return a
	}

	var stack []attrFrame
	stack = append(stack, attrFrame{
		groups:    append(initGroups, a.Key),
		attrs:     a.Value.Group(),
		processed: make([]any, 0, len(a.Value.Group())),
		key:       a.Key,
		index:     0,
	})

	var result slog.Attr
	for len(stack) > 0 {
		currentIdx := len(stack) - 1
		current := &stack[currentIdx]
		if current.index < len(current.attrs) {
			child := current.attrs[current.index]
			current.index++
			child = replace(current.groups, child)
			if child.Key == "" {
				continue
			}
			if child.Value.Kind() != slog.KindGroup {
				current.processed = append(current.processed, child)
				continue
			}
			stack = append(stack, attrFrame{
				groups:    append(current.groups, child.Key),
				attrs:     child.Value.Group(),
				processed: make([]any, 0, len(child.Value.Group())),
				key:       child.Key,
				index:     0,
			})
			continue
		}
		groupAttr := slog.Group(current.key, current.processed...)
		stack = stack[:len(stack)-1]
		if len(stack) == 0 {
			result = groupAttr
			break
		}
		parent := &stack[len(stack)-1]
		parent.processed = append(parent.processed, groupAttr)
	}
	return result
}

func (p *attrProcessor) processAttrs(in []slog.Attr) []slog.Attr {
	var attrs []slog.Attr
	for _, a := range in {
		a = applyReplaceIter([]string{}, a, p.config.replaceAttr)
		if a.Key != "" {
			attrs = append(attrs, a)
		}
	}
	return attrs
}

func (p *attrProcessor) process(ctx context.Context, src slog.Record) (r Record) {

	r.Ctx = ctx
	r.Record.Time = src.Time
	r.Record.Level = src.Level
	r.Record.Message = src.Message
	r.Record.PC = src.PC

	if src.NumAttrs() == 0 {
		return
	}

	attrs := p.pool.Get().([]slog.Attr)
	defer func() {
		attrs = attrs[:0]
		p.pool.Put(attrs)
	}()

	src.Attrs(func(a slog.Attr) bool {
		if a.Key == internal.HttpRecordKey {
			r.setHTTP(a)
			return true
		}
		a = applyReplaceIter([]string{}, a, p.config.replaceAttr)
		if a.Key != "" {
			attrs = append(attrs, a)
		}
		return true
	})

	r.Record.AddAttrs(attrs...)
	return
}
