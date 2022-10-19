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
	"github.com/robfig/cron/v3"
)

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
