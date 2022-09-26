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

package service

import (
	"context"
	"fmt"
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/happyx"
	"github.com/mkungla/happy/x/pkg/vars"
	"github.com/robfig/cron/v3"
	"net/url"
	"strings"
	"sync"
	"time"
)

var ErrService = happyx.NewError("service error")

type Service struct {
	slug happy.Slug
	name string
	path string
	opts happy.Variables
	url  happy.URL

	svc *BackgroundService
}

// path to register service to
func New(slug, name, path string, defaultOptions ...happy.OptionSetFunc) (happy.Service, happy.Error) {
	s, err := happyx.NewSlug(slug)
	if err != nil {
		return nil, ErrService.Wrap(err)
	}

	opts, err := happyx.OptionsToVariables(defaultOptions...)
	if err != nil {
		return nil, ErrService.Wrap(err)
	}

	svc := &Service{
		slug: s,
		path: path,
		name: name,
		opts: opts,
		svc:  &BackgroundService{},
	}
	return svc, nil
}

func (s *Service) Slug() happy.Slug { return s.slug }
func (s *Service) Name() string     { return s.name }

func (s *Service) URL() happy.URL { return s.url }

func (s *Service) OnInitialize(action happy.ActionFunc) {
	s.svc.initialize = action
}

func (s *Service) OnStart(action happy.ActionWithArgsFunc) {
	s.svc.start = action
}

func (s *Service) OnStop(action happy.ActionFunc) {
	s.svc.stop = action
}

func (s *Service) OnRequest(happy.ServiceRouter) {}

func (s *Service) OnTick(action happy.ActionTickFunc) {
	s.svc.tick = action
}

func (s *Service) OnTock(action happy.ActionTickFunc) {
	s.svc.tock = action
}
func (s *Service) OnEvent(key string, cb happy.ActionWithEventFunc) {}
func (s *Service) OnAnyEvent(happy.ActionWithEventFunc)             {}

func (s *Service) Cron(setup happy.ActionCronSchedulerSetup) {
	s.svc.cronsetup = setup
}

func (s *Service) Register(sess happy.Session) (happy.BackgroundService, happy.Error) {
	if !sess.Opts().Has("app.peer.addr") {
		return nil, happyx.Errorf("can not initialize service %s - app.peer.addr not set", s.slug)
	}
	url, err := url.Parse(sess.Get("app.peer.addr").String() + s.path)
	if err != nil {
		return nil, ErrService.Wrap(err)
	}
	s.url = url
	return s.svc, nil
}

type BackgroundService struct {
	// initOnce    sync.Once
	initialize  happy.ActionFunc
	start       happy.ActionWithArgsFunc
	stop        happy.ActionFunc
	tick        happy.ActionTickFunc
	tock        happy.ActionTickFunc
	cronsetup   happy.ActionCronSchedulerSetup
	cron        *Cron
	initialized bool
}

func (s *BackgroundService) Initialize(sess happy.Session) happy.Error {

	if s.initialized || s.initialize == nil {
		return nil
	}

	if err := s.initialize(sess); err != nil {
		return ErrService.Wrap(err)
	}
	s.initialized = true
	if s.cronsetup != nil {
		s.cron = newCron(sess)
		s.cronsetup(s.cron)
	}

	return nil
}

type Cron struct {
	sess   happy.Session
	lib    *cron.Cron
	jobIDs []cron.EntryID
}

func newCron(sess happy.Session) *Cron {
	c := &Cron{}
	c.sess = sess
	c.lib = cron.New(cron.WithParser(cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)))
	return c
}

func (cs *Cron) Job(expr string, cb happy.ActionCronFunc) {
	id, err := cs.lib.AddFunc(expr, func() {
		if err := cb(cs.sess); err != nil {
			cs.sess.Log().Error(err)
		}
	})
	cs.jobIDs = append(cs.jobIDs, id)
	if err != nil {
		cs.sess.Log().Errorf("cron(%d): %s", id, err)
		return
	}
}

func (cs *Cron) Start() happy.Error {
	for _, id := range cs.jobIDs {
		job := cs.lib.Entry(id)
		if job.Job != nil {
			go job.Job.Run()
		}
	}
	cs.lib.Start()
	return nil
}

func (cs *Cron) Stop() happy.Error {
	ctx := cs.lib.Stop()
	<-ctx.Done()
	return nil
}

func (s *BackgroundService) Start(sess happy.Session, args happy.Variables) happy.Error {
	if s.start == nil {
		return nil
	}

	if err := s.start(sess, args); err != nil {
		return ErrService.Wrap(err)
	}

	if s.cron != nil {
		sess.Log().SystemDebug("starting cron scheduler")
		s.cron.Start()
	}
	return nil
}

