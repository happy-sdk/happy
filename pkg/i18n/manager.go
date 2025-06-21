// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"sync"

	"github.com/happy-sdk/happy/sdk/logging"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type contextKey string

const (
	langContextKey contextKey = "language"
	loglevel                  = math.MinInt + 1
)

type manager struct {
	mu             sync.RWMutex
	langs          []language.Tag
	defaultLang    language.Tag
	printerCache   map[language.Tag]*message.Printer
	defaultPrinter *message.Printer
	logger         logging.Logger
	loggerDisposed bool
}

func (sl *manager) supports(lang language.Tag) bool {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return slices.Contains(sl.langs, lang)
}

func (sl *manager) support(lang language.Tag) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.langs = append(sl.langs, lang)
}

func (sl *manager) set(lang language.Tag) error {
	if !sl.supports(lang) {
		return fmt.Errorf("i18n(%s): language not supported", lang.String())
	}
	sl.mu.Lock()
	defer sl.mu.Unlock()

	sl.defaultLang = lang
	sl.defaultPrinter = message.NewPrinter(sl.defaultLang, message.Catalog(globalCatalog))
	return nil
}

func (sl *manager) get() language.Tag {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	lang := sl.defaultLang
	return lang
}

func (sl *manager) log(level logging.Level, msg string) {
	if sl.loggerDisposed {
		slog.Log(context.Background(), slog.Level(level), msg)
		return
	}

	sl.logger.LogDepth(0, level, msg)

	ql, ok := sl.logger.(*logging.QueueLogger)
	if ok && ql.Consumed() {
		sl.logger = nil
		sl.loggerDisposed = true
	}
}

func (sl *manager) getDefault() language.Tag {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	lang := sl.defaultLang
	return lang
}

func (sl *manager) getSupported() []language.Tag {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	langs := sl.langs
	return langs
}

func (sl *manager) reload() {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	sl.defaultPrinter = message.NewPrinter(sl.defaultLang, message.Catalog(globalCatalog))
	sl.printerCache[sl.defaultLang] = sl.defaultPrinter
	for lang, _ := range sl.printerCache {
		if lang == sl.defaultLang {
			continue
		}
		sl.printerCache[lang] = message.NewPrinter(lang, message.Catalog(globalCatalog))
	}

}
func (sl *manager) getDefaultPrinter() *message.Printer {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	printer := sl.defaultPrinter
	return printer
}

func (sl *manager) getPrinter(lang language.Tag) (*message.Printer, error) {
	if !sl.supports(lang) {
		return nil, fmt.Errorf("i18n(%s): language not supported", lang.String())
	}
	sl.mu.RLock()
	if printer, exists := sl.printerCache[lang]; exists {
		sl.mu.RUnlock()
		return printer, nil
	}

	sl.mu.RUnlock()
	p := sl.printerCache[lang]
	if p == nil {
		p = message.NewPrinter(lang)
		sl.printerCache[lang] = p
	}
	return p, nil
}
