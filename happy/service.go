// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package happy

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mkungla/happy/pkg/address"
	"github.com/mkungla/happy/pkg/vars"
	"github.com/robfig/cron/v3"
	"golang.org/x/exp/slog"
)

type Service struct {
	slug string

	EventListener
	TickerFuncs

	initializeAction Action
	startAction      Action
	stopAction       Action
	tickAction       ActionTick
	tockAction       ActionTick
	listeners        map[string][]ActionWithEvent

	cronsetup func(schedule CronScheduler)
}

func NewService(slug string, opts ...OptionAttr) *Service {
	svc := &Service{
		slug: slug,
	}
	return svc
}

// OnInitialize is called when app is preparing runtime
// and attaching services.
func (s *Service) OnInitialize(action Action) {
	s.initializeAction = action
}

// OnStart is called when service is requested to be started.
// For instace when command is requiring this service or whenever
// service is required on runtime via sess.RequireService call.
//
// Start can be called multiple times in case of service restarts.
// If you do not want to allow service restarts you should implement
// your logic in OnStop when it's called first time and check that
// state OnStart.
func (s *Service) OnStart(action Action) {
	s.startAction = action
}

func (s *Service) OnStop(action Action) {
	s.stopAction = action
}

func (s *Service) OnTick(action ActionTick) {
	s.tickAction = action
}

func (s *Service) OnTock(action ActionTick) {
	s.tockAction = action
}

func (s *Service) OnEvent(scope, key string, cb ActionWithEvent) {
	if s.listeners == nil {
		s.listeners = make(map[string][]ActionWithEvent)
	}

	lid := scope + "." + key
	s.listeners[lid] = append(s.listeners[lid], cb)
}

func (s *Service) OnAnyEvent(cb ActionWithEvent) {
	if s.listeners == nil {
		s.listeners = make(map[string][]ActionWithEvent)
	}
	s.listeners["any"] = append(s.listeners["any"], cb)
}

func (s *Service) Cron(setup func(schedule CronScheduler)) {
	s.cronsetup = setup
}

func (s *Service) container(sess *Session, addr *address.Address) *serviceContainer {
	return &serviceContainer{
		svc: s,
		info: &ServiceInfo{
			addr: addr,
			slug: s.slug,
		},
	}
}

type ServiceLoader struct {
	mu       sync.Mutex
	loading  bool
	loaderCh chan struct{}
	errs     []error
	sess     *Session
	hostaddr *address.Address
	svcs     []*address.Address
}

func NewServiceLoader(sess *Session, svcs ...string) *ServiceLoader {
	loader := &ServiceLoader{
		sess:     sess,
		loaderCh: make(chan struct{}),
	}
	hostaddr, err := address.Parse(sess.Get("app.host.addr").String())
	if err != nil {
		loader.addErr(err)
		loader.addErr(fmt.Errorf(
			"%w: loader requires valid app.host.addr",
			ErrService,
		))
	}
	loader.hostaddr = hostaddr
	for _, addr := range svcs {
		svc, err := hostaddr.ResolveService(addr)
		if err != nil {
			loader.addErr(err)
		} else {
			loader.svcs = append(loader.svcs, svc)
		}
	}

	return loader
}

func (sl *ServiceLoader) Load() <-chan struct{} {
	sl.mu.Lock()
	if sl.loading {
		sl.mu.Unlock()
		return sl.loaderCh
	}
	sl.loading = true
	if len(sl.errs) > 0 {
		sl.cancel(fmt.Errorf(
			"%w: loader initializeton failed",
			ErrService,
		))
		sl.mu.Unlock()
		return sl.loaderCh
	}

	timeout := time.Duration(sl.sess.Get("app.service.loader.timeout").Int64())
	if timeout <= 0 {
		timeout = time.Duration(time.Second * 30)
		sl.sess.Log().SystemDebug(
			"service loader using default timeout",
			slog.Duration("timeout", timeout),
			slog.Int64("app.service.loader.timeout", sl.sess.Get("app.service.loader.timeout").Int64()),
		)
	}

	sl.sess.Log().SystemDebug(
		"loading services...",
		slog.String("host", sl.hostaddr.Host),
		slog.String("instance", sl.hostaddr.Instance))

	queue := make(map[string]*ServiceInfo)
	var require []string

	for _, svcaddr := range sl.svcs {
		svcaddrstr := svcaddr.String()
		info, err := sl.sess.ServiceInfo(svcaddrstr)
		if err != nil {
			sl.cancel(err)
			sl.mu.Unlock()
			return sl.loaderCh
		}
		if _, ok := queue[svcaddrstr]; ok {
			sl.cancel(fmt.Errorf(
				"%w: duplicated service request %s",
				ErrService,
				svcaddrstr,
			))
			sl.mu.Unlock()
			return sl.loaderCh
		}
		if info.Running() {
			sl.sess.Log().SystemDebug(
				"requested service is already running",
				slog.String("service", svcaddrstr),
			)
			continue
		}
		sl.sess.Log().SystemDebug(
			"requesting service",
			slog.String("service", svcaddrstr),
		)
		queue[svcaddrstr] = info
		require = append(require, svcaddrstr)
	}

	sl.mu.Unlock()

	sl.sess.Dispatch(newRequireServicesEvent(require))

	ctx, cancel := context.WithTimeout(sl.sess, timeout)

	go func() {
		defer cancel()
		ltick := time.NewTicker(time.Millisecond * 250)
		defer ltick.Stop()
		qlen := len(queue)

	loader:
		for {
			select {
			case <-ctx.Done():
				sl.mu.Lock()
				sl.cancel(ctx.Err())
				for _, status := range queue {
					if !status.Running() {
						sl.addErr(fmt.Errorf("service did not load on time %s", status.Addr().String()))
					}
				}
				sl.mu.Unlock()
				return
			case <-ltick.C:

				var loaded int
				for _, status := range queue {
					if errs := status.Errs(); errs != nil {
						sl.mu.Lock()
						for _, err := range errs {
							sl.addErr(err)
						}
						sl.mu.Unlock()
						return
					}
					if status.Running() {
						loaded++
					}
				}
				if loaded == qlen {
					break loader
				}
			}
		}
		sl.done()
	}()

	return sl.loaderCh
}

