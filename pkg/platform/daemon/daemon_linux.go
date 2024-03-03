// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package daemon

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func (s *Service) load() error {
	if !isSystemdInstalled() {
		return fmt.Errorf("daemon: systemd not installed on this system")
	}
	bin, err := os.Readlink("/proc/self/exe")
	if err != nil {
		return fmt.Errorf("daemon: unable to read the binary path: %w", err)
	}
	s.binpath = bin

	if strings.HasPrefix(bin, "/tmp/") {
		return fmt.Errorf("daemon: binary is located in /tmp, which is not allowed")
	}

	switch s.s.Kind {
	case KindUser:
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("daemon: unable to get user home directory: %w", err)
		}
		s.unitpath = filepath.Join(home, ".config/systemd/user", s.s.Slug.String()+".service")
	case KindSystem:
		return fmt.Errorf("daemon: system daemons not implemented yet")
	default:
		return fmt.Errorf("daemon: unknown daemon kind: %s", s.s.Kind)
	}

	return nil
}

func (s *Service) sysInstall() error {
	if !s.isInstalled() {
		slog.Info("daemon: installing")
	} else if s.isRunning() {
		return fmt.Errorf("daemon: service is running")
	} else {
		slog.Info("daemon: reinstalling")
	}

	args := strings.Join(strings.Split(s.s.Args.String(), "|"), " ")
	unitfile := fmt.Sprintf(`[Unit]
Description=%s

[Service]
PIDFile=%s
ExecStartPre=%s -h
ExecStart=%s service run %s --system-debug
ExecStop=/bin/kill -s TERM $MAINPID
PrivateTmp=true

[Install]
WantedBy=default.target
`,
		s.s.Description.String(),
		s.pidfilepath,
		s.binpath,
		s.binpath,
		args,
	)

	if err := os.WriteFile(s.unitpath, []byte(unitfile), 0600); err != nil {
		return fmt.Errorf("daemon: unable to write unit file: %w", err)
	}

	s.systemdReload()
	return nil
}

func (s *Service) sysUninstall() error {
	if !s.isInstalled() {
		return fmt.Errorf("daemon: service is not installed")
	}
	slog.Info("uninstalling daemon")
	if s.isRunning() {
		if err := s.Stop(); err != nil {
			return fmt.Errorf("daemon: unable to uninstall service: %w", err)
		}
	}

	if err := s.sysDisable(); err != nil {
		return fmt.Errorf("daemon: unable to disable service: %w", err)
	}
	if !strings.HasSuffix(s.unitpath, ".service") {
		return fmt.Errorf("daemon: unexpected unit file name: %s", s.unitpath)
	}

	if err := os.Remove(s.unitpath); err != nil {
		return fmt.Errorf("daemon: unable to remove unit file: %w", err)
	}

	s.systemdReload()
	return nil
}

func (s *Service) sysEnable() error {
	if !s.isInstalled() {
		return fmt.Errorf("daemon: service is not installed")
	}

	slog.Info("Enabling daemon")

	return nil
}

func (s *Service) sysDisable() error {
	if !s.isInstalled() {
		return fmt.Errorf("daemon: service is not installed")
	}

	slog.Info("Disabling daemon")
	return nil
}

func (s *Service) sysStart() error {
	if !s.isInstalled() {
		return fmt.Errorf("daemon: service is not installed")
	}
	if s.isRunning() {
		return fmt.Errorf("daemon: service is already running")
	}
	out, err := exec.Command("systemctl", "--user", "start", s.s.Slug.String()).CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		slog.Error("systemctl start failed", "error", err.Error(), "output", string(out))
		return fmt.Errorf("daemon: unable to start service: %w", err)
	}

	slog.Info("starting daemon")
	return nil
}

func (s *Service) sysStop() error {
	if !s.isInstalled() {
		return fmt.Errorf("daemon: service is not installed")
	}
	if !s.isRunning() {
		return fmt.Errorf("daemon: service is not running")
	}
	slog.Info("stopping daemon")
	out, err := exec.Command("systemctl", "--user", "stop", s.s.Slug.String()).CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		slog.Error("systemctl stop failed", "error", err.Error(), "output", string(out))
		return fmt.Errorf("daemon: unable to stop service: %w", err)
	}
	return nil
}

func (s *Service) sysRestart() error {
	if !s.isInstalled() {
		return fmt.Errorf("daemon: service is not installed")
	}
	slog.Info("reloading daemon")
	if s.isRunning() {
		if err := s.Stop(); err != nil {
			return fmt.Errorf("daemon: unable to reload service: %w", err)
		}
	}
	return s.Start()
}

