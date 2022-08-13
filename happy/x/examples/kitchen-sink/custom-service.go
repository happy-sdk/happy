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

package main

import "github.com/mkungla/happy"

type CustomService struct {
}

func customService() happy.Service {
	svc := &CustomService{}

	return svc
}

// API
func (svc *CustomService) AfterAlwaysMessaage() string {
	return "this is AfterAlways message from CustomService"
}

// SERVICE IMPLEMENTATION
func (svc *CustomService) Slug() happy.Slug {
	return nil
}

func (svc *CustomService) URL() happy.URL {
	return nil
}

func (svc *CustomService) OnInitialize(cb happy.ActionFunc)    {}
func (svc *CustomService) OnStart(cb happy.ActionWithArgsFunc) {}
func (svc *CustomService) OnStop(cb happy.ActionFunc)          {}
func (svc *CustomService) OnRequest(r happy.ServiceRouter)     {}
func (svc *CustomService) OnTick(cb happy.ActionTickFunc)      {}
func (svc *CustomService) OnTock(cb happy.ActionTickFunc)      {}

// happy.EventListener interface
func (svc *CustomService) OnAnyEvent(cb happy.ActionWithEventFunc)          {}
func (svc *CustomService) OnEvent(key string, cb happy.ActionWithEventFunc) {}
