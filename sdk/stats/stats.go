// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package stats

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/strings/humanize"
	"github.com/happy-sdk/happy/pkg/strings/textfmt"
	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/sdk/app/session"
	"github.com/happy-sdk/happy/sdk/custom"
	"github.com/happy-sdk/happy/sdk/internal/fsutils"
	"github.com/happy-sdk/happy/sdk/services"
	"github.com/happy-sdk/happy/sdk/services/service"
)

type Settings struct {
	Enabled settings.Bool `key:"enabled,save" default:"false" mutation:"once"  desc:"Enable runtime statistics"`
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	b, err := settings.New(s)
	if err != nil {
		return nil, err
	}

	return b, nil
}

type Profiler struct {
	custom.API
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

func New(title string) *Profiler {
	return &Profiler{
		title: title,
		db:    new(vars.Map),
		tsloc: time.UTC,
	}
}

func (r *Profiler) Get(key string) vars.Variable {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v := r.db.Get(key)
	return v
}

func (r *Profiler) Set(key string, value any) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.db == nil {
		r.db = new(vars.Map)
	}
	return r.db.Store(key, value)
}

func (r *Profiler) State() State {
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

func (r *Profiler) Update() {
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
	_ = r.db.Store("goroutines.current", numGoroutines)
	_ = r.db.Store("goroutines.min", r.goroutines.min)
	_ = r.db.Store("goroutines.max", r.goroutines.max)

	// Key memory statistics
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	// _ = r.db.Store("mem.allocated", humanize.IBytes(mem.Alloc))
	_ = r.db.Store("mem.total_allocated", humanize.IBytes(mem.TotalAlloc))
	_ = r.db.Store("mem.sys", humanize.IBytes(mem.Sys))
	_ = r.db.Store("mem.heap.alloc", humanize.IBytes(mem.HeapAlloc))
	_ = r.db.Store("mem.heap.sys", humanize.IBytes(mem.HeapSys))
	_ = r.db.Store("mem.heap.heap_objects", mem.HeapObjects)
	_ = r.db.Store("mem.pointer.lookups", mem.Lookups)
	_ = r.db.Store("mem.mallocs", mem.Mallocs)

	// Critical GC metrics
	_ = r.db.Store("mem.gc.next", humanize.IBytes(mem.NextGC))
	_ = r.db.Store("mem.gc.num", mem.NumGC)
	_ = r.db.Store("mem.gc.cpu_fraction", mem.GCCPUFraction)
	_ = r.db.Store("mem.stack.inuse", humanize.IBytes(mem.StackInuse))
	r.lastUpdated = time.Now().In(r.tsloc)
}

func (r *Profiler) SetTimeLocation(loc *time.Location) {
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

// stateJSON is an internal type for marshaling/unmarshaling.
type stateJSON struct {
	Title string            `json:"title"`
	Time  string            `json:"time"`
	Vars  map[string]string `json:"vars"`
}

// MarshalJSON customizes JSON serialization for State.
func (s State) Vars() map[string]vars.Variable {
	return s.vars
}
func (s State) MarshalJSON() ([]byte, error) {
	vars := make(map[string]string)
	for key, v := range s.vars {
		vars[key] = v.String()
	}
	return json.Marshal(stateJSON{
		Title: s.title,
		Time:  s.time.Format(time.RFC3339),
		Vars:  vars,
	})
}

// UnmarshalJSON customizes JSON deserialization for State.
func (s *State) UnmarshalJSON(data []byte) error {
	var temp stateJSON
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Convert fields
	s.title = temp.Title
	t, err := time.Parse(time.RFC3339, temp.Time)
	if err != nil {
		return err
	}
	s.time = t
	s.vars = make(map[string]vars.Variable)
	for key, v := range temp.Vars {
		s.vars[key], err = vars.New(key, v, true)
		if err != nil {
			return err
		}
	}
	return nil
}

func AsService(prof *Profiler) *services.Service {
	svc := services.New(service.Config{
		Name: "Stats",
		Slug: "app-runtime-stats",
	})

	svc.Tick(func(sess *session.Context, ts time.Time, delta time.Duration) error {
		startedAt := prof.Get("app.started.at").String()
		if startedAt != "" {
			started, err := time.Parse(time.RFC3339, startedAt)
			if err != nil {
				return err
			}
			uptime := time.Since(started)
			if err := prof.Set("app.uptime", uptime.String()); err != nil {
				return err
			}
		}
		prof.Update()
		return nil
	})

	svc.Cron(func(schedule services.CronScheduler) {
		schedule.Job("stats:collect-storage-info", "@every 15s", func(sess *session.Context) error {
			cachePath := sess.Get("app.fs.path.cache").String()
			tmpPath := sess.Get("app.fs.path.tmp").String()

			if cacheSize, err := fsutils.DirSize(cachePath); err != nil {
				sess.Log().Error("failed to get cache size", slog.String("err", err.Error()))
			} else {
				_ = prof.Set("fs.cache.size", humanize.Bytes(uint64(cacheSize)))
			}

			if tmpSize, err := fsutils.DirSize(tmpPath); err != nil {
				sess.Log().Error("failed to get tmp size", slog.String("err", err.Error()))
			} else {
				_ = prof.Set("fs.tmp.size", humanize.Bytes(uint64(tmpSize)))
			}

			if availableSpace, err := fsutils.AvailableSpace(cachePath); err != nil {
				sess.Log().Error("failed to get available space", slog.String("err", err.Error()))
			} else {
				_ = prof.Set("fs.available", humanize.Bytes(uint64(availableSpace)))
			}

			return nil
		})
	})
	return svc
}
