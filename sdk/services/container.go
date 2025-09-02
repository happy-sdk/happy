// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/happy-sdk/happy/pkg/networking/address"
	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/sdk/events"
	"github.com/happy-sdk/happy/sdk/internal"
	"github.com/happy-sdk/happy/sdk/services/service"
	"github.com/happy-sdk/happy/sdk/session"
)

type Container struct {
	mu       sync.RWMutex
	info     *service.Info
	svc      *Service
	cancel   context.CancelCauseFunc
	ctx      context.Context
	cron     *serviceCron
	retries  int
	lockInfo atomic.Value
}

func NewContainer(sess *session.Context, addr *address.Address, svc *Service) (*Container, error) {
	if svc == nil {
		return nil, fmt.Errorf("%w: service is nil", Error)
	}
	if addr == nil {
		return nil, fmt.Errorf("%w: address is nil", Error)
	}
	container := &Container{
		info: service.NewInfo(svc.Slug(), svc.Name(), addr, time.Duration(svc.settings.LoaderTimeout)),
		svc:  svc,
	}

	if err := session.AttachServiceInfo(sess, container.Info()); err != nil {
		return nil, err
	}
	return container, nil
}

func (c *Container) Cancel(err error) {
	c.rlock("cancel service")
	defer c.mu.RUnlock()
	c.cancel(err)
}

func (c *Container) CanRetry() bool {
	c.rlock("check if service can retry")
	defer c.mu.RUnlock()
	return bool(c.svc.settings.RetryOnError) &&
		int(c.svc.settings.MaxRetries) > 0 &&
		c.retries <= int(c.svc.settings.MaxRetries)
}

func (c *Container) Done() <-chan struct{} {
	c.rlock("get done channel")
	defer c.mu.RUnlock()
	return c.ctx.Done()
}

func (c *Container) Info() *service.Info {
	c.rlock("get info")
	defer c.mu.RUnlock()
	return c.info
}

func (c *Container) IsLocked() bool {
	locked := c.mu.TryLock()
	if locked {
		c.mu.Unlock()
	}
	return !locked
}

func (c *Container) HandleEvent(sess *session.Context, ev events.Event) {
	c.rlock("handle event")
	defer c.mu.RUnlock()
	if c.svc.listeners == nil || !c.info.Running() {
		return
	}
	lid := ev.Scope() + "." + ev.Key()
	for sk, listeners := range c.svc.listeners {
		for _, listener := range listeners {
			if sk == "any" || sk == lid {
				if err := listener(sess, ev); err != nil {
					service.AddError(c.info, err)
					sess.Log().Error(err.Error(), slog.String("service", c.info.Addr().String()))
				}
			}
		}
	}
}

func (c *Container) HasTick() bool {
	c.rlock("check for tick")
	defer c.mu.RUnlock()
	return c.svc.tickAction != nil
}

func (c *Container) Listeners() []string {
	c.rlock("get listeners")
	defer c.mu.RUnlock()
	if c.svc.listeners == nil {
		return nil
	}
	var listeners []string
	for sk := range c.svc.listeners {
		listeners = append(listeners, sk)
	}
	return listeners
}

func (c *Container) Register(sess *session.Context) error {
	c.rlock("register service")
	defer c.mu.RUnlock()
	initerrs := errors.Join(c.svc.errs...)
	if initerrs != nil {
		return fmt.Errorf("%w(%s): service failed to initialize: %w", Error, c.info.Name(), initerrs)
	}
	if c.svc.registerAction != nil {
		if err := c.svc.registerAction(sess); err != nil {
			service.AddError(c.info, err)
			return err
		}
	}

	if c.svc.cronsetup != nil {
		c.cron = newCron(sess)
		c.svc.cronsetup(c.cron)
	}
	sess.Log().Debug("service registered",
		slog.String("name", c.info.Name()),
		slog.String("service", c.info.Addr().String()))
	return nil
}

func (c *Container) Retries() int {
	c.rlock("get retries")
	defer c.mu.RUnlock()
	return c.retries
}

func (c *Container) Settings() service.Config {
	c.rlock("get settings")
	defer c.mu.RUnlock()
	return c.svc.settings
}

