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

package console

import (
	"fmt"
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/happyx"
	"net/http"
	"os"
	"time"
)

type TransportLogger struct {
	log *Logger
}

func NewTransportLogger(lvl happy.LogPriority) *TransportLogger {
	logger := New(
		os.Stdout,
		happyx.Option("filenames.level", -1),
	)
	logger.SetPriority(lvl)
	return &TransportLogger{
		log: logger,
	}
}

func (l *TransportLogger) Error(args ...any) {
	l.log.Error(args...)
}

func (l *TransportLogger) HTTP(code int, ts time.Time, w http.ResponseWriter, r *http.Request) {

	// fmt.Println("REQUEST:")
	// fmt.Println("method:", r.Method)
	// fmt.Println("path:", r.URL.Path)
	// fmt.Println("code:", code)
	// fmt.Println("path:")
	// for k, h := range r.Header {
	// 	fmt.Printf("header: %s - %s %s = %s\n", r.URL.Path, r.Method, k, h)
	// }
	// fmt.Println("RESPONSE:")
	// for k, h := range w.Header() {
	// 	fmt.Printf("header: %s - %s %s = %s\n", r.URL.Path, r.Method, k, h)
	// }

	var (
		fg, bg color
		msg    string
		label  string
		lvl    happy.LogPriority
	)

	msg = r.URL.Path

	label = fmt.Sprintf("%-4s %d ", r.Method, code)
	if code >= 200 && code < 300 {
		fg = greenFg
		bg = blackBg
		lvl = happy.LOG_INFO
	} else if code >= 300 && code < 400 {
		fg = yellowFg
		bg = blackBg
		lvl = happy.LOG_INFO
	} else if code >= 400 && code < 500 {
		fg = redFg
		bg = blackBg
		lvl = happy.LOG_NOTICE
	} else if code >= 500 {
		fg = whiteFg
		bg = redBg
		lvl = happy.LOG_ERR
	} else {
		fg = whiteFg
		bg = cyanBg
		lvl = happy.LOG_INFO
	}
	l.log.outputWithDuration(lvl, label, 4, msg, ts, fg, bg)
}
