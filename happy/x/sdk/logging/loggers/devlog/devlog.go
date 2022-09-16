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

// Package devlog is for DEVELOPMENT only. This logger sacrifices performace
// for development friendliness.
//
// very similar to go log package.
//
// Oprions:
// level              default(happy.LOG_NOTICE)
// colors             default(false)
//
//	stdlog supports output colors when platform supports it.
//
// prefix             default("")
//
//	prefix on each line to identify the logger.
//
// filenames.level    default(happy.LOG_NOTICE)
//
//	log level since when to print filenames (-1) value = never
//
// filenames.long     default(false)
//
//	set option true to show long filenames.
package devlog

import (
	"errors"
	"fmt"
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/happyx"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var std = New(os.Stderr)

type color uint

type Logger struct {
	dlvl      happy.LogPriority
	lvl       happy.LogPriority
	mu        sync.Mutex // ensures atomic writes; protects the following fields
	out       io.Writer  // destination for output
	buf       []byte     // for accumulating text to write
	isDiscard int32      // atomic boolean: whether out == io.Discard

	scope          string
	filenamesLvl   happy.LogPriority
	filenamesLong  bool
	filenamesPre   string
	colors         bool
	tsdate         bool
	tstime         bool
	tsmicroseconds bool
	tsutc          bool

	last time.Time
}

func Default() happy.Logger { return std }

func New(out io.Writer, options ...happy.OptionWriteFunc) happy.Logger {
	l := &Logger{
		out:            out,
		dlvl:           happy.LOG_NOTICE, // default level
		lvl:            happy.LOG_SYSTEMDEBUG,
		scope:          "",
		filenamesLvl:   happy.LOG_NOTICE,
		filenamesLong:  true,
		filenamesPre:   "",
		colors:         true,
		tsdate:         false,
		tstime:         true,
		tsmicroseconds: false,
		tsutc:          true,
		last:           time.Now(),
	}
	if out == io.Discard {
		l.isDiscard = 1
	}
	if len(options) > 0 {
		for _, opt := range options {
			v, err := happyx.OptionParseFuncFor(opt)()
			if err != nil {
				l.Emergencyf("error while reading provided devlog options")
				continue
			}

			if err := l.SetOptionDefault(v); err != nil {
				l.Emergency(err)
			}
		}
	}
	if len(l.filenamesPre) == 0 {
		wd, err := os.Getwd()
		if err != nil {
			l.Error(err)
		}
		l.filenamesPre = wd
	}
	return l
}

// API
func (l *Logger) GetPriority() happy.LogPriority {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.lvl
}

func (l *Logger) SetPriority(lvl happy.LogPriority) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.dlvl = lvl
	l.lvl = lvl
}

func (l *Logger) SetRuntimePriority(lvl happy.LogPriority) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lvl = lvl
}

func (l *Logger) ResetPriority() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lvl = l.dlvl
}

func (l *Logger) Writer() io.Writer {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.out
}

func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.out = w
	isDiscard := int32(0)
	if w == io.Discard {
		isDiscard = 1
	}
	atomic.StoreInt32(&l.isDiscard, isDiscard)
}

func (l *Logger) Print(v ...any) {
	if atomic.LoadInt32(&l.isDiscard) != 0 {
		return
	}
	l.Output(2, fmt.Sprint(v...))
}

func (l *Logger) Printf(format string, v ...any) {
	if atomic.LoadInt32(&l.isDiscard) != 0 {
		return
	}
	l.Output(2, fmt.Sprintf(format, v...))
}

func (l *Logger) Println(v ...any) {
	if atomic.LoadInt32(&l.isDiscard) != 0 {
		return
	}
	l.Output(2, fmt.Sprintln(v...))
}

func (l *Logger) Output(calldepth int, s string) error {
	return l.output(-1, "", calldepth, s, 0, 0)
}

// LOG_EMERG
func (l *Logger) Emergency(args ...any) {
	l.handleHappyErrors(args)
	l.output(happy.LOG_EMERG, "emerg   ", 2, fmt.Sprint(args...), whiteFg, redBg)
}

