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
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/happyx"
	"strings"
	"sync"
	"time"
)

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
		u, err := happyx.ParseURL(svc)
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
