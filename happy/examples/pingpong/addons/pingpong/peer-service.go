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
	"fmt"
	"sync"
	"time"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/addon"
	"github.com/mkungla/vars/v6"
)

type Stats struct {
	Recieved uint64
	Sent     uint64
	Peers    *vars.Collection
}

type PeerService struct {
	mu           sync.RWMutex
	id           uint
	recieved     uint64
	sent         uint64
	peers        *vars.Collection
	pendingPeers []string
}

func NewPeerService(id uint) *PeerService {
	return &PeerService{
		id:    id,
		peers: new(vars.Collection),
	}
}

func (s *PeerService) Name() string        { return fmt.Sprintf("Peer %d", s.id) }
func (s *PeerService) Slug() string        { return fmt.Sprintf("peer-%d", s.id) }
func (s *PeerService) Description() string { return "peer service" }

func (s *PeerService) Tick(ctx happy.Session, ts time.Time, delta time.Duration) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.pendingPeers) == 0 {
		return nil
	}

	// initial ping fo new peer
	for _, purl := range s.pendingPeers {
		s.peers.Store(purl, "destination")

		ctx.Dispatch(
			"ping",
			vars.New("src", fmt.Sprintf("happy://addon.pingpong/services/peer-%d", s.id)),
			vars.New("dest", purl),
		)
		s.sent++
	}

	s.pendingPeers = nil
	return nil
}

func (s *PeerService) Call(fn string, args ...vars.Variable) (any, error) {
	switch fn {
	case "add-peer":
		if len(args) != 1 || args[0].Key() != "service" {
			return nil, fmt.Errorf("%s: send expects 1 argument (service = url)", s.Slug())
		}

		s.mu.Lock()
		defer s.mu.Unlock()

		purl := args[0].String()
		if s.peers.Has(purl) {
			return nil, fmt.Errorf("%s: has already peer %s", s.Slug(), purl)
		}

		s.pendingPeers = append(s.pendingPeers, purl)

		return nil, nil
	case "stats":
		return Stats{
			Recieved: s.recieved,
			Sent:     s.sent,
			Peers:    s.peers,
		}, nil
	}
	return nil, fmt.Errorf("%s: unknown call %s", s.Slug(), fn)
}

func (s *PeerService) OnEvent(ctx happy.Session, ev happy.Event) {
	if ev.Key != "ping" && ev.Key != "pong" {
		return
	}
	key := fmt.Sprintf("happy://addon.pingpong/services/peer-%d", s.id)
	if ev.Payload.Get("dest").String() != key {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.recieved++
	s.sent++

	ctx.Log().Outf("%s: %s %s", s.Slug(), ev.Key, ev.Payload.Get("src").String())

	switch ev.Key {
	case "ping":
		if !s.peers.Has(ev.Payload.Get("src").String()) {
			s.peers.Set(ev.Payload.Get("src").String(), "incoming")
		}
		ctx.Dispatch(
			"pong",
			vars.New("src", key),
			vars.New("dest", ev.Payload.Get("src").String()),
		)
	case "pong":
		ctx.Dispatch(
			"ping",
			vars.New("src", key),
			vars.New("dest", ev.Payload.Get("src").String()),
		)
	}
}

// implement interface
func (s *PeerService) Version() happy.Version             { return addon.Version{} }
func (s *PeerService) Initialize(ctx happy.Session) error { return nil }
func (s *PeerService) Start(ctx happy.Session) error      { return nil }
func (s *PeerService) Stop(ctx happy.Session) error       { return nil }