func (s *BackgroundService) Stop(sess happy.Session) happy.Error {
	if s.stop == nil {
		return nil
	}

	if err := s.stop(sess); err != nil {
		return ErrService.Wrap(err)
	}
	if s.cron != nil {
		sess.Log().SystemDebug("stopping cron scheduler, waiting jobs to finish")
		s.cron.Stop()
	}
	return nil
}

func (s *BackgroundService) Tick(sess happy.Session, ts time.Time, delta time.Duration) happy.Error {
	if s.tick == nil {
		return nil
	}

	if err := s.tick(sess, ts, delta); err != nil {
		return ErrService.Wrap(err)
	}
	return nil
}

func (s *BackgroundService) Tock(sess happy.Session, ts time.Time, delta time.Duration) happy.Error {
	if s.tock == nil {
		return nil
	}

	if err := s.tock(sess, ts, delta); err != nil {
		return ErrService.Wrap(err)
	}
	return nil
}

func NewServiceLoader(sess happy.Session, status happy.ApplicationStatus, svcs ...string) *ServiceLoader {
	var urls []happy.URL

	loader := &ServiceLoader{
		loaded: make(chan struct{}),
	}

	peeraddr := sess.Get("app.peer.addr").String()
	for _, svc := range svcs {
		if strings.HasPrefix(svc, "/") {
			svc = peeraddr + svc
		}
		u, err := url.Parse(svc)
		if err != nil {
			loader.err = ErrServiceLoader.Wrap(err)
			loader.done = true
			break
		}
		urls = append(urls, u)
	}

	loader.request(sess, status, urls...)

	return loader
}

type ServiceLoader struct {
	mu     sync.Mutex
	done   bool
	loaded chan struct{}
	err    happy.Error
}

var ErrServiceLoader = happyx.NewError("service loader error")

func (sl *ServiceLoader) Err() happy.Error {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	if !sl.done {
		return happyx.BUG.WithTextf("Service loader Error checked before loader finished! Did you wait for .Loaded? %v", sl.err)
	}
	return sl.err
}

func (sl *ServiceLoader) Loaded() <-chan struct{} {
	return sl.loaded
}

func (sl *ServiceLoader) request(sess happy.Session, status happy.ApplicationStatus, urls ...happy.URL) {
	go func() {
		defer close(sl.loaded)

		var needloading []happy.URL
		for _, url := range urls {
			stat, err := status.GetServiceStatus(url)
			// key := "service.[" + url.String() + "].registered"
			if err != nil {
				sl.mu.Lock()
				sl.err = err
				sl.done = true
				sl.mu.Unlock()
				return
			}
			// check := "service.[" + url.String() + "].running"
			// check if service is already running
			if !stat.Running {
				needloading = append(needloading, url)
			}
		}

		if len(needloading) == 0 {
			sl.mu.Lock()
			sl.done = true
			sl.mu.Unlock()
			return
		}

		sess.Dispatch(NewRequireServicesEvent(urls...))

		timeout := time.Duration(sess.Settings().Get("engine.service.discovery.timeout").Int64())
		if timeout <= 0 {
			timeout = time.Second * 30
		}

		ctx, cancel := context.WithTimeout(sess, timeout)
		defer cancel()
	queue:
		for {
			select {
			case <-ctx.Done():
				sl.mu.Lock()
				sl.err = ErrService.WithTextf("service loader timeout %s", timeout)
				sl.done = true
				sl.mu.Unlock()
				break queue
			default:
				loaded := 0
				for _, url := range needloading {
					stat, err := status.GetServiceStatus(url)
					if err != nil {
						sl.mu.Lock()
						sl.err = err
						sl.mu.Unlock()
						continue
					}
					if stat.Running {
						loaded++
					}
				}
				if loaded == len(needloading) {
					sl.mu.Lock()
					sl.done = true
					sl.mu.Unlock()
					break queue
				}
			}
		}
	}()
}

type Event struct {
	key     string
	ts      time.Time
	payload happy.Variables
}

func (ev Event) Key() string {
	return ev.key
}

func (ev Event) Scope() string {
	return "session"
}

func (ev Event) Err() happy.Error {
	return nil
}

func (ev Event) Payload() happy.Variables {
	return ev.payload
}

func (ev Event) Time() time.Time {
	return ev.ts
}

func NewRequireServicesEvent(urls ...happy.URL) happy.Event {
	svcs := vars.AsMap[happy.Variables, happy.Variable, happy.Value](new(vars.Map))
	for i, url := range urls {
		svcs.Store(fmt.Sprintf("service.%d", i), url)
	}
	return Event{
		key:     "require.services",
		payload: svcs,
		ts:      time.Now(),
	}
}
