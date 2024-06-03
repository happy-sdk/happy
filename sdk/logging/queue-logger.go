// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package logging

import (
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"
)

type QueueLogger struct {
	mu      sync.Mutex
	records []QueueRecord
}

func NewQueueLogger() *QueueLogger {
	return &QueueLogger{}
}

func (l *QueueLogger) Debug(msg string, attrs ...slog.Attr) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, NewQueueRecord(LevelDebug, msg, 3, attrs...))
}

func (l *QueueLogger) Info(msg string, attrs ...slog.Attr) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, NewQueueRecord(LevelInfo, msg, 3, attrs...))
}

func (l *QueueLogger) Ok(msg string, attrs ...slog.Attr) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, NewQueueRecord(LevelOk, msg, 3, attrs...))
}

func (l *QueueLogger) Notice(msg string, attrs ...slog.Attr) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, NewQueueRecord(LevelNotice, msg, 3, attrs...))
}

func (l *QueueLogger) Warn(msg string, attrs ...slog.Attr) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, NewQueueRecord(LevelWarn, msg, 3, attrs...))
}

func (l *QueueLogger) NotImplemented(msg string, attrs ...slog.Attr) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, NewQueueRecord(LevelNotImplemented, msg, 3, attrs...))
}

func (l *QueueLogger) Deprecated(msg string, attrs ...slog.Attr) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, NewQueueRecord(LevelDeprecated, msg, 3, attrs...))
}

func (l *QueueLogger) Error(msg string, attrs ...slog.Attr) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, NewQueueRecord(LevelError, msg, 3, attrs...))
}

func (l *QueueLogger) BUG(msg string, attrs ...slog.Attr) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, NewQueueRecord(LevelBUG, msg, 3, attrs...))
}

func (l *QueueLogger) Println(msg string, attrs ...slog.Attr) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, NewQueueRecord(LevelAlways, msg, 3, attrs...))
}

func (l *QueueLogger) Printf(format string, v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, NewQueueRecord(LevelAlways, fmt.Sprintf(format, v...), 3))
}

func (l *QueueLogger) HTTP(status int, method, path string, attrs ...slog.Attr) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, NewQueueRecord(LevelAlways, fmt.Sprintf("%d %s %s", status, method, path), 3, attrs...))
}

func (l *QueueLogger) Enabled(lvl Level) bool {
	return true
}

func (l *QueueLogger) Level() Level {
	return levelHappy
}

func (l *QueueLogger) SetLevel(lvl Level) {
	l.NotImplemented("QueueLogger.SetLevel(lvl) is not implemented")
}

func (l *QueueLogger) LogDepth(depth int, lvl Level, msg string, attrs ...slog.Attr) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, NewQueueRecord(lvl, msg, depth+3, attrs...))
}

// Handle
func (l *QueueLogger) Handle(r slog.Record) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	record := &QueueRecord{
		pc:  r.PC,
		lvl: Level(r.Level),
		ts:  r.Time,
		msg: r.Message,
	}

	r.Attrs(func(a slog.Attr) bool {
		record.attrs = append(record.attrs, a)
		return true
	})
	l.records = append(l.records, *record)
	return nil
}

func (l *QueueLogger) Logger() *slog.Logger {
	l.NotImplemented("QueueLogger.Logger() is not implemented")
	return nil
}

func (l *QueueLogger) ConsumeQueue(queue *QueueLogger) error {
	if queue == nil || l == queue {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, queue.Consume()...)
	return nil
}

func (l *QueueLogger) Consume() []QueueRecord {
	l.mu.Lock()
	defer l.mu.Unlock()

	records := l.records
	l.records = nil
	return records
}

type QueueRecord struct {
	pc    uintptr
	lvl   Level
	ts    time.Time
	msg   string
	attrs []slog.Attr
}

func NewQueueRecord(lvl Level, msg string, detph int, attrs ...slog.Attr) QueueRecord {
	var pcs [1]uintptr
	runtime.Callers(detph, pcs[:])

	return QueueRecord{
		lvl:   lvl,
		ts:    time.Now(),
		msg:   msg,
		attrs: attrs,
		pc:    pcs[0],
	}
}

func (qr QueueRecord) Record(loc *time.Location) slog.Record {
	r := slog.NewRecord(qr.ts.In(loc), slog.Level(qr.lvl), qr.msg, qr.pc)
	r.AddAttrs(qr.attrs...)
	return r
}