func (l *Logger) Emergencyf(template string, args ...any) {
	l.handleHappyErrors(args)
	l.output(happy.LOG_EMERG, "emerg   ", 2, fmt.Sprintf(template, args...), whiteFg, redBg)
}

// LOG_ALERT
func (l *Logger) Alert(args ...any) {
	l.handleHappyErrors(args)
	l.output(happy.LOG_ALERT, "alert   ", 2, fmt.Sprint(args...), whiteFg, redBg)
}
func (l *Logger) Alertf(template string, args ...any) {
	l.handleHappyErrors(args)
	l.output(happy.LOG_ALERT, "alert   ", 2, fmt.Sprintf(template, args...), whiteFg, redBg)
}

// LOG_CRIT
func (l *Logger) Critical(args ...any) {
	l.handleHappyErrors(args)
	l.output(happy.LOG_CRIT, "crit    ", 2, fmt.Sprint(args...), whiteFg, redBg)
}

func (l *Logger) Criticalf(template string, args ...any) {
	l.handleHappyErrors(args)
	l.output(happy.LOG_CRIT, "crit    ", 2, fmt.Sprintf(template, args...), whiteFg, redBg)
}

func (l *Logger) BUG(args ...any) {
	l.handleHappyErrors(args)
	l.output(happy.LOG_CRIT, "BUG     ", 2, fmt.Sprint(args...), whiteFg, redBg)
}
func (l *Logger) BUGF(template string, args ...any) {
	l.handleHappyErrors(args)
	l.output(happy.LOG_CRIT, "BUG     ", 2, fmt.Sprintf(template, args...), whiteFg, redBg)
}

// LOG_ERR
func (l *Logger) Error(args ...any) {
	l.handleHappyErrors(args)
	l.output(happy.LOG_ERR, "error   ", 2, fmt.Sprint(args...), redFg, blackBg)
}

func (l *Logger) Errorf(template string, args ...any) {
	l.handleHappyErrors(args)
	l.output(happy.LOG_ERR, "error ", 2, fmt.Sprintf(template, args...), redFg, blackBg)
}

// LOG_WARNING
func (l *Logger) Warn(args ...any) {
	l.handleHappyErrors(args)
	l.output(happy.LOG_WARNING, "warn    ", 2, fmt.Sprint(args...), yellowFg, blackBg)
}
func (l *Logger) Warnf(template string, args ...any) {
	l.handleHappyErrors(args)
	l.output(happy.LOG_WARNING, "warn    ", 2, fmt.Sprintf(template, args...), yellowFg, blackBg)
}
func (l *Logger) Deprecated(args ...any) {
	l.handleHappyErrors(args)
	l.output(happy.LOG_WARNING, "depr    ", 2, fmt.Sprint(args...), whiteFg, yellowBg)
}
func (l *Logger) Deprecatedf(template string, args ...any) {
	l.handleHappyErrors(args)
	l.output(happy.LOG_WARNING, "depr    ", 2, fmt.Sprintf(template, args...), whiteFg, yellowBg)
}

// LOG_NOTICE
func (l *Logger) Notice(args ...any) {
	l.output(happy.LOG_NOTICE, "notice  ", 2, fmt.Sprint(args...), blackFg, cyanBg)
}

func (l *Logger) Noticef(template string, args ...any) {
	l.output(happy.LOG_NOTICE, "notice  ", 2, fmt.Sprintf(template, args...), blackFg, cyanBg)
}

func (l *Logger) Ok(args ...any) {
	l.output(happy.LOG_NOTICE, "ok      ", 2, fmt.Sprint(args...), greenFg, blackBg)
}
func (l *Logger) Okf(template string, args ...any) {
	l.output(happy.LOG_NOTICE, "ok ", 2, fmt.Sprintf(template, args...), greenFg, blackBg)
}

