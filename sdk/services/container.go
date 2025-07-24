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
	"time"

	"github.com/happy-sdk/happy/pkg/networking/address"
	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/sdk/events"
	"github.com/happy-sdk/happy/sdk/internal"
	"github.com/happy-sdk/happy/sdk/services/service"
	"github.com/happy-sdk/happy/sdk/session"
)

type Container struct {
	mu      sync.RWMutex
	info    *service.Info
	svc     *Service
	cancel  context.CancelCauseFunc
	ctx     context.Context
	cron    *serviceCron
	retries int
}

func NewContainer(sess *session.Context, addr *address.Address, svc *Service) (*Container, error) {
	if svc == nil {
		return nil, fmt.Errorf("%w: service is nil", Error)
	}
	if addr == nil {
		return nil, fmt.Errorf("%w: address is nil", Error)
	}
	container := &Container{
		info: service.NewInfo(svc.Name(), addr),
		svc:  svc,
	}

	if err := session.AttachServiceInfo(sess, container.Info()); err != nil {
		return nil, err
	}
	return container, nil
}

func (c *Container) Info() *service.Info {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.info
}

func (c *Container) Settings() service.Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.svc.settings
}

func (c *Container) Register(sess *session.Context) error {
	c.mu.RLock()
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

func (c *Container) CanRetry() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return bool(c.svc.settings.RetryOnError) &&
		int(c.svc.settings.MaxRetries) > 0 &&
		c.retries <= int(c.svc.settings.MaxRetries)
}

func (c *Container) Retries() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.retries
}

func (c *Container) Start(ectx context.Context, sess *session.Context) (err error) {
	c.mu.RLock()
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

	c.mu.Lock()
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
	sess.Log().Debug("service started", slog.String("service", c.info.Addr().String()))
	return nil
}

func (c *Container) Stop(sess *session.Context, e error) (err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if e != nil {
		sess.Log().Error(e.Error(), slog.String("service", c.info.Addr().String()))
	}
	if c.cron != nil {
		internal.Log(sess.Log(), "stopping cron scheduler, waiting jobs to finish", slog.String("service", c.info.Addr().String()))
		if err := c.cron.Stop(); err != nil {
			sess.Log().Error("error while stoping cron", slog.String("service", c.info.Addr().String()), slog.String("err", err.Error()))
		}
	}

	c.cancel(e)
	if c.svc.stopAction != nil {
		err = c.svc.stopAction(sess, e)
	}

	service.MarkStopped(c.info)

	payload := new(vars.Map)
	if err != nil {
		if errset := payload.Store("err", err); errset != nil {
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

func (c *Container) Done() <-chan struct{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ctx.Done()
}

func (c *Container) HasTick() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.svc.tickAction != nil
}

func (c *Container) Tick(sess *session.Context, ts time.Time, delta time.Duration) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.svc.tickAction == nil {
		return nil
	}
	return c.svc.tickAction(sess, ts, delta)
}

func (c *Container) Tock(sess *session.Context, delta time.Duration, tps int) error {
	c.mu.RLock()
	if c.svc.tockAction == nil {
		c.mu.RUnlock()
		return nil
	}
	if err := c.svc.tockAction(sess, delta, tps); err != nil {
		c.mu.RUnlock()
		return err
	}
	retries := c.retries
	c.mu.RUnlock()

	if retries > 0 {
		c.mu.Lock()
		c.retries = 0
		c.mu.Unlock()
	}
	return nil
}

func (c *Container) HandleEvent(sess *session.Context, ev events.Event) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.svc.listeners == nil {
		return
	}
	lid := ev.Scope() + "." + ev.Key()
	for sk, listeners := range c.svc.listeners {
		for _, listener := range listeners {
			if sk == "any" || sk == lid {
				if err := listener(sess, ev); err != nil {
					service.AddError(c.info, err)
					sess.Log().Error(Error.Error(), slog.String("service", c.info.Addr().String()), slog.String("err", err.Error()))
				}
			}
		}
	}
}

func (c *Container) Listeners() []string {
	c.mu.RLock()
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

func (c *Container) Cancel(err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	c.cancel(err)
}
