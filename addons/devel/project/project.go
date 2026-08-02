// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package project

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/goccy/go-yaml"
	"github.com/happy-sdk/happy/addons/devel/pkg/gomodule"
	"github.com/happy-sdk/happy/lib/scm/gitutils"
	"github.com/happy-sdk/happy/pkg/devel/goutils"
	"github.com/happy-sdk/happy/pkg/fsutils"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/version"
	"github.com/happy-sdk/happy/sdk/session"
)

const (
	ConfigFileName   = ".happy.yaml"
	ConfigVersion    = version.Version("v1.0.0")
	ConfigVersionMin = version.Version("v1.0.0")
)

var (
	Error             = errors.New("project")
	ErrOpeningProject = fmt.Errorf("%w: opening project", Error)
)

// IsProjectDir reports whether dir is a Happy project root.
// A directory is considered a project root if it contains:
//  1. The Happy config file (ConfigFileName = "happy.yml").
//  2. A Git repository marker (via git.IsRepository).
//
// It returns true if either condition is met.
func IsProjectDir(dir string, all bool) bool {
	if !fsutils.IsDir(dir) {
		return false
	}
	if _, exists := ContainsHappyConfigFile(dir); exists {
		return true
	}

	if _, yes, err := goutils.DependsOnHappy(dir); yes && err == nil {
		return true
	}

	if !all {
		return false
	}
	return gitutils.IsRepository(dir)
}

// ContainsHappyConfigFile reports whether dir contains the Happy config file.
// It returns true if the file exists and is a regular file.
// If the file exists, it returns the absolute path to the file.
func ContainsHappyConfigFile(dir string) (string, bool) {
	cnfFilePath := filepath.Join(dir, ConfigFileName)
	stat, err := os.Stat(cnfFilePath)
	if err != nil || !stat.Mode().IsRegular() {
		return "", false
	}
	return cnfFilePath, true
}

type DirInfo struct {
	Path           string          `json:"path"`
	HasConfigFile  bool            `json:"has_config_file"`
	ConfigFile     string          `json:"config_file"`
	HappyVersion   version.Version `json:"happy_version"`
	Version        version.Version `json:"version"`
	DependsOnHappy bool            `json:"depends_on_happy"`
	HasGit         bool            `json:"has_git"`
}

func Detect(dir string) (info DirInfo, found bool, err error) {
	info.Path, err = filepath.Abs(dir)
	if err != nil {
		return
	}

	info.ConfigFile, info.HasConfigFile = ContainsHappyConfigFile(dir)
	info.HappyVersion, info.DependsOnHappy, err = goutils.DependsOnHappy(dir)
	if err != nil {
		return
	}

	info.Version = version.OfDir(dir)
	info.HasGit = gitutils.IsRepository(dir)

	found = info.HasConfigFile || info.DependsOnHappy || info.HasGit
	return
}

// FindProjectDir locates the root of a Happy project by ascending from wd.
// It looks for either:
//  1. A directory containing the Happy config file (ConfigFileName = "happy.yml").
//  2. A Git repository marker (via git.IsRepository).
//
// Returns:
//
//	dir:   absolute path to the discovered project root, or original wd if none found.
//	found: true if a project root is detected; false otherwise.
//	err:   any error encountered resolving wd to an absolute path.
//
// The search ascends parent directories until it reaches the filesystem root,
// preferring the outermost match so that a directory inside a monorepo resolves
// to the monorepo root rather than to itself.
//
// A Git repository root stops that ascent. A repository checked out inside
// another repository - as with the happy-sdk workspace, which holds each
// repository under a gitignored src/ directory of an outer repository - is a
// separate project, not a subdirectory of the outer one. Without this boundary
// every project under such a root resolves to the outer repository, and its own
// .happy.yaml is never read. Module directories within a monorepo are not
// repository roots, so monorepo detection is unaffected.
func FindProjectDir(wd string) (dir string, found bool, err error) {
	dir, err = filepath.Abs(wd)
	if err != nil {
		return wd, false, err
	}

	for {
		if IsProjectDir(dir, true) {
			if !gitutils.IsRepository(dir) {
				if pdir, found, err := FindProjectDir(filepath.Dir(dir)); err != nil {
					return pdir, found, err
				} else if found {
					return pdir, true, nil
				}
			}
			return dir, true, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir || dir == "/" || dir == "." {
			break
		}
		dir = parent
	}

	return wd, false, nil
}

type Project struct {
	mu        sync.RWMutex
	dir       DirInfo
	cnf       *settings.Profile
	gomodules []*gomodule.Package
	dist      string
}

func Open(sess *session.Context, dir string) (*Project, error) {
	dirInfo, found, err := Detect(dir)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("no project found at %s", dir)
	}

	prj := &Project{dir: dirInfo}

	if err := prj.loadConfig(sess); err != nil {
		return nil, err
	}

	prj.dist = filepath.Join(prj.dir.Path, prj.cnf.Get("releaser.dist").String())

	return prj, nil
}

func (prj *Project) Config() *settings.Profile {
	prj.mu.RLock()
	defer prj.mu.RUnlock()
	return prj.cnf
}

func (prj *Project) Dir() DirInfo {
	prj.mu.RLock()
	defer prj.mu.RUnlock()
	return prj.dir
}

func (prj *Project) Dist() string {
	prj.mu.RLock()
	defer prj.mu.RUnlock()
	return prj.dist
}

func (prj *Project) GoModules(sess *session.Context) ([]*gomodule.Package, error) {
	prj.mu.Lock()
	defer prj.mu.Unlock()

	if prj.gomodules != nil {
		return prj.gomodules, nil
	}

	modules, err := gomodule.LoadAll(sess, prj.dir.Path)
	if err != nil {
		return nil, err
	}

	prj.gomodules = modules
	return modules, nil
}

func (prj *Project) loadConfig(sess *session.Context) (err error) {
	cnf := &Config{}

	if prj.dir.HasGit {
		cnf.Git, err = gitutils.LoadConfig(sess, prj.dir.Path)
		if err != nil {
			return err
		}
	}

	cnfBp, err := cnf.Blueprint()
	if err != nil {
		return err
	}

	cnfSchema, err := cnfBp.Schema("project", ConfigVersion)
	if err != nil {
		return err
	}

	pref := &settings.Preferences{}
	if prj.dir.HasConfigFile {
		prefFile, err := os.Open(prj.dir.ConfigFile)
		if err != nil {
			return err
		}
		defer func() {
			if err := prefFile.Close(); err != nil {
				sess.Log().Error(
					"failed to close project configuration file",
					slog.String("path", prj.dir.ConfigFile),
					slog.String("error", err.Error()))
			}
		}()
		cnfDecoder := yaml.NewDecoder(prefFile, yaml.UseJSONUnmarshaler())
		if err := cnfDecoder.Decode(pref); err != nil {
			sess.Log().Error(
				err.Error(),
				slog.String("path", prj.dir.ConfigFile))
			return err
		}
	}

	cnfProfile, err := cnfSchema.Profile("latest", pref)
	if err != nil {
		sess.Log().Error(
			err.Error(),
			slog.String("path", prj.dir.ConfigFile))
		return err
	}

	prj.cnf = cnfProfile
	return nil
}
