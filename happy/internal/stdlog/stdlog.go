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

package stdlog

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"

	"github.com/logrusorgru/aurora/v3"
	"github.com/mkungla/happy"
)

type Logger struct {
	mu         sync.Mutex
	level      happy.LogLevel
	writer     io.Writer
	isTerminal bool
	last       time.Time
}

func New() *Logger {
	return &Logger{
		writer:     os.Stdout,
		isTerminal: term.IsTerminal(0),
		last:       time.Now(),
	}
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
	l.level = lvl
}

func (l *Logger) Sync() error { return nil }

func (l *Logger) Write(data []byte) (n int, err error) {
	return l.writer.Write(data)
}

func (l *Logger) write(lvl happy.LogLevel, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	durr := now.Sub(l.last)
	l.last = now

	prefix, plen := colorPrefix(lvl)
	suffix := fmt.Sprintf(" +%s %s\n", durr.String(), now.Format("15:04:05.000000"))

	var entry bytes.Buffer
	entry.WriteString(prefix + " ")

	if !l.isTerminal {
		entry.WriteString(fmt.Sprint(args...))
		entry.WriteString(aurora.Gray(16-1, suffix).String())
	} else {
		width, _, err := term.GetSize(0)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		linelen := width - plen - len(suffix) + 1
		fullmsg := fmt.Sprint(args...)

		if strings.Contains(fullmsg, "\n") {
			entry.WriteString(strings.Repeat("-", linelen))
			entry.WriteString(aurora.Gray(16-1, suffix).String())

			entry.WriteString(fullmsg)
			if !strings.HasSuffix(fullmsg, "\n") {
				entry.WriteRune('\n')
			}
		} else if len(fullmsg) <= linelen {
			entry.WriteString(fullmsg)
			entry.WriteString(strings.Repeat(" ", linelen-len(fullmsg)))
			entry.WriteString(aurora.Gray(16-1, suffix).String())
		}
	}

	if _, err := l.Write(entry.Bytes()); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func (l *Logger) writef(lvl happy.LogLevel, template string, args ...any) {
	l.write(lvl, fmt.Sprintf(template, args...))
}

func colorPrefix(lvl happy.LogLevel) (string, int) {
	pfx := fmt.Sprintf(" %-8s", lvl.ShortString())
	plen := len(pfx)

	switch lvl {
	case happy.LevelSystemDebug, happy.LevelDebug:
		return aurora.Gray(15, pfx).BgGray(3).Bold().String(), plen
	case happy.LevelVerbose:
		return aurora.White(pfx).BgBrightBlue().Bold().String(), plen
	case happy.LevelNotice:
		return aurora.White(pfx).BgBlue().Bold().String(), plen
	case happy.LevelOk:
		return aurora.White(pfx).BgGreen().Bold().String(), plen
	case happy.LevelIssue:
		return aurora.White(pfx).BgBrightBlack().Bold().String(), plen
	case happy.LevelTask:
		return aurora.White(pfx).BgBlack().Bold().String(), plen
	case happy.LevelWarn:
		return aurora.Black(pfx).BgYellow().Bold().String(), plen
	case happy.LevelDeprecated, happy.LevelNotImplemented:
		return aurora.Black(pfx).BgBrightYellow().Bold().String(), plen
	case happy.LevelError:
		return aurora.White(pfx).BgRed().Bold().String(), plen
	case happy.LevelCritical, happy.LevelAlert, happy.LevelEmergency:
		return aurora.Black(pfx).BgBrightRed().Bold().String(), plen
	case happy.LevelOut:
		return aurora.White(pfx).Bold().String(), plen
	default:
		return pfx, plen
	}
}