// LOG_INFO
func (l *Logger) Info(args ...any) {
	l.output(happy.LOG_INFO, "info    ", 2, fmt.Sprint(args...), cyanFg, blackBg)
}
func (l *Logger) Infof(template string, args ...any) {
	l.output(happy.LOG_INFO, "info ", 2, fmt.Sprintf(template, args...), 0, 0)
}
func (l *Logger) Experimental(args ...any) {
	l.output(happy.LOG_INFO, "exp     ", 2, fmt.Sprint(args...), yellowFg, whiteBg)
}
func (l *Logger) Experimentalf(template string, args ...any) {
	l.output(happy.LOG_INFO, "exp     ", 2, fmt.Sprintf(template, args...), yellowFg, whiteBg)
}
func (l *Logger) NotImplemented(args ...any) {
	l.output(happy.LOG_INFO, "notimpl ", 2, fmt.Sprint(args...), blackFg, yellowBg)
}
func (l *Logger) NotImplementedf(template string, args ...any) {
	l.output(happy.LOG_INFO, "notimpl ", 2, fmt.Sprintf(template, args...), blackFg, yellowBg)
}

// LOG_DEBUG
func (l *Logger) Debug(args ...any) {
	l.output(happy.LOG_DEBUG, "debug   ", 2, fmt.Sprint(args...), whiteFg, 0)
}
func (l *Logger) Debugf(template string, args ...any) {
	l.output(happy.LOG_DEBUG, "debug   ", 2, fmt.Sprintf(template, args...), whiteFg, 0)
}

// LOG_SYSTEMDEBUG
func (l *Logger) SystemDebug(args ...any) {
	l.output(happy.LOG_SYSTEMDEBUG, "system  ", 2, fmt.Sprint(args...), blueFg, 0)
}

func (l *Logger) SystemDebugf(template string, args ...any) {
	l.output(happy.LOG_SYSTEMDEBUG, "system  ", 2, fmt.Sprintf(template, args...), blueFg, 0)
}

func (l *Logger) SetOptionDefault(o happy.Variable) happy.Error {
	switch o.Key() {
	case "colors":
		l.colors = o.Bool()
	case "scope":
		l.scope = o.String()
	case "level":
		l.SetPriority(happy.LogPriority(o.Int()))
	case "filenames.level":
		l.filenamesLvl = happy.LogPriority(o.Int())
	case "filenames.long":
		l.filenamesLong = o.Bool()
	case "filenames.pre":
		l.filenamesPre = o.String()
	case "ts.date":
		l.tsdate = o.Bool()
	case "ts.time":
		l.tstime = o.Bool()
	case "ts.microseconds":
		l.tsmicroseconds = o.Bool()
	case "ts.utc":
		l.tsutc = o.Bool()
	default:
		return happyx.InvalidOptionError("devlog", o.Key())
	}
	return nil
}

func (l *Logger) SetOptionDefaultKeyValue(key string, val any) happy.Error {
	return happyx.Errorf("%w: devlog.SetOptionDefaultKeyValue", happyx.ErrNotImplemented)
}

func (l *Logger) SetOptionsDefaultFuncs(vfuncs ...happy.VariableParseFunc) happy.Error {
	return happyx.Errorf("%w: devlog.SetOptionsDefaultFuncs", happyx.ErrNotImplemented)
}

func (l *Logger) handleHappyErrors(args []any) {
	for _, a := range args {
		if e, ok := a.(error); ok {
			if errors.Is(e, happyx.ErrNotImplemented) {
				l.output(happy.LOG_INFO, "notimpl ", 3, fmt.Sprint(args...), redFg, blackBg)
			}
		}
	}
}