func (c *Container) Start(ectx context.Context, sess *session.Context) (err error) {
	c.rlock("starting service")
	retries := c.retries
	if c.svc.settings.RetryOnError && c.svc.settings.MaxRetries > 0 && c.retries > 0 {
		if c.retries > int(c.svc.settings.MaxRetries) {
			c.mu.RUnlock()
			return fmt.Errorf("%w: service start cancelled: max retries reached", Error)
		}
		if c.svc.settings.RetryBackoff > 0 {
			ctx, cancel := context.WithTimeout(ectx, time.Duration(c.svc.settings.RetryBackoff))
			defer cancel()
			<-ctx.Done()
			if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
				c.mu.RUnlock()
				return fmt.Errorf("%w: service start cancelled: %s", Error, ctx.Err())
			}
			c.mu.RUnlock()
		}
	} else {
		c.mu.RUnlock()
	}

	if retries > 0 {
		c.lock(fmt.Sprintf("service start retry (%d)", retries))
	} else {
		c.lock("calling service start action")
	}
	defer c.mu.Unlock()

	c.retries++
	if c.svc.startAction != nil {
		if err := c.svc.startAction(sess); err != nil {
			return err
		}
	}
	if c.cron != nil {
		internal.Log(sess.Log(), "starting cron jobs", slog.String("service", c.info.Addr().String()))
		if err := c.cron.Start(); err != nil {
			return err
		}
	}

	c.ctx, c.cancel = context.WithCancelCause(ectx) // with engine context

	payload := new(vars.Map)

	if err == nil {
		service.MarkStarted(c.info)
	} else {
		service.AddError(c.info, err)
		if errset := payload.Store("err", err); errset != nil {
			return errors.Join(errset, err)
		}
	}

	kv := map[string]any{
		"addr":       c.info.Addr().String(),
		"running":    c.info.Running(),
		"started.at": c.info.StartedAt(),
	}
	for k, v := range kv {
		if err := payload.Store(k, v); err != nil {
			return err
		}
	}

	sess.Dispatch(service.StartedEvent.Create(c.info.Name(), payload))
	sess.Log().Debug("service started",
		slog.String("service", c.info.Addr().String()),
		slog.String("name", c.info.Name()),
	)
	return nil
}

func (c *Container) Stop(sess *session.Context, e error) (err error) {
	c.rlock("calling service stop")
	defer c.mu.RUnlock()
	if !c.info.Running() {
		sess.Dispatch(service.StoppedEvent.Create(c.info.Name(), nil))
		return nil
	}
	if e != nil {
		sess.Log().Error(e.Error(), slog.String("service", c.info.Addr().String()))
	}
	if c.cron != nil {
		internal.Log(sess.Log(), "stopping cron scheduler, waiting jobs to finish", slog.String("service", c.info.Addr().String()))
		if err := c.cron.Stop(); err != nil {
			sess.Log().Error("error while stoping cron", slog.String("service", c.info.Addr().String()), slog.String("err", err.Error()))
		}
	}

	cancelErr := e
	if cancelErr == nil {
		cancelErr = ErrServiceStopped
	}
	c.cancel(cancelErr)
	if c.svc.stopAction != nil {
		err = c.svc.stopAction(sess, e)
	}

	service.MarkStopped(c.info)

	payload := new(vars.Map)
	if err != nil {
		if errset := payload.Store("err", err.Error()); errset != nil {
			err = errors.Join(errset, err)
		}
	}

	kv := map[string]any{
		"name":       c.info.Name(),
		"addr":       c.info.Addr().String(),
		"running":    c.info.Running(),
		"stopped.at": c.info.StoppedAt(),
	}

	for k, v := range kv {
		if errset := payload.Store(k, v); errset != nil {
			err = errors.Join(err, errset)
		}
	}
	sess.Dispatch(service.StoppedEvent.Create(c.info.Name(), payload))

	sess.Log().Debug("service stopped", slog.String("service", c.info.Addr().String()))
	return err
}

func (c *Container) Tick(sess *session.Context, ts time.Time, delta time.Duration) error {
	c.rlock("call servicetick ")
	defer c.mu.RUnlock()
	if c.svc.tickAction == nil {
		return nil
	}
	return c.svc.tickAction(sess, ts, delta)
}

func (c *Container) Tock(sess *session.Context, delta time.Duration, tps int) error {
	c.rlock("tock calling service")
	if c.svc.tockAction == nil {
		c.mu.RUnlock()
		return nil
	}
	if err := c.svc.tockAction(sess, delta, tps); err != nil {
		c.mu.RUnlock()
		return err
	}
	retries := c.retries
	c.rlock("performing service tock cleanup")

	if retries > 0 {
		c.lock("when resetting service tock retries")
		c.retries = 0
		c.mu.Unlock()
	}
	return nil
}

func (c *Container) ForceShutdown(sess *session.Context, err error) error {
	if !c.IsLocked() {
		c.lock("force shutdown")
	} else {
		sess.Log().Warn(fmt.Sprintf("service previously locked when %s", c.lockInfo.Load().(string)))
	}

	if c.cancel != nil {
		c.cancel(err)
	}
	c.svc.tickAction = nil
	c.svc.tockAction = nil
	c.mu.Unlock()
	return c.Stop(sess, err)
}

func (c *Container) lock(lockInfo string) {
	c.mu.Lock()
	c.lockInfo.Store(lockInfo)
}

func (c *Container) rlock(lockInfo string) {
	c.mu.RLock()
	c.lockInfo.Store(lockInfo)
}
