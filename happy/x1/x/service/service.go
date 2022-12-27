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
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/happyx"
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

func (s *Service) OnInitialize(action happy.ActionWithStatusFunc) {
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

func (s *Service) OnEvent(scope, key string, cb happy.ActionWithEventFunc) {
	if s.svc.listeners == nil {
		s.svc.listeners = make(map[string][]happy.ActionWithEventFunc)
	}

	lid := scope + "." + key
	s.svc.listeners[lid] = append(s.svc.listeners[lid], cb)
}

func (s *Service) OnAnyEvent(cb happy.ActionWithEventFunc) {
	if s.svc.listeners == nil {
		s.svc.listeners = make(map[string][]happy.ActionWithEventFunc)
	}
	s.svc.listeners["any"] = append(s.svc.listeners["any"], cb)
}

func (s *Service) Cron(setup happy.ActionCronSchedulerSetup) {
	s.svc.cronsetup = setup
}

func (s *Service) Register(sess happy.Session) (happy.BackgroundService, happy.Error) {
	if !sess.Opts().Has("app.peer.addr") {
		return nil, happyx.Errorf("can not initialize service %s - app.peer.addr not set", s.slug)
	}
	u, err := happyx.ParseURL(sess.Get("app.peer.addr").String() + s.path)
	if err != nil {
		sess.Log().Errorf("failed to parse service url: %s", err)
		return s.svc, ErrService.Wrap(err)
	}

	s.url = u
	return s.svc, nil
}

type BackgroundService struct {
	// initOnce    sync.Once
	initialize happy.ActionWithStatusFunc
	start      happy.ActionWithArgsFunc
	stop       happy.ActionFunc
	tick       happy.ActionTickFunc
	tock       happy.ActionTickFunc
	cronsetup  happy.ActionCronSchedulerSetup
	cron       *Cron
	listeners  map[string][]happy.ActionWithEventFunc

	initialized bool
}

func (s *BackgroundService) Initialize(sess happy.Session, status happy.ApplicationStatus) happy.Error {

	if s.initialized || s.initialize == nil {
		return nil
	}

	if err := s.initialize(sess, status); err != nil {
		return ErrService.Wrap(err)
	}
	s.initialized = true
	if s.cronsetup != nil {
		s.cron = newCron(sess)
		s.cronsetup(s.cron)
	}

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

func (s *BackgroundService) HandleEvent(sess happy.Session, ev happy.Event) error {
	var err error
	lid := ev.Scope() + "." + ev.Key()

	for sk, listeners := range s.listeners {
		for _, listener := range listeners {
			if sk == "any" || sk == lid {
				if e := listener(sess, ev); e != nil {
					err = e
					sess.Log().Error(e)
				}
			}
		}
	}
	return err
}
