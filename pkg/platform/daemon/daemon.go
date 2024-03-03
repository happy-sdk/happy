// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package daemon

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/happy-sdk/happy/pkg/settings"
)

type Settings struct {
	Name        settings.String      `key:"name" default:"Happy Prototype"`
	Description settings.String      `key:"decription" default:"Happy Prototype Daemon"`
	Slug        settings.String      `key:"slug" default:"happy-prototype"`
	Kind        Kind                 `key:"kind" default:"user"`
	Args        settings.StringSlice `key:"args" default:""`
}

type Status uint

const (
	StatusUnknown Status = iota
	StatusActive
	StatusInactive
	StatusFailed
	StatusReloading
	StatusActivating
	StatusDeactivating
)

func (s Status) String() string {
	switch s {
	case StatusUnknown:
		return "unknown"
	case StatusActive:
		return "active"
	case StatusInactive:
		return "inactive"
	case StatusFailed:
		return "failed"
	case StatusReloading:
		return "reloading"
	case StatusActivating:
		return "activating"
	case StatusDeactivating:
		return "deactivating"
	}
	return "unknown"
}

type Kind uint8

const (
	// KindUser is a daemon that runs as a user process.
	KindUser Kind = 1
	// KindSystem is a daemon that runs as a system process.
	KindSystem Kind = 2
)

func (k Kind) MarshalSetting() ([]byte, error) {
	return []byte(k.String()), nil
}

func (k Kind) SettingKind() settings.Kind {
	return settings.KindString
}

// UnmarshalSetting updates the Uint setting from a byte slice, interpreting it as an unsigned integer.
func (k Kind) UnmarshalSetting(data []byte) error {
	str := string(data)
	if str == "" {
		return nil
	}
	switch str {
	case "user":
		k = KindUser
	case "system":
		k = KindSystem
	}
	return nil
}

func (k Kind) String() string {
	switch k {
	case KindUser:
		return "user"
	case KindSystem:
		return "system"
	}
	return "unknown"
}

type Service struct {
	s           Settings
	unitpath    string
	binpath     string
	pidfilepath string
	objectPath  string
	status      Status
}

func New(s Settings) (*Service, error) {
	if s.Name == "" {
		return nil, errors.New("daemon: name is required")
	}
	if s.Slug == "" {
		return nil, errors.New("daemon: slug is required")
	}
	if s.Kind != KindUser {
		return nil, errors.New("daemon: only kind user implemented")
	}
	svc := &Service{
		s: s,
	}

	if err := svc.load(); err != nil {
		return nil, err
	}
	return svc, nil
}

func (s *Service) SetPIDFile(p string) {
	s.pidfilepath = p
}

// Install installs the service.
func (s *Service) Install() error {
	return s.sysInstall()
}

// Uninstall uninstalls the service.
func (s *Service) Uninstall() error {
	return s.sysUninstall()
}

// Enable enables the service.
func (s *Service) Enable() error {
	return s.sysEnable()
}

// Disable disables the service.
func (s *Service) Disable() error {
	return s.sysDisable()
}

func (s *Service) Start() error {
	return s.sysStart()
}

func (s *Service) Stop() error {
	return s.sysStop()
}

func (s *Service) Restart() error {
	return s.sysRestart()
}

func (s *Service) Status() (Status, error) {
	status, err := s.sysStatus()
	if err != nil {
		return StatusUnknown, err
	}
	out, err := exec.Command("systemctl", "--user", "status", s.s.Slug.String()).CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		return StatusUnknown, err
	}

	return status, nil
}

func (s *Service) Logs() error {
	return s.sysLogs()
}
