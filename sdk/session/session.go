// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/pkg/networking/address"
	"github.com/happy-sdk/happy/pkg/options"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/sdk/api"
	"github.com/happy-sdk/happy/sdk/events"
	"github.com/happy-sdk/happy/sdk/internal"
	"github.com/happy-sdk/happy/sdk/services/service"
)

var (
	Error          = errors.New("session")
	ErrDestroyed   = fmt.Errorf("%w:destroyed", Error)
	ErrExitSuccess = fmt.Errorf("%w:exit(0)", Error)
)

type Register interface {
	Log() logging.Logger
	Settings() *settings.Profile
	Opts() *options.Options
	Time(t time.Time) time.Time
	Has(key string) bool
	Get(key string) vars.Variable
	Context() context.Context
}

type Context struct {
	mu sync.RWMutex

	logger  logging.Logger
	profile *settings.Profile
	opts    *options.Options
	timeloc *time.Location

	err             error
	allowUserCancel bool
	disposed        bool
	valid           bool

	done chan struct{}

	evch chan<- events.Event

	isReady     bool
	released    bool
	ready       context.Context
	readyCancel context.CancelFunc
	readyEvent  events.Event

	terminated bool
	sigCtx     context.Context // os signal listener
	sigCancel  context.CancelFunc

	svss map[string]*service.Info
	apis map[string]api.Provider

	// Cached lightweight context for Context()
	ctx       context.Context
	ctxCancel context.CancelFunc
}

// Deadline returns the time when work done on behalf of this context
// should be canceled. Deadline returns ok==false when no deadline is
// set. Successive calls to Deadline return the same results.
func (c *Context) Deadline() (deadline time.Time, ok bool) {
	return
}

// Wait allows user to cancel application by pressing Ctrl+C or sending
// SIGINT or SIGTERM while application is running. By default this is not allowed.
// It returns a Done channel which blocks until application is closed by user or signal is reveived.
// Argument ctrlc=true will print to stdout "Press Ctrl+C to cancel"
func (c *Context) Wait(ctrlc bool) <-chan struct{} {
	c.mu.Lock()
	c.allowUserCancel = true
	c.mu.Unlock()

	c.mu.RLock()
	if c.sigCancel != nil {
		internal.Log(c.logger, "waiting for user cancel or session termination")
		if ctrlc {
			fmt.Println("Press Ctrl+C to cancel")
		}
	} else {
		internal.Log(c.logger, "waiting for session termination")
	}
	c.mu.RUnlock()

	return c.Done()
}

// Done enables you to hook into chan to know when application exits
// however DO NOT use that for graceful shutdown actions.
// Use Application.AddExitFunc or Wait instead.
func (c *Context) Done() <-chan struct{} {
	c.mu.Lock()
	if c.done == nil {
		c.done = make(chan struct{})
	}
	d := c.done
	c.mu.Unlock()
	return d
}

// Err returns session error if any or nil
// If Done is not yet closed, Err returns nil.
// If Done is closed, Err returns a non-nil error explaining why:
// Canceled if the context was canceled
// or DeadlineExceeded if the context's deadline passed.
// After Err returns a non-nil error, successive calls to Err return the same error.
func (c *Context) Err() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	err := c.err
	return err
}

func (*Context) String() string {
	return "happy.Session"
}

// Value returns the value associated with this context for key, or nil
func (c *Context) Value(key any) any {
	c.mu.RLock()
	disposed := c.disposed
	c.mu.RUnlock()
	if disposed {
		return ErrDestroyed
	}

	var (
		err           error
		shouldDestroy bool
	)

	switch k := key.(type) {
	case string:
		if v, ok := c.Opts().Load(k); ok {
			return v
		}
	case *int:
		c.mu.RLock()
		if !c.released {
			if c.sigCtx != nil && c.sigCtx.Err() != nil {
				shouldDestroy = true
				if c.allowUserCancel {
					err = ErrExitSuccess
				} else {
					err = c.sigCtx.Err()
				}
			}
		}
		c.mu.RUnlock()
	}
	if shouldDestroy {
		c.Destroy(err)
	}
	return nil
}

// Release releases os.Interrupt and os.Kill signals.
// Caller becomes responsible of signals.
func (c *Context) Release() {
	c.mu.Lock()
	c.released = true
	sigCancel := c.sigCancel
	c.sigCancel = nil
	logger := c.logger
	c.mu.Unlock()

	if sigCancel != nil {
		sigCancel()
	}

	internal.Log(logger, "session released SIGINT, SIGKILL signals")
}

