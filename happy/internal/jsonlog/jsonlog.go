// Copyright 2022 The Happy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jsonlog

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mkungla/happy"
)

type Entry struct {
	Type    string        `json:"type"`
	SS      time.Duration `json:"ss"`
	Message string        `json:"message"`
}

type Logger struct {
	mu      sync.Mutex
	level   happy.LogLevel
	last    time.Time
	logs    []Entry
	started time.Time
	err     string
	res     any
}

type Output struct {
	Time  time.Time `json:"time"`
	Logs  []Entry   `json:"logs,omitempty"`
	Error string    `json:"error,omitempty"`
	Data  any       `json:"data,omitempty"`
}

func New(lvl happy.LogLevel) *Logger {
	if lvl < happy.LevelError {
		lvl = happy.LevelError
	}
	return &Logger{
		level:   lvl,
		last:    time.Now(),
		started: time.Now(),
	}
}

func (l *Logger) SetResponse(response any) {
	if l.res != nil {
		l.Error("json response set multiple times")
		return
	}
	l.res = response
}

func (l *Logger) GetOutput(data any) Output {
	o := Output{
		Time:  l.started,
		Logs:  l.logs,
		Error: l.err,
	}
	if len(o.Error) > 0 && len(o.Logs) == 1 {
		o.Logs = nil
	}
	if len(o.Error) == 0 {
		o.Data = l.res
	}
	return o
}

func (l *Logger) SystemDebug(args ...any) {
	l.write(happy.LevelSystemDebug, args...)
}

func (l *Logger) SystemDebugf(template string, args ...any) {
	l.writef(happy.LevelSystemDebug, template, args...)
}

func (l *Logger) Debug(args ...any) {
	l.write(happy.LevelDebug, args...)
}

func (l *Logger) Debugf(template string, args ...any) {
	l.writef(happy.LevelDebug, template, args...)
}

func (l *Logger) Info(args ...any) {
	l.write(happy.LevelVerbose, args...)
}

func (l *Logger) Infof(template string, args ...any) {
	l.writef(happy.LevelVerbose, template, args...)
}

func (l *Logger) Notice(args ...any) {
	l.write(happy.LevelNotice, args...)
}

func (l *Logger) Noticef(template string, args ...any) {
	l.writef(happy.LevelNotice, template, args...)
}

func (l *Logger) Ok(args ...any) {
	l.write(happy.LevelOk, args...)
}

func (l *Logger) Okf(template string, args ...any) {
	l.writef(happy.LevelOk, template, args...)
}

func (l *Logger) Issue(nr int, args ...any) {
	args = append([]interface{}{fmt.Sprintf("(%d): ", nr)}, args...)
	l.write(happy.LevelIssue, args...)
}

func (l *Logger) Issuef(nr int, template string, args ...any) {
	args = append([]interface{}{fmt.Sprintf("(%d): ", nr)}, args...)
	l.writef(happy.LevelIssue, fmt.Sprintf("%%s%s", template), args...)
}

func (l *Logger) Task(args ...any) {
	l.write(happy.LevelTask, args...)
}

func (l *Logger) Taskf(template string, args ...any) {
	l.writef(happy.LevelTask, template, args...)
}

func (l *Logger) Experimental(args ...any) {
	l.write(happy.LevelWarn, args...)
}

func (l *Logger) Experimentalf(template string, args ...any) {
	l.writef(happy.LevelWarn, template, args...)
}
func (l *Logger) Warn(args ...any) {
	l.write(happy.LevelWarn, args...)
}

func (l *Logger) Warnf(template string, args ...any) {
	l.writef(happy.LevelWarn, template, args...)
}

func (l *Logger) Deprecated(args ...any) {
	l.write(happy.LevelDeprecated, args...)
}

func (l *Logger) Deprecatedf(template string, args ...any) {
	l.writef(happy.LevelDeprecated, template, args...)
}

func (l *Logger) NotImplemented(args ...any) {
	l.write(happy.LevelNotImplemented, args...)
}

func (l *Logger) NotImplementedf(template string, args ...any) {
	l.writef(happy.LevelNotImplemented, template, args...)
}

func (l *Logger) Error(args ...any) {
	l.write(happy.LevelError, args...)
}

func (l *Logger) Errorf(template string, args ...any) {
	l.writef(happy.LevelError, template, args...)
}

func (l *Logger) Critical(args ...any) {
	l.write(happy.LevelCritical, args...)
}

func (l *Logger) Criticalf(template string, args ...any) {
	l.writef(happy.LevelCritical, template, args...)
}

func (l *Logger) Alert(args ...any) {
	l.write(happy.LevelAlert, args...)
}

func (l *Logger) Alertf(template string, args ...any) {
	l.writef(happy.LevelAlert, template, args...)
}

func (l *Logger) Emergency(args ...any) {
	l.write(happy.LevelEmergency, args...)
}

func (l *Logger) Emergencyf(template string, args ...any) {
	l.writef(happy.LevelEmergency, template, args...)
}

func (l *Logger) Out(args ...any) {
	l.write(happy.LevelOut, args...)
}

func (l *Logger) Outf(template string, args ...any) {
	l.writef(happy.LevelOut, template, args...)
}

func (l *Logger) Level() happy.LogLevel {
	return l.level
}

func (l *Logger) SetLevel(lvl happy.LogLevel) {
	if lvl < happy.LevelError {
		return
	}
	l.level = lvl
}

func (l *Logger) Sync() error { return nil }

func (l *Logger) Write(data []byte) (n int, err error) {
	return 0, errors.New("json logger does not support Write")
}

func (l *Logger) write(lvl happy.LogLevel, args ...any) {
	if l.level > lvl {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	durr := now.Sub(l.last)
	l.last = now
	e := Entry{
		Type:    lvl.ShortString(),
		SS:      durr,
		Message: fmt.Sprint(args...),
	}
	l.logs = append(l.logs, e)

	if lvl >= happy.LevelError {
		l.err = e.Message
	}
}

func (l *Logger) writef(lvl happy.LogLevel, template string, args ...any) {
	l.write(lvl, fmt.Sprintf(template, args...))
}