func (s *Service) sysStatus() (Status, error) {
	if !s.isInstalled() {
		return StatusUnknown, fmt.Errorf("daemon: service is not installed")
	}
	slog.Info("Checking daemon status")
	objpath, err := s.getObjectPath()
	if err != nil {
		slog.Error("daemon: unable to get object path:", slog.String("error", err.Error()))
		return StatusUnknown, err
	}
	out, err := exec.Command(
		"gdbus",
		"call",
		"--session",
		"--dest", "org.freedesktop.systemd1",
		"--object-path", objpath,
		"--method", "org.freedesktop.DBus.Properties.Get",
		"org.freedesktop.systemd1.Unit", "ActiveState",
	).CombinedOutput()

	if err != nil {
		slog.Error("gdbus call failed", "error", err.Error())
		return StatusUnknown, err
	}
	// check status
	re := regexp.MustCompile(`'(\w+)'`)
	matches := re.FindStringSubmatch(string(out))
	if len(matches) < 2 {
		slog.Error("Failed to parse ActiveState from gdbus output", "output", string(out))
		return StatusUnknown, fmt.Errorf("failed to parse ActiveState")
	}
	state := matches[1] // The first submatch contains the ActiveState value

	switch state {
	case "active":
		slog.Debug("Service is active")
		s.status = StatusActive
	case "reloading":
		slog.Debug("Service is reloading")
		s.status = StatusReloading
	case "inactive":
		slog.Debug("Service is inactive")
		s.status = StatusInactive
	case "failed":
		slog.Debug("Service has failed")
		s.status = StatusFailed
	case "activating":
		slog.Debug("Service is activating")
		s.status = StatusActivating
	case "deactivating":
		slog.Debug("Service is deactivating")
		s.status = StatusDeactivating
	default:
		slog.Debug("Service is in an unknown state", slog.String("state", state))
		s.status = StatusUnknown
	}

	return s.status, nil
}

func (s *Service) sysLogs() error {
	if !s.isInstalled() {
		return fmt.Errorf("daemon: service is not installed")
	}
	slog.Info("getting daemon logs")
	return nil
}

// isSystemdInstalled checks if systemd is the init system.
func isSystemdInstalled() bool {
	_, err := exec.LookPath("systemctl")
	if err != nil {
		slog.Error("systemctl command not found:", "err", err.Error())
		return false
	}

	// Read the name of the init system from /proc/1/comm.
	content, err := os.ReadFile("/proc/1/comm")
	if err != nil {
		slog.Error("Error reading /proc/1/comm:", "err", err.Error())
		return false
	}

	return bytes.Contains(content, []byte("systemd"))
}

func (s *Service) isInstalled() bool {
	if _, err := os.Stat(s.unitpath); err != nil {
		return false
	}
	return true
}

func (s *Service) isRunning() bool {
	status, err := s.sysStatus()
	if err != nil {
		slog.Error("Error getting service status", "error", err.Error())
		return false
	}
	if status == StatusActive || status == StatusActivating {
		return true
	}
	return false
}

func (s *Service) systemdReload() {
	if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
		slog.Error("daemon: unable to reload systemd:", slog.String("error", err.Error()))
		return
	}
	slog.Info("systemctl: daemon-reload ok")
}

// ExtractObjectPath extracts the object path from a gdbus call response.
func (s *Service) getObjectPath() (string, error) {

	if s.objectPath != "" {
		return s.objectPath, nil
	}
	gdbusResponse, err := exec.Command(
		"gdbus",
		"call",
		"--session",
		"--dest", "org.freedesktop.systemd1",
		"--object-path", "/org/freedesktop/systemd1",
		"--method", "org.freedesktop.systemd1.Manager.GetUnit", fmt.Sprintf("%s.service", s.s.Slug.String()),
	).CombinedOutput()
	if err != nil {
		slog.Debug("gdbus call failed", "error", err.Error(), "out", string(gdbusResponse))
		return "", fmt.Errorf("gdbus call failed: %w", err)
	}

	gdbusResponseStr := strings.TrimSpace(string(gdbusResponse))
	// Trim the leading and trailing parentheses.
	trimmed := strings.Trim(gdbusResponseStr, "()")

	// Split the string on spaces to isolate the object path.
	parts := strings.Split(trimmed, " ")

	if len(parts) == 2 {
		pathParts := strings.Split(parts[1], "'")
		if len(pathParts) >= 2 {
			// The object path should be between two apostrophes.
			s.objectPath = pathParts[1]
			slog.Debug("object path found", "path", s.objectPath)
			return s.objectPath, nil
		}
	}

	return "", fmt.Errorf("object path not found in gdbus response")
}