// Destroy can be called do destroy session.
func (c *Context) Destroy(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.err != nil {
		// prevent Destroy to be called multiple times
		// e.g. by sig release or other contexts.
		// however update error if it is not exit success
		if errors.Is(c.err, ErrExitSuccess) && !errors.Is(err, ErrExitSuccess) {
			c.err = err
		}
		return
	}

	c.disposed = true
	c.ctxCancel()

	// s.err is nil otherwise we would not be here
	c.err = err
	if c.err == nil {
		c.err = ErrExitSuccess
	}

	if c.readyCancel != nil {
		c.readyCancel()
	}

	if c.sigCancel != nil {
		c.sigCancel()
		c.sigCancel = nil
	}

	if c.done != nil {
		close(c.done)
		c.done = nil
	}
}

// Terminate is used to terminate session. and called only internally
func (c *Context) terminateSession() {
	if c.evch != nil {
		close(c.evch)
		c.evch = nil
	}
}

func (c *Context) Log() logging.Logger {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.logger
}

// Settings returns a map of all settings which are defined by application
// and are user configurable.
func (c *Context) Settings() *settings.Profile {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.profile == nil || !c.profile.Loaded() {
		c.logger.Warn("session profile not loaded, while accessing settings profile")
	}
	profile := c.profile
	return profile
}

// Opts returns a map of all options which are defined by application
// turing current session life cycle.
func (c *Context) Opts() *options.Options {
	c.mu.RLock()
	defer c.mu.RUnlock()
	opts := c.opts
	return opts
}

// Valid returns true if the session is valid, false otherwise.
func (c *Context) Valid() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.valid
}

// Time returns the time in the session's time location.
func (s *Context) Time(t time.Time) time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return t.In(s.timeloc)
}

func (c *Context) Has(key string) bool {
	if c.profile != nil && c.profile.Has(key) {
		return true
	}
	return c.opts.Accepts(key)
}

func (c *Context) Get(key string) vars.Variable {
	if !c.Has(key) {
		return vars.EmptyVariable
	}
	if c.profile != nil && c.profile.Has(key) {
		return c.profile.Get(key).Value()
	}
	return c.opts.Get(key).Variable()
}

// Ready returns channel which blocks until session considers application to be ready.
// It is ensured that Ready closes before root or command Do function is called.
func (c *Context) Ready() <-chan struct{} {
	c.mu.RLock()
	d := c.ready.Done()
	if !c.isReady {
		internal.Log(c.Log(), "waiting session to become ready")
	}
	c.mu.RUnlock()
	return d
}

func (c *Context) Dispatch(ev events.Event) {
	if ev == nil {
		c.Log().Warn("received <nil> event")
		return
	}

	c.mu.Lock()

	if !c.isReady && ev == c.readyEvent {
		c.readyEvent = nil
		c.isReady = true
		c.readyCancel()
		c.mu.Unlock()
		internal.Log(c.Log(), "session is ready")
		return
	}

	if ev == internal.TerminateSessionEvent {
		c.terminateSession()
		c.mu.Unlock()
		return
	}

	if c.evch == nil {
		c.Log().Error("event channel is closed, dropping event", slog.String("event", ev.String()))
		c.mu.Unlock()
		return
	}

	c.evch <- ev
	c.mu.Unlock()
}

func (c *Context) CanRecover(err error) bool {

	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.err != nil && !errors.Is(err, ErrExitSuccess) {
		return false
	}

	if c.allowUserCancel && c.terminated {
		c.Log().Warn("session terminated by user")
		return true
	}

	if err == nil {
		return true
	}

	return false
}

func (c *Context) ServiceInfo(svcaddr string) (*service.Info, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	hostaddr, err := address.Parse(c.opts.Get("app.address").String())
	if err == nil {
		if addr, err := hostaddr.ResolveService(svcaddr); err == nil {
			svcaddr = addr.Path()
		}
	}
	svcinfo, ok := c.svss[svcaddr]
	if !ok {
		return nil, fmt.Errorf("%w: unknown service %s", Error, svcaddr)
	}
	return svcinfo, nil
}

func (c *Context) Describe(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.profile != nil && c.profile.Has(key) {
		return c.profile.Get(key).Description()
	}
	return c.opts.Describe(key)
}

