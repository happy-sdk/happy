// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package stats

import (
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/strings/humanize"
	"github.com/happy-sdk/happy/pkg/strings/textfmt"
	"github.com/happy-sdk/happy/pkg/vars"
)

type Settings struct {
	Enabled settings.Bool `key:"enabled,save" default:"false" mutation:"once"`
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	b, err := settings.New(s)
	if err != nil {
		return nil, err
	}

	return b, nil
}

type Runtime struct {
	title       string
	mu          sync.RWMutex
	db          *vars.Map
	lastUpdated time.Time
	tsloc       *time.Location

	goroutines struct {
		current int
		min     int
		max     int
	}
}

func New(title string) *Runtime {
	return &Runtime{
		title: title,
		db:    new(vars.Map),
		tsloc: time.UTC,
	}
}

func (r *Runtime) Get(key string) vars.Variable {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v := r.db.Get(key)
	return v
}

func (r *Runtime) Set(key string, value any) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.db == nil {
		r.db = new(vars.Map)
	}
	return r.db.Store(key, value)
}

func (r *Runtime) State() State {
	r.Update()

	r.mu.RLock()
	defer r.mu.RUnlock()

	state := State{
		vars:  make(map[string]vars.Variable),
		title: r.title,
		time:  r.lastUpdated,
	}
	r.db.Range(func(v vars.Variable) bool {
		state.vars[v.Name()] = v
		return true
	})
	return state
}

func (r *Runtime) Update() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Goroutine statistics
	numGoroutines := runtime.NumGoroutine()
	if r.goroutines.min == 0 || r.goroutines.min > numGoroutines {
		r.goroutines.min = numGoroutines
	}
	if r.goroutines.max < numGoroutines {
		r.goroutines.max = numGoroutines
	}
	_ = r.db.Store("app.goroutines.current", numGoroutines)
	_ = r.db.Store("app.goroutines.min", r.goroutines.min)
	_ = r.db.Store("app.goroutines.max", r.goroutines.max)

	// Key memory statistics
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	_ = r.db.Store("mem.allocated", humanize.IBytes(mem.Alloc))
	_ = r.db.Store("mem.total_allocated", humanize.IBytes(mem.TotalAlloc))
	_ = r.db.Store("mem.sys", humanize.IBytes(mem.Sys))
	_ = r.db.Store("mem.heap.alloc", humanize.IBytes(mem.HeapAlloc))
	_ = r.db.Store("mem.heap.sys", humanize.IBytes(mem.HeapSys))

	// Critical GC metrics
	_ = r.db.Store("mem.gc.next", humanize.IBytes(mem.NextGC))
	_ = r.db.Store("mem.gc.num", mem.NumGC)
	_ = r.db.Store("mem.gc.cpu_fraction", mem.GCCPUFraction)
	r.lastUpdated = time.Now().In(r.tsloc)
}

func (r *Runtime) SetTimezone(loc *time.Location) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tsloc = loc
}

type State struct {
	title string
	time  time.Time
	vars  map[string]vars.Variable
}

func (s State) Time() time.Time {
	return s.time
}

func (s State) Get(key string) vars.Variable {
	if s.vars == nil {
		return vars.EmptyVariable
	}
	if v, ok := s.vars[key]; ok {
		return v
	}
	return vars.EmptyVariable
}

func (s State) Range(cb func(v vars.Variable)) {
	if s.vars == nil {
		return
	}
	for _, v := range s.vars {
		cb(v)
	}
}

func (s State) String() string {
	tbl := textfmt.Table{
		Title:      fmt.Sprintf(s.title+" @ %s", s.time.Format(time.RFC3339)),
		WithHeader: true,
	}
	tbl.AddRow("METRIC", "VALUE")

	keys := make([]string, len(s.vars))
	i := 0
	for key := range s.vars {
		keys[i] = key
		i++
	}
	sort.Strings(keys)

	for _, v := range keys {
		tbl.AddRow(s.vars[v].Name(), s.vars[v].String())
	}
	return tbl.String()
}
