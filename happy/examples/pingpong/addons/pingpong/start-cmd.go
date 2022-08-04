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

package pingpong

import (
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/sdk/create"
	"github.com/mkungla/varflag/v6"
	"github.com/mkungla/vars/v6"
)

func cmdStart() (happy.Command, error) {
	cmd, err := create.Command("start", 0)
	if err != nil {
		return nil, err
	}

	cmd.SetShortDesc("run ping pong example")

	// Add peer 1 py default
	cmd.RequireServices(
		"happy://addon.pingpong/services/monitor",
		"happy://addon.pingpong/services/peer-1",
	)

	mflag, err := varflag.Uint("messages", 15, "exit after n messages", "m")
	if err != nil {
		return nil, err
	}
	cmd.AddFlag(mflag)

	// configure pingpong
	cmd.Before(func(ctx happy.Session) error {
		// If we don't have direct acces to flag we can use follwoing
		// total := ctx.Flag("iterations").Var().Uint()
		total := mflag.Value()

		ctx.Set("pingpong.exit.after", total)

		// add second peer, this will block until loaded
		ctx.RequireService("happy://addon.pingpong/services/peer-2")
		return nil
	})

	// play pingpong
	cmd.Do(func(ctx happy.Session) error {

		// Start pinging with service instance-1
		if _, err = ctx.ServiceCall(
			"happy://addon.pingpong/services/peer-1",
			"add-peer",
			vars.New("service", "happy://addon.pingpong/services/peer-2"),
		); err != nil {
			return err
		}

		// Load peer 3 on demand
		ctx.RequireService("happy://addon.pingpong/services/peer-3")
		ctx.Log().Out("add peer 3...")
		if _, err = ctx.ServiceCall(
			"happy://addon.pingpong/services/peer-3",
			"add-peer",
			vars.New("service", "happy://addon.pingpong/services/peer-1"),
		); err != nil {
			return err
		}

		ctx.Log().Out("waiting...")
		<-ctx.Done()
		return nil
	})

	// read stats of pingpong
	cmd.AfterSuccess(func(ctx happy.Session) error {
		peer1stats, err := ctx.ServiceCall(
			"happy://addon.pingpong/services/peer-1",
			"stats",
		)
		if err != nil {
			return err
		}
		peer2stats, err := ctx.ServiceCall(
			"happy://addon.pingpong/services/peer-2",
			"stats",
		)
		if err != nil {
			return err
		}
		peer3stats, err := ctx.ServiceCall(
			"happy://addon.pingpong/services/peer-3",
			"stats",
		)
		if err != nil {
			return err
		}
		if ctx.Flag("json").Present() {
			ctx.Out([]any{
				peer1stats,
				peer2stats,
				peer3stats,
			})
		}

		return nil
	})
	return cmd, nil
}
