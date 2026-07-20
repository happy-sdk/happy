// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors
package projects

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"iter"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/happy-sdk/happy/addons/devel/project"
	"github.com/happy-sdk/happy/lib/scm/gitutils"
	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/strings/bexp"
	"github.com/happy-sdk/happy/sdk/session"
)

var (
	Error              = errors.New("projects")
	ErrNoProjectsFound = fmt.Errorf("%w: no projects found", Error)
)

type Settings struct {
	SearchPaths       settings.StringSlice `key:"search_paths,save" mutation:"once" desc:"The search paths to look for projects"`
	SearchPathIgnore  settings.StringSlice `key:"search_path_ignore,save" mutation:"once" desc:"The search paths to ignore when looking for projects"`
	CacheListDisabled settings.Bool        `key:"cache_list_disabled,save" mutation:"once" desc:"Disable caching of project list"`
	CacheListLifetime settings.Duration    `key:"cache_list_lifetime,save" default:"1h" mutation:"once" desc:"The lifetime of the project list cache"`
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	return settings.New(s)
}

type API struct {
	mu sync.RWMutex
}

func New() *API {
	// fmt.Println(sess.Get("devel.projects.search_path_ignore"))
	// fmt.Println(sess.Get("devel.projects.search_paths"))
	return &API{}
}

type cacheList struct {
	CreatedAt time.Time         `json:"created_at"`
	Prjs      []project.DirInfo `json:"projects"`
}

func (api *API) List(sess *session.Context, withSubprojects, all, fresh bool) (iter.Seq[project.DirInfo], error) {
	cacheEnabled := !sess.Get("devel.projects.cache_list_disabled").Bool()

	// Try to load from cache first
	if cacheEnabled && !fresh {
		if projects, found := api.loadFromCache(sess, withSubprojects, all); found {
			return createIterator(projects), nil
		}
	}

	// Generate fresh project list
	projects, err := api.generateFreshProjectList(sess, withSubprojects, all)
	if err != nil {
		return nil, err
	}

	// Save to cache if enabled
	if cacheEnabled {
		if err := api.saveToCache(sess, projects, withSubprojects, all); err != nil {
			// Log error but don't fail the operation
			sess.Log().Warn("failed to save projects cache", slog.String("error", err.Error()))
		}
	}

	return createIterator(projects), nil
}

func (api *API) loadFromCache(sess *session.Context, withSubprojects, all bool) ([]project.DirInfo, bool) {
	cacheFileName := fmt.Sprintf("projects-list-%t-%t.json", withSubprojects, all)
	cacheFilePath := filepath.Join(sess.Get("app.fs.path.profile.cache").String(), cacheFileName)

	// Check if cache file exists
	if _, err := os.Stat(cacheFilePath); err != nil {
		return nil, false
	}

	// Read and unmarshal cache
	data, err := os.ReadFile(cacheFilePath)
	if err != nil {
		return nil, false
	}

	var cache cacheList
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, false
	}

	// Check if cache is still valid
	lifetime := sess.Get("devel.projects.cache_list_lifetime").Duration()
	if time.Since(cache.CreatedAt) > lifetime {
		return nil, false
	}

	projects := api.ensureWorkingDirIncluded(sess, cache.Prjs)
	return projects, true
}

func (api *API) ensureWorkingDirIncluded(sess *session.Context, projects []project.DirInfo) []project.DirInfo {
	wd := sess.Get("app.fs.path.wd").String()

	// Check if working directory is already in the list
	for _, prj := range projects {
		if prj.Path == wd {
			return projects
		}
	}

	// Try to detect project in working directory
	info, found, err := project.Detect(wd)
	if err != nil {
		sess.Log().Warn("failed to detect project in working directory",
			slog.String("error", err.Error()),
			slog.String("path", wd))
		return projects
	}

	if found {
		return append(projects, info)
	}

	return projects
}

func (api *API) generateFreshProjectList(sess *session.Context, withSubprojects, all bool) ([]project.DirInfo, error) {
	// Project search paths or patterns
	search := sess.Get("devel.projects.search_paths").Fields()
	// Project search paths or patterns to ignore
	ignore := sess.Get("devel.projects.search_path_ignore").Fields()
	// Current working directory
	wd := sess.Get("app.fs.path.wd").String()

	searchPaths, searchWD := resolveSearchPaths(search, ignore, wd)
	if searchWD {
		sess.Log().Log(sess.Context(), logging.LevelNotImpl.Level(), "should add wd to saved search paths")
	}

	api.mu.RLock()
	defer api.mu.RUnlock()

	projects := api.listProjects(sess, searchPaths, ignore, withSubprojects, all)
	projects = api.ensureWorkingDirIncluded(sess, projects)

	return projects, nil
}

func (api *API) saveToCache(sess *session.Context, projects []project.DirInfo, withSubprojects, all bool) error {
	cacheFileName := fmt.Sprintf("projects-list-%t-%t.json", withSubprojects, all)
	cacheFilePath := filepath.Join(sess.Get("app.fs.path.profile.cache").String(), cacheFileName)

	cache := cacheList{
		CreatedAt: time.Now(),
		Prjs:      make([]project.DirInfo, len(projects)),
	}
	copy(cache.Prjs, projects)

	data, err := json.Marshal(&cache)
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFilePath, data, 0640)
}

func (api *API) listProjects(sess *session.Context, searchPaths, ignore []string, withSubprojects bool, all bool) []project.DirInfo {

	ignorem := gitutils.NewIgnoreMatcher(ignore, nil)

	var prjs []project.DirInfo

	for _, searchPath := range searchPaths {
		err := filepath.WalkDir(searchPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return filepath.SkipDir
			}

			if !d.IsDir() {
				return nil
			}

			pathParts := strings.Split(path, string(filepath.Separator))
			if ignorem.Match(pathParts, true) {
				return filepath.SkipDir
			}

			info, found, err := project.Detect(path)
			if err != nil {
				return err
			}

			if !found {
				return nil
			}

			if !all && found && (!info.HasConfigFile && !info.DependsOnHappy) && info.HasGit {
				return filepath.SkipDir
			}

			prjs = append(prjs, info)

			if !withSubprojects {
				return filepath.SkipDir
			}
			return nil
		})
		if err != nil {
			sess.Log().Error(err.Error())
			continue
		}
	}

	return prjs
}

func resolveSearchPaths(search, ignore []string, wd string) (result []string, addWD bool) {
	seen := make(map[string]struct{})
	ignorem := gitutils.NewIgnoreMatcher(ignore, nil)

	// Collect only directories and apply ignore patterns immediately
	for _, pattern := range search {
		for _, pat := range bexp.Parse(pattern) {
			matches, err := filepath.Glob(pat)
			if err != nil {
				continue
			}

			for _, m := range matches {
				// Single stat call per match, filter directories immediately
				if info, err := os.Stat(m); err == nil && info.IsDir() {
					pathParts := strings.Split(m, string(filepath.Separator))
					if !ignorem.Match(pathParts, true) {
						seen[m] = struct{}{}
					}
				}
			}
		}
	}

	for p := range seen {
		result = append(result, p)
	}

	// Ensure wd fallback
	addWD = true
	for _, rp := range result {
		if strings.HasPrefix(wd, rp) {
			addWD = false
			break
		}
	}
	if addWD {
		result = append(result, wd)
	}

	sort.Strings(result)
	return
}

func createIterator[T any](slice []T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, item := range slice {
			if !yield(item) {
				break
			}
		}
	}
}