func (c *Context) AttachAPI(slug string, api api.Provider) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.apis[slug]; ok {
		return fmt.Errorf("%w: api %s already registered", Error, slug)
	}
	c.apis[slug] = api
	return nil
}

func (c *Context) start() (err error) {
	c.ready, c.readyCancel = context.WithCancel(context.Background())

	c.sigCtx, c.sigCancel = signal.NotifyContext(c, os.Interrupt, os.Kill)

	if timelocStr := c.Get("app.datetime.location").String(); timelocStr != "" {
		c.timeloc, err = time.LoadLocation(timelocStr)
		if err != nil {
			return fmt.Errorf("failed to load time location: %w", err)
		}
	} else {
		c.timeloc = time.Local
	}
	internal.LogDepth(c.logger, 1, "session started")
	return err
}

// Context returns a lightweight context that becomes done when the session is done.
// It is cached during session initialization, ensuring minimal overhead for reuse in
// context-aware functions. Use this for efficient cancellation handling in sync with
// the session's lifecycle, instead of the full session context.
func (c *Context) Context() context.Context {
	c.mu.RLock()
	ctx := c.ctx
	c.mu.RUnlock()
	return ctx
}

func APIBySlug[API api.Provider](sess *Context, apiSlug string) (api API, err error) {
	capi, ok := sess.apis[apiSlug]
	if !ok {
		return api, fmt.Errorf("%w: %s named api not registered", Error, apiSlug)
	}
	if aa, ok := capi.(API); ok {
		return aa, nil
	}
	return api, fmt.Errorf("%w: unable to cast %s API to given type", Error, apiSlug)
}

func API[API api.Provider](sess *Context, api *API) error {
	for _, lapi := range sess.apis {
		if fapi, ok := lapi.(API); ok {
			*api = fapi
			return nil
		}
	}
	return fmt.Errorf("%w: unable to find API for given type", Error)
}

func AttachServiceInfo(c *Context, svcinfo *service.Info) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if svcinfo == nil {
		return fmt.Errorf("%w: service info is nil", Error)
	}

	if c.svss == nil {
		c.svss = make(map[string]*service.Info)
	}

	if !svcinfo.Valid() {
		return fmt.Errorf("%w: service info is invalid name(%s) addr(%s)", Error, svcinfo.Name(), svcinfo.Addr())

	}
	if _, ok := c.svss[svcinfo.Addr().String()]; ok {
		c.Log().NotImplemented("service info already attached", slog.String("service", svcinfo.Addr().String()))
		return fmt.Errorf("%w: service info already attached (%s)", Error, svcinfo.Name())
	}
	c.svss[svcinfo.Addr().String()] = svcinfo
	return nil
}

// Config is a session builder used internally by the SDK to initialize a session.
type Config struct {
	Logger       logging.Logger
	Profile      *settings.Profile
	Opts         *options.Options
	TimeLocation *time.Location
	ReadyEvent   events.Event
	EventCh      chan<- events.Event
	APIs         map[string]api.Provider
	Context      context.Context
	CancelFunc   context.CancelFunc
}

func (c *Config) Init() (*Context, error) {
	sess := &Context{
		apis: c.APIs,
	}

	// shared context
	if c.Context != nil {
		sess.ctx, sess.ctxCancel = c.Context, c.CancelFunc
	}

	if c.Logger == nil {
		return nil, fmt.Errorf("%w: logger is nil", Error)
	}
	sess.logger = c.Logger

	if c.Profile == nil {
		return nil, fmt.Errorf("%w: profile is nil", Error)
	}
	sess.profile = c.Profile

	if c.Opts == nil {
		return nil, fmt.Errorf("%w: options is nil", Error)
	}

	if c.ReadyEvent == nil {
		return nil, fmt.Errorf("%w: ready event is nil", Error)
	}
	sess.readyEvent = c.ReadyEvent

	if c.EventCh == nil {
		return nil, fmt.Errorf("%w: event channel is nil", Error)
	}
	sess.evch = c.EventCh

	sess.opts = c.Opts

	if err := sess.start(); err != nil {
		return nil, fmt.Errorf("%w: %v", Error, err)
	}

	return sess, nil
}

var readyEvent = events.New("session", "ready")

func ReadyEvent() events.Event {
	return readyEvent.Create(time.Now().UnixNano(), nil)
}
