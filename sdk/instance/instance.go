// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package instance

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/sdk/app/session"
	"github.com/happy-sdk/happy/sdk/internal"
)

type Settings struct {
	// How many instances of the applications can be booted at the same time.
	Max settings.Uint `key:"max" default:"1" desc:"Maximum number of instances of the application that can be booted at the same time"`
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	b, err := settings.New(s)
	if err != nil {
		return nil, err
	}

	return b, nil
}

type Instance struct {
	id      ID
	sess    *session.Context
	pidfile string
}

var Error = errors.New("instance error")

type ID string

func NewID() ID {
	hasher := sha1.New()
	_, _ = fmt.Fprint(hasher, time.Now().UnixMilli())
	hashSum := hasher.Sum(nil)
	fullID := hex.EncodeToString(hashSum)
	return ID(fullID[:8])
}

func (id ID) String() string {
	return string(id)
}

// New creates a new instance for the application.
func New(sess *session.Context) (*Instance, error) {
	if sess == nil {
		return nil, fmt.Errorf("%w: session is nil", Error)
	}

	pidsdir := sess.Opts().Get("app.fs.path.pids").String()
	if _, err := os.Stat(pidsdir); err != nil {
		return nil, fmt.Errorf("%w: pids directory not found: %s", Error, pidsdir)
	}

	pidfiles, err := verifyPidFiles(sess, pidsdir)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read pids directory: %s", Error, err.Error())
	}

	inst := &Instance{
		id:   ID(sess.Opts().Get("app.instance.id").String()),
		sess: sess,
	}

	if len(pidfiles) >= sess.Settings().Get("app.instance.max").Value().Int() {
		sess.Log().Error("existing pid files")
		for _, pidfile := range pidfiles {
			sess.Log().Println(filepath.Join(pidsdir, pidfile.Name()))
		}
		return nil, fmt.Errorf("%w: max instances reached (%s)", Error, sess.Settings().Get("app.instance.max").String())
	}

	inst.pidfile = filepath.Join(
		pidsdir,
		fmt.Sprintf("instance-%s.pid", inst.id.String()),
	)
	internal.Log(sess.Log(), "create pid lock file", slog.String("file", inst.pidfile))

	if err := os.WriteFile(inst.pidfile, []byte(inst.sess.Opts().Get("app.pid").String()), 0644); err != nil {
		return nil, fmt.Errorf("%w: failed to write intance PID file: %s", Error, err.Error())
	}

	return inst, nil
}

func (inst *Instance) Dispose() error {
	internal.Log(inst.sess.Log(), "disposing instance", slog.String("id", inst.id.String()))
	// delete the pidfile
	if _, err := os.Stat(inst.pidfile); err == nil {
		if err := os.Remove(inst.pidfile); err != nil {
			return fmt.Errorf("failed to delete pidfile %s: %w", inst.pidfile, err)
		}
		if inst.sess != nil {
			internal.Log(inst.sess.Log(), "successfully deleted pidfile", slog.String("pidfile", inst.pidfile))
		}
	}
	return nil
}

func verifyPidFiles(sess *session.Context, pidsdir string) ([]os.DirEntry, error) {

	pidfiles, err := os.ReadDir(pidsdir)
	if err != nil {
		return nil, err
	}

	var res []os.DirEntry
	for _, pidfile := range pidfiles {
		if pidfile.IsDir() {
			continue
		}
		if ok, err := verifyPidFile(sess, pidfile); err != nil {
			return nil, err
		} else if ok {
			res = append(res, pidfile)
		}
	}
	return res, nil
}

func verifyPidFile(sess *session.Context, pidfile os.DirEntry) (ok bool, err error) {
	pidfileAbs := filepath.Join(sess.Opts().Get("app.fs.path.pids").String(), pidfile.Name())
	data, err := os.ReadFile(pidfileAbs)
	if err != nil {
		return false, err
	}
	pid := string(bytes.TrimSpace(data))
	if pid == "" {
		return false, nil
	}
	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		return false, err
	}

	p, err := os.FindProcess(pidInt)
	if err != nil {
		return false, err
	}
	if err := p.Signal(syscall.Signal(0)); err != nil {
		sess.Log().Notice(fmt.Sprintf("previous process %d: %s", pidInt, err.Error()))
		return false, os.Remove(pidfileAbs)
	}
	return true, nil
}