func (l *Logger) output(lvl happy.LogPriority, ltype string, calldepth int, s string, fg, bg color) error {
	now := time.Now() // get this early.
	var file string
	var line int
	l.mu.Lock()
	defer l.mu.Unlock()

	// log priority not satisfied
	if lvl >= 0 && lvl > l.lvl {
		return nil
	}

	durr := now.Sub(l.last)
	l.last = now

	if l.colors {
		// Set color based on time from last log entry
		var durcolor color

		if durr < time.Microsecond*100 {
			durcolor = greenFg // >= 100000 ticks / ps
		} else if durr < time.Microsecond { // >= 1000 ticks / ps
			durcolor = yellowFg
		} else {
			durcolor = cyanFg // < 1000 ticks / ps
		}
		s = s + colorize("+"+durr.String(), durcolor, 0, 2)
	} else {
		s = s + " +" + durr.String()
	}

	if l.filenamesLvl != -1 && lvl <= l.filenamesLvl {
		// Release lock while getting caller info - it's expensive.
		l.mu.Unlock()
		var ok bool
		_, file, line, ok = runtime.Caller(calldepth)
		if !ok {
			file = "???"
			line = 0
		}
		l.mu.Lock()
	}
	if len(l.scope) > 0 {
		l.buf = append([]byte(l.scope[0:]), ' ')
	} else {
		l.buf = l.buf[:0]
	}
	if l.colors && (fg+bg) > 0 {
		ltype = colorize(ltype, fg, bg, 1)
	}
	l.formatHeader(ltype, lvl, &l.buf, now, file, line)
	l.buf = append(l.buf, s...)

	if len(s) == 0 || s[len(s)-1] != '\n' {
		l.buf = append(l.buf, '\n')
	}
	_, err := l.out.Write(l.buf)
	return err
}

func (l *Logger) formatHeader(ltype string, lvl happy.LogPriority, buf *[]byte, t time.Time, file string, line int) {
	if len(ltype) > 0 {
		*buf = append(*buf, ltype...)
	}

	if l.tsdate || l.tstime || l.tsmicroseconds {
		if l.tsutc {
			t = t.UTC()
		}
		if l.colors {
			*buf = append(*buf, '\033', '[', '2', 'm')
		}
		if l.tsdate {
			year, month, day := t.Date()
			itoa(buf, year, 4)
			*buf = append(*buf, '-')
			itoa(buf, int(month), 2)
			*buf = append(*buf, '-')
			itoa(buf, day, 2)
			*buf = append(*buf, ' ')
		}
		if l.tstime || l.tsmicroseconds {
			hour, min, sec := t.Clock()
			itoa(buf, hour, 2)
			*buf = append(*buf, ':')
			itoa(buf, min, 2)
			*buf = append(*buf, ':')
			itoa(buf, sec, 2)
			if l.tsmicroseconds {
				*buf = append(*buf, '.')
				itoa(buf, t.Nanosecond()/1e3, 6)
			}
			*buf = append(*buf, ' ')
		}
		if l.colors {
			*buf = append(*buf, '\033', '[', '0', 'm')
		}
	}

	if l.filenamesLvl != -1 && lvl <= l.filenamesLvl {
		if l.colors {
			*buf = append(*buf, '\033', '[', '3', '3', 'm')
		}
		if !l.filenamesLong {
			short := file
			for i := len(file) - 1; i > 0; i-- {
				if file[i] == '/' {
					short = file[i+1:]
					break
				}
			}
			file = short
		} else if len(l.filenamesPre) > 0 {
			file = "." + strings.TrimPrefix(file, l.filenamesPre)
		}
		*buf = append(*buf, file...)
		*buf = append(*buf, ':')
		itoa(buf, line, -1)
		*buf = append(*buf, ": "...)
		if l.colors {
			*buf = append(*buf, '\033', '[', '0', 'm')
		}
	}
}

// from go's std log
// Cheap integer to fixed-width decimal ASCII.
// Give a negative width to avoid zero-padding.
func itoa(buf *[]byte, i int, wid int) {
	// Assemble decimal in reverse order.
	var b [20]byte
	bp := len(b) - 1
	for i >= 10 || wid > 1 {
		wid--
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	// i < 10
	b[bp] = byte('0' + i)
	*buf = append(*buf, b[bp:]...)
}