func (sl *ServiceLoader) Err() error {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	if sl.loading {
		return fmt.Errorf("%w: service loader error checked before loader finished! did you wait for .Loaded?", ErrService)
	}
	return errors.Join(sl.errs...)
}

// cancel is used internally to cancel loading
func (sl *ServiceLoader) cancel(reason error) {
	sl.sess.Log().Warn("sevice loader canceled", slog.Any("reason", reason))
	sl.addErr(reason)
	sl.loading = false
	defer close(sl.loaderCh)
	return
}

func (sl *ServiceLoader) done() {
	sl.loading = false
	defer close(sl.loaderCh)
}

func (sl *ServiceLoader) addErr(err error) {
	if err == nil {
		return
	}
	sl.errs = append(sl.errs, err)
}

func newRequireServicesEvent(svcs []string) Event {
	var payload vars.Map
	for i, url := range svcs {
		payload.Store(fmt.Sprintf("service.%d", i), url)
	}

	return NewEvent("service.loader", "require.services", &payload, nil)
}

type ServiceInfo struct {
	mu        sync.RWMutex
	slug      string
	addr      *address.Address
	running   bool
	errs      map[time.Time]error
	startedAt time.Time
	stoppedAt time.Time
}

func (s *ServiceInfo) Running() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *ServiceInfo) Errs() map[time.Time]error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.errs
}

func (s *ServiceInfo) addErr(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.errs == nil {
		s.errs = make(map[time.Time]error)
	}
	s.errs[time.Now().UTC()] = err
}

func (s *ServiceInfo) Addr() *address.Address {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.addr
}

func (s *ServiceInfo) start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = true
	s.startedAt = time.Now().UTC()
}

func (s *ServiceInfo) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = false
	s.stoppedAt = time.Now().UTC()
}

type serviceContainer struct {
	info   *ServiceInfo
	svc    *Service
	cancel context.CancelCauseFunc
	ctx    context.Context
	cron   *Cron
}

func (s *serviceContainer) initialize(sess *Session) error {
	if s.svc.initializeAction != nil {
		if err := s.svc.initializeAction(sess); err != nil {
			return err
		}
	}

	if s.svc.cronsetup != nil {
		s.cron = newCron(sess)
		s.svc.cronsetup(s.cron)
	}
	return nil
}

func (s *serviceContainer) start(sess *Session) error {
	if s.svc.startAction != nil {
		if err := s.svc.startAction(sess); err != nil {
			return err
		}
	}
	if s.cron != nil {
		sess.Log().SystemDebug("starting cron jobs", slog.String("service", s.info.Addr().String()))
		s.cron.Start()
	}
	return nil
}

func (s *serviceContainer) stop(sess *Session) error {
	if s.svc.stopAction == nil {
		return nil
	}
	err := s.svc.stopAction(sess)
	if s.cron != nil {
		sess.Log().SystemDebug("stopping cron scheduler, waiting jobs to finish", slog.String("service", s.info.Addr().String()))
		s.cron.Stop()
	}
	return err
}

func (s *serviceContainer) tick(sess *Session, ts time.Time, delta time.Duration) error {
	if s.svc.tickAction == nil {
		return nil
	}
	return s.svc.tickAction(sess, ts, delta)
}

func (s *serviceContainer) tock(sess *Session, ts time.Time, delta time.Duration) error {
	if s.svc.tockAction == nil {
		return nil
	}
	return s.svc.tockAction(sess, ts, delta)
}

func (s *serviceContainer) handleEvent(sess *Session, ev Event) {
	if s.svc.listeners == nil {
		return
	}
	lid := ev.Scope() + "." + ev.Key()
	for sk, listeners := range s.svc.listeners {
		for _, listener := range listeners {
			if sk == "any" || sk == lid {
				if err := listener(sess, ev); err != nil {
					s.info.addErr(err)
					sess.Log().Error("event handler error", err, slog.String("service", s.info.Addr().String()))
				}
			}
		}
	}
}

type CronScheduler interface {
	Job(expr string, cb Action)
}

type Cron struct {
	sess   *Session
	lib    *cron.Cron
	jobIDs []cron.EntryID
}

func newCron(sess *Session) *Cron {
	c := &Cron{}
	c.sess = sess
	c.lib = cron.New(cron.WithParser(cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)))
	return c
}

func (cs *Cron) Job(expr string, cb Action) {
	id, err := cs.lib.AddFunc(expr, func() {
		if err := cb(cs.sess); err != nil {
			cs.sess.Log().Error("job failed", err)
		}
	})
	cs.jobIDs = append(cs.jobIDs, id)
	if err != nil {
		cs.sess.Log().Error("failed to add job", err, slog.Int("id", int(id)))
		return
	}
}

func (cs *Cron) Start() error {
	for _, id := range cs.jobIDs {
		job := cs.lib.Entry(id)
		if job.Job != nil {
			go job.Job.Run()
		}
	}
	cs.lib.Start()
	return nil
}

func (cs *Cron) Stop() error {
	ctx := cs.lib.Stop()
	<-ctx.Done()
	return nil
}
