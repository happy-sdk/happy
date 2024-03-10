// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package instance

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/sdk/networking/address"
)

type Settings struct {
	Slug       settings.String `key:"slug" default:"" mutation:"once"`
	Max        settings.Uint   `key:"max" default:"1" mutation:"once"`
	ReverseDNS settings.String `key:"reverse_dns" default:"" mutation:"once"`
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	b, err := settings.New(s)
	if err != nil {
		return nil, err
	}

	return b, nil
}

type Instance struct {
	mu          sync.RWMutex
	addr        *address.Address
	slug        string
	pidFilePath string
	id          int
	max         int
	pid         int
}

func New(slug, rdns string) (*Instance, error) {
	curr, err := address.Current()
	if err != nil {
		return nil, err
	}
	if len(slug) == 0 {
		return nil, errors.New("instance can not be created without slug")
	}
	a, err := curr.Parse(slug)
	if err != nil {
		return nil, err
	}
	return &Instance{
		addr: a,
		slug: slug,
		max:  1,
	}, nil
}

func (i *Instance) Address() *address.Address {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.addr
}

func (i *Instance) Boot(pidsDir string) error {
	// depending on Max setting, we can boot multiple instances
	// lets check if we are within the limit and assign id for current instance.
	pidfiles, err := os.ReadDir(pidsDir)
	if err != nil {
		return fmt.Errorf("failed to read PIDs directory: %w", err)
	}
	instanceID := 1
	existingInstances := make(map[int]bool)
	for _, file := range pidfiles {
		fmt.Println(file.Name())
		if strings.HasPrefix(file.Name(), i.slug+"-") && strings.HasSuffix(file.Name(), ".pid") {
			after, found := strings.CutPrefix(file.Name(), i.slug+"-")
			if !found {
				return fmt.Errorf("unexpected PID file name: %s", file.Name())
			}

			previd, found := strings.CutSuffix(after, ".pid")
			if !found {
				return fmt.Errorf("unexpected PID file name: %s", file.Name())
			}

			id, err := strconv.Atoi(previd)
			if err == nil {
				existingInstances[id] = true
			}
		}
	}
	// Find the next available instanceID.
	for existingInstances[instanceID] && instanceID <= i.max {
		instanceID++
	}
	if instanceID > i.max {
		return fmt.Errorf("maximum number of instances (%d) reached", i.max)
	}
	i.id = instanceID

	i.pidFilePath = filepath.Join(pidsDir, fmt.Sprintf("%s-%d.pid", i.slug, instanceID))
	i.pid = os.Getpid()
	if err := os.WriteFile(i.pidFilePath, []byte(strconv.Itoa(i.pid)), 0644); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}
	return nil
}

func (i *Instance) ID() int {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.id
}

func (i *Instance) PID() int {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.pid
}

func (i *Instance) Shutdown() error {
	i.mu.RLock()
	defer i.mu.RUnlock()
	if i.pidFilePath == "" {
		return errors.New("instance is not booted, missing pid file")
	}
	if _, err := os.Stat(i.pidFilePath); err == nil {
		if err := os.Remove(i.pidFilePath); err != nil {
			return fmt.Errorf("failed to remove PID file: %w", err)
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to stat PID file: %w", err)
	}

	return nil
}
