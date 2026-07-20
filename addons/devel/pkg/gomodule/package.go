// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package gomodule

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/happy-sdk/happy/lib/changelog"
	"github.com/happy-sdk/happy/lib/scm/gitutils"
	tr "github.com/happy-sdk/happy/lib/taskrunner"
	"github.com/happy-sdk/happy/pkg/version"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/session"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

type Package struct {
	ModFilePath                string
	Dir                        string
	TagPrefix                  string
	Import                     string
	Modfile                    *modfile.File
	FirstRelease               bool
	NeedsRelease               bool
	PendingRelease             bool
	IsInternal                 bool
	UpdateDeps                 bool
	NextReleaseTag             string
	NextReleaseTagRemoteExists bool
	LastReleaseTag             string
	Changelog                  *changelog.Release
}

func Load(sess *session.Context, root, path string) (pkg *Package, err error) {
	if path == "" {
		return nil, errors.New("can not load module, path is empty")
	}

	pkg = &Package{}

	if filepath.Base(path) == "go.mod" {
		pkg.ModFilePath = path
		pkg.Dir = filepath.Dir(path)
	} else {
		pkg.Dir, err = filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		pkg.ModFilePath = filepath.Join(path, "go.mod")
	}

	if len(pkg.Dir) < 5 {
		return nil, fmt.Errorf("invalid module directory %s", pkg.Dir)
	}

	dirstat, err := os.Stat(pkg.Dir)
	if err != nil {
		return nil, err
	}
	if !dirstat.IsDir() {
		return nil, fmt.Errorf("invalid module directory %s", pkg.Dir)
	}

	modstat, err := os.Stat(pkg.ModFilePath)
	if err != nil {
		return nil, err
	}
	if modstat.IsDir() {
		return nil, fmt.Errorf("invalid module go.mod path %s", pkg.ModFilePath)
	}

	data, err := os.ReadFile(pkg.ModFilePath)
	if err != nil {
		return nil, err
	}

	pkg.Modfile, err = modfile.Parse("go.mod", data, nil)
	if err != nil {
		return nil, err
	}
	pkg.Import = pkg.Modfile.Module.Mod.Path

	pkg.TagPrefix = strings.TrimPrefix(pkg.Dir+"/", root+"/")

	return pkg, nil
}

func LoadAll(sess *session.Context, wd string) ([]*Package, error) {
	var pkgs []*Package

	if err := filepath.Walk(wd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		goModPath := filepath.Join(path, "go.mod")
		if _, err := os.Stat(goModPath); err != nil {
			return nil
		}

		pkg, err := Load(sess, wd, goModPath)
		if err != nil {
			return err
		}
		pkgs = append(pkgs, pkg)
		return nil
	}); err != nil {
		return nil, err
	}
	return pkgs, nil
}

func (p *Package) SetDep(dep string, ver version.Version) error {
	if p.IsInternal {
		return nil
	}
	for _, require := range p.Modfile.Require {
		if require.Mod.Path == dep {
			requireModVersion, err := version.Parse(require.Mod.Version)
			if err != nil {
				return err
			}
			if version.Compare(ver, requireModVersion) <= 0 {
				return nil
			}
			break
		}
	}

	if err := p.Modfile.AddRequire(dep, ver.String()); err != nil {
		return err
	}
	p.NeedsRelease = true
	p.UpdateDeps = true

	if p.NextReleaseTag == "" || p.LastReleaseTag == p.NextReleaseTag {
		nextver, err := bumpPatch(p.TagPrefix, p.LastReleaseTag)
		if err != nil {
			return fmt.Errorf("failed to bump patch version for(%s): %w", p.Import, err)
		}
		p.NextReleaseTag = nextver
	}

	p.Modfile.Cleanup()
	return nil
}

// SyncGoVersion updates the module's go directive to match the root
// module's go version, so introducing a new Go-version-gated feature at
// the root (e.g. bumping to 1.27 for new generic methods) rolls out to
// every module in the monorepo on the next release, not just the ones
// whose own code happened to change. Must be called after LoadReleaseInfo,
// since it relies on LastReleaseTag and NextReleaseTag being populated.
//
// A Go version sync is treated the same as a breaking change release-wise:
// it bumps bumpKind (advanced per bumpStrategy) rather than just a patch,
// since raising the minimum required Go version is itself the kind of
// significant, compatibility-affecting event bumpKind/bumpStrategy exist to
// express. That significant bump always wins over whatever ordinary
// (non-breaking) minor/patch bump getChangelog may have already computed
// from the package's own commits, since it's frequently the larger of the
// two - e.g. a plain feat commit's ordinary +1 minor bump must not stand in
// for a "jump straight to the next full hundred" Go version sync. The two
// candidates are compared explicitly (not just "is anything already set")
// so the actually-larger bump always wins, regardless of which one ran
// first.
func (p *Package) SyncGoVersion(rootGoVersion string, bumpKind BumpKind, bumpStrategy BumpStrategy) error {
	if p.IsInternal || rootGoVersion == "" {
		return nil
	}
	if p.Modfile.Go != nil && p.Modfile.Go.Version == rootGoVersion {
		return nil
	}

	if err := p.Modfile.AddGoStmt(rootGoVersion); err != nil {
		return fmt.Errorf("failed to sync go version for(%s): %w", p.Import, err)
	}
	p.NeedsRelease = true

	candidate, err := bumpSignificant(p.TagPrefix, p.LastReleaseTag, bumpKind, bumpStrategy)
	if err != nil {
		return fmt.Errorf("failed to bump version for(%s): %w", p.Import, err)
	}

	if p.NextReleaseTag == "" || p.LastReleaseTag == p.NextReleaseTag {
		p.NextReleaseTag = candidate
	} else {
		candidateVersion := version.Version(path.Base(candidate))
		pendingVersion := version.Version(path.Base(p.NextReleaseTag))
		if candidateVersion.IsValid() && pendingVersion.IsValid() && version.Compare(candidateVersion, pendingVersion) == 1 {
			p.NextReleaseTag = candidate
		}
	}

	p.Modfile.Cleanup()
	return nil
}

func (p *Package) LoadReleaseInfo(sess *session.Context, rootPath, remoteName string, checkRemote bool, bumpKind BumpKind, bumpStrategy BumpStrategy) error {
	sess.Log().Debug(
		"getting latest release",
		slog.String("package", p.Modfile.Module.Mod.Path),
		slog.String("tag.prefix", p.TagPrefix),
	)

	tagscmd := exec.Command("git", "tag", "--list", p.TagPrefix+"*")
	tagscmd.Dir = rootPath
	tagsout, err := cli.Exec(sess, tagscmd)
	if err != nil {
		return err
	}

	var nextVersion version.Version = "v0.1.0"
	nextVersionFile := filepath.Join(p.Dir, "VERSION")
	nextVersionBytes, err := os.ReadFile(nextVersionFile)
	if err == nil {
		nextVersionStr := strings.TrimSpace(string(nextVersionBytes))
		nextVersion, err = version.Parse(nextVersionStr)
		if err != nil {
			nextVersion = "v0.1.0"
		}
	}

	defer func() {
		if !nextVersion.IsValid() {
			return
		}
		nextReleaseTagVersion := version.Version(path.Base(p.NextReleaseTag))
		lastReleaseTagVersion := version.Version(path.Base(p.LastReleaseTag))

		if lastReleaseTagVersion.IsValid() {
			if version.Compare(nextVersion, lastReleaseTagVersion) == 1 {
				p.NextReleaseTag = fmt.Sprintf("%s%s", p.TagPrefix, nextVersion)
				p.NeedsRelease = true
			}
		}
		if nextReleaseTagVersion.IsValid() && len(nextVersionBytes) > 0 {
			if version.Compare(nextVersion, nextReleaseTagVersion) == 1 {
				p.NextReleaseTag = fmt.Sprintf("%s%s", p.TagPrefix, nextVersion)
				p.NeedsRelease = true
			}
		}

		if p.NeedsRelease && p.Changelog == nil {
			p.Changelog = changelog.NewRelease()
			p.Changelog.Add("", "", "", "initial release", changelog.EntryType{
				Typ:  "feat",
				Kind: changelog.EntryKindPatch,
			})
		}
	}()

	defer func() {
		if err := p.addMissing(sess); err != nil {
			sess.Log().Error(err.Error())
		}

	}()

	if tagsout == "" {
		// First release
		p.FirstRelease = true
		p.NeedsRelease = true
		p.NextReleaseTag = fmt.Sprintf("%s%s", p.TagPrefix, nextVersion)
		p.LastReleaseTag = fmt.Sprintf("%s%s", p.TagPrefix, "v0.0.0")
		if strings.Contains(p.Import, "internal") {
			p.FirstRelease = false
			p.NeedsRelease = false
			p.IsInternal = true
			p.LastReleaseTag = "."
			p.NextReleaseTag = "."
		}
		return nil
	}

	fulltags := strings.Split(tagsout, "\n")
	var tags []string
	for _, tag := range fulltags {

		ntag := strings.TrimPrefix(tag, p.TagPrefix)
		// skip nested package
		if strings.Contains(ntag, "/") {
			continue
		}
		tags = append(tags, ntag)
	}
	semver.Sort(tags)
	p.LastReleaseTag = fmt.Sprintf("%s%s", p.TagPrefix, tags[len(tags)-1])

	// Handle pending release
	if !checkRemote {
		return p.getChangelog(sess, rootPath, bumpKind, bumpStrategy)
	}

	if gitutils.RemoteTagExists(sess, rootPath, remoteName, p.LastReleaseTag) {
		p.NextReleaseTagRemoteExists = true
		return p.getChangelog(sess, rootPath, bumpKind, bumpStrategy)
	}

	p.NextReleaseTag = p.LastReleaseTag
	p.LastReleaseTag = ""
	p.NeedsRelease = true
	p.PendingRelease = true

	tags = tags[:len(tags)-1]
	for i := len(tags) - 1; i >= 0; i-- {
		tag := fmt.Sprintf("%s%s", p.TagPrefix, tags[i])
		if gitutils.RemoteTagExists(sess, rootPath, remoteName, tag) {
			p.LastReleaseTag = tag
			break
		}
	}
	if p.LastReleaseTag == "" {
		p.LastReleaseTag = fmt.Sprintf("%s%s", p.TagPrefix, "v0.0.0")
		p.FirstRelease = true
	}

	return p.getChangelog(sess, rootPath, bumpKind, bumpStrategy)
}

func (p *Package) ApplyTagTask(sess *session.Context, r *tr.Executor, dep tr.TaskID, prjwd string, internalDeps []*Package) tr.TaskID {
	var (
		failed bool
		name   = path.Base(p.Dir)
	)

	t1 := r.SubtaskD(dep, fmt.Sprintf("%s: check need release", name), func(ex *tr.Executor) (res tr.Result) {
		if !p.NeedsRelease {
			return tr.Skip(p.LastReleaseTag).WithDesc(p.Import)
		} else if p.PendingRelease {
			return tr.Skip(fmt.Sprintf("pending release %s -> %s", path.Base(p.LastReleaseTag), path.Base(p.NextReleaseTag))).WithDesc(p.Import)
		}
		msg := fmt.Sprintf("%s%s -> %s",
			p.TagPrefix,
			path.Base(p.LastReleaseTag),
			path.Base(p.NextReleaseTag),
		)
		if p.FirstRelease {
			msg = fmt.Sprintf("%s%s",
				p.TagPrefix,
				path.Base(p.NextReleaseTag),
			)
		}
		return tr.Success(msg).WithDesc(p.Import)
	})

	var monorepoDeps []string

	t2 := r.SubtaskD(t1, fmt.Sprintf("%s: verify deps", name),
		func(ex *tr.Executor) tr.Result {
			for _, require := range p.Modfile.Require {
				var dep *Package
				if !slices.ContainsFunc(internalDeps, func(p *Package) bool {
					if p.Import == require.Mod.Path {
						dep = p
						return true
					}
					return false
				}) {
					continue
				}
				if !dep.NeedsRelease {
					continue
				}
				if !gitutils.TagExists(sess, prjwd, dep.NextReleaseTag) {
					failed = true
					return tr.Failure(fmt.Sprintf("tag %s does not exist", dep.NextReleaseTag))
				}
				monorepoDeps = append(monorepoDeps, dep.Import)
				if err := p.Modfile.AddReplace(dep.Import, "", dep.Dir, ""); err != nil {
					failed = true
					return tr.Failure("add tmp replace").WithDesc(err.Error())
				}
			}
			return tr.Success("ok")
		})

	t3 := r.SubtaskD(t2, fmt.Sprintf("%s: update go.mod", name),
		func(ex *tr.Executor) tr.Result {
			if p.PendingRelease {
				return tr.Success(fmt.Sprintf("pending release %s -> %s", path.Base(p.LastReleaseTag), path.Base(p.NextReleaseTag))).WithDesc(p.Import)
			}
			p.Modfile.Cleanup()
			updatedModFile, err := p.Modfile.Format()
			if err != nil {
				failed = true
				return tr.Failure("format go.mod").WithDesc(err.Error())
			}
			if err := os.WriteFile(p.ModFilePath, updatedModFile, 0644); err != nil {
				failed = true
				return tr.Failure("write go.mod").WithDesc(err.Error())
			}
			if err := p.GoModTidy(sess); err != nil {
				failed = true
				return tr.Failure("tidy go.mod").WithDesc(err.Error())
			}
			return tr.Success("go.mod updated")
		})

	t4 := r.SubtaskD(t3, fmt.Sprintf("%s: write go.mod", name),
		func(ex *tr.Executor) tr.Result {
			if len(monorepoDeps) > 0 {
				for _, rep := range p.Modfile.Replace {
					if !slices.ContainsFunc(internalDeps, func(p *Package) bool {
						return p.Import == rep.Old.Path
					}) {
						continue
					}
					if err := p.Modfile.DropReplace(rep.Old.Path, rep.Old.Version); err != nil {
						failed = true
						return tr.Failure("drop replace").WithDesc(err.Error())
					}
				}
			}

			p.Modfile.Cleanup()
			updatedModFile, err := p.Modfile.Format()
			if err != nil {
				failed = true
				return tr.Failure("format go.mod after drop replace").WithDesc(err.Error())
			}

			if err := os.WriteFile(p.ModFilePath, updatedModFile, 0644); err != nil {
				return tr.Failure("write go.mod after drop replace").WithDesc(err.Error())
			}
			if err := p.GoModTidy(sess); err != nil {
				failed = true
				return tr.Failure("tidy go.mod").WithDesc(err.Error())
			}
			return tr.Success("go.mod updated")
		})

	_ = r.SubtaskD(t4, fmt.Sprintf("%s: commit", name),
		func(ex *tr.Executor) tr.Result {
			if !gitutils.Dirty(sess, prjwd, p.Dir) {
				return tr.Skip("git path clean")
			}

			msg := fmt.Sprintf("chore(%s): :label: prepare release %s", name, path.Base(p.NextReleaseTag))
			if err := gitutils.Commit(sess, prjwd, []string{p.Dir}, msg); err != nil {
				failed = true
				return tr.Failure("commit").WithDesc(err.Error())
			}
			return tr.Success("changes committed")
		})

	_ = r.SubtaskD(dep, fmt.Sprintf("%s: tag", name),
		func(ex *tr.Executor) tr.Result {
			if !p.NeedsRelease {
				return tr.Skip("no tag needed").WithDesc(fmt.Sprintf("latest tag: %s", p.LastReleaseTag))
			} else if failed {
				return tr.Skip("deps failed")
			} else if p.PendingRelease {
				return tr.Skip("tag already exists").WithDesc(fmt.Sprintf("tag: %s", p.NextReleaseTag))
			}

			if err := gitutils.Tag(sess, prjwd, p.NextReleaseTag, path.Base(p.NextReleaseTag)); err != nil {
				failed = true
				return tr.Failure("tag").WithDesc(err.Error())
			}
			pushcmd := exec.Command("git", "push")
			pushcmd.Dir = prjwd
			if err := pushcmd.Run(); err != nil {
				failed = true
				return tr.Failure("push commits").WithDesc(err.Error())
			}
			tagpushcmd := exec.Command("git", "push", "--tags")
			tagpushcmd.Dir = prjwd
			if err := tagpushcmd.Run(); err != nil {
				failed = true
				return tr.Failure("push tags").WithDesc(err.Error())
			}
			return tr.Success(fmt.Sprintf("tag %s created", p.NextReleaseTag))
		})

	tFinal := r.SubtaskD(dep, fmt.Sprintf("%s: passed", name), func(ex *tr.Executor) (res tr.Result) {
		if failed {
			return tr.Failure("previous task did not pass")
		}
		if p.NeedsRelease {
			return tr.Success("ok")
		}

		return tr.Info("skip")
	})
	return tFinal
}
func (p *Package) GoModTidy(sess *session.Context) error {
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = p.Dir
	_, err := cli.ExecRaw(sess, tidyCmd)
	return err
}

func (p *Package) getChangelog(sess *session.Context, rootPath string, bumpKind BumpKind, bumpStrategy BumpStrategy) error {
	if p.IsInternal {
		return nil
	}
	var lastTagQuery = []string{"log"}
	upto := "HEAD"
	if p.PendingRelease {
		upto = p.NextReleaseTag
	}
	if !p.FirstRelease {
		lastTagQuery = append(lastTagQuery, fmt.Sprintf("%s..%s", p.LastReleaseTag, upto))
	}

	localpath := strings.TrimSuffix(p.TagPrefix, "/")
	if len(localpath) == 0 {
		localpath = "."
	}
	lastTagQuery = append(lastTagQuery, []string{"--pretty=format::COMMIT_START:%nSHORT:%h%nLONG:%H%nAUTHOR:%an%nMESSAGE:%B:COMMIT_END:", "--", localpath}...)

	// Add exclusions by walking the directory tree
	// filepath.Join(rootPath, localpath) and exclude all dirs which have go.mod
	exclusions, err := buildExclusions(rootPath, localpath)
	if err == nil {
		lastTagQuery = append(lastTagQuery, exclusions...)
	}

	logcmd := exec.Command("git", lastTagQuery...)
	logcmd.Dir = rootPath
	logout, err := cli.Exec(sess, logcmd)
	if err != nil {
		return err
	}
	changelog, err := changelog.ParseGitLog(logout)
	if err != nil {
		return err
	}

	p.Changelog = changelog
	if p.Changelog.Empty() {
		sess.Log().Debug("no changelog", slog.String("package", p.Import))
		return nil
	}
	if p.Changelog.HasMajorUpdate() {
		nextTag, err := bumpSignificant(p.TagPrefix, p.LastReleaseTag, bumpKind, bumpStrategy)
		if err != nil {
			return fmt.Errorf("failed to bump version for(%s): %w", p.Import, err)
		}
		p.NextReleaseTag = nextTag
		p.NeedsRelease = true
	} else if p.Changelog.HasMinorUpdate() {
		nextTag, err := bumpMinor(p.TagPrefix, p.LastReleaseTag)
		if err != nil {
			return fmt.Errorf("failed to bump minor version for(%s): %w", p.Import, err)
		}
		p.NextReleaseTag = nextTag
		p.NeedsRelease = true
	} else if p.Changelog.HasPatchUpdate() {
		nextTag, err := bumpPatch(p.TagPrefix, p.LastReleaseTag)
		if err != nil {
			return fmt.Errorf("failed to bump patch version for(%s): %w", p.Import, err)
		}
		p.NextReleaseTag = nextTag
		p.NeedsRelease = true
	}
	return nil
}

// buildExclusions finds directories with go.mod files or tags and returns exclusion patterns
func buildExclusions(rootPath, localpath string) ([]string, error) {
	var exclusions []string

	// Full path to search
	searchPath := filepath.Join(rootPath, localpath)

	// Get all tagged paths for exclusion
	taggedPaths, _ := getTaggedPaths(rootPath, localpath)

	// Walk the directory tree starting from searchPath
	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if !info.IsDir() {
			return nil
		}

		// Convert to relative path from rootPath
		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return nil
		}

		// Skip if this is the same as our localpath (don't exclude ourselves)
		if relPath == localpath || relPath == "." {
			return nil
		}

		// Skip if this directory is not nested under our localpath
		if !strings.HasPrefix(relPath, localpath+string(filepath.Separator)) {
			return nil
		}

		shouldExclude := false

		// Check if this directory has a go.mod file
		goModPath := filepath.Join(path, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			shouldExclude = true
		}

		// Check if this directory has tags
		if !shouldExclude {
			shouldExclude = slices.Contains(taggedPaths, relPath)
		}

		if shouldExclude {
			// Add exclusion pattern for this directory
			exclusions = append(exclusions, ":!"+relPath+"/*")
			// Skip walking into this directory since we're excluding it
			return filepath.SkipDir
		}

		return nil
	})

	return exclusions, err
}

// getTaggedPaths returns all directory paths that have at least one tag
func getTaggedPaths(rootPath, localpath string) ([]string, error) {
	var taggedPaths []string

	// Get all tags from git
	cmd := exec.Command("git", "tag", "-l")
	cmd.Dir = rootPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// Parse tags and extract paths
	pathMap := make(map[string]bool)
	tags := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, tag := range tags {
		if tag == "" {
			continue
		}

		// Extract path from tag (assuming format like "path/to/module/v1.2.3")
		// Remove version suffix to get the path
		tagPath := extractPathFromTag(tag)
		if tagPath == "" {
			continue
		}

		// Only include paths that are nested under our localpath
		if strings.HasPrefix(tagPath, localpath+"/") && tagPath != localpath {
			pathMap[tagPath] = true
		}
	}

	// Convert map to slice
	for path := range pathMap {
		taggedPaths = append(taggedPaths, path)
	}

	return taggedPaths, nil
}

// extractPathFromTag extracts the directory path from a tag
// Assumes tags follow pattern like "path/to/module/v1.2.3" or "path/to/module/v1.2.3-beta.1"
func extractPathFromTag(tag string) string {
	// Find the last segment that looks like a version
	parts := strings.Split(tag, "/")
	if len(parts) < 2 {
		return ""
	}

	// Look for version pattern from the end
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		// Check if this part looks like a version (starts with v and has dots)
		if strings.HasPrefix(part, "v") && strings.Contains(part, ".") {
			// Everything before this part is the path
			if i == 0 {
				return "" // No path, just version
			}
			return strings.Join(parts[:i], "/")
		}
	}

	// If no version pattern found, assume the whole thing is a path
	return tag
}

func bumpMinor(prefix, tag string) (string, error) {

	clean := strings.TrimPrefix(tag, prefix+"v")
	parts := strings.Split(clean, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid version: %s", tag)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%s", prefix, fmt.Sprintf("v%s.%d.0", parts[0], minor+1)), nil
}

func bumpPatch(prefix, tag string) (string, error) {

	clean := strings.TrimPrefix(tag, prefix+"v")
	parts := strings.Split(clean, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid version: %s", tag)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%s", prefix, fmt.Sprintf("v%s.%s.%d", parts[0], parts[1], patch+1)), nil
}

// ModuleInfo represents the JSON output from go list -m -json
type ModuleInfo struct {
	Path      string `json:"Path"`
	Version   string `json:"Version"`
	Main      bool   `json:"Main"`
	Dir       string `json:"Dir,omitempty"`
	GoMod     string `json:"GoMod,omitempty"`
	GoVersion string `json:"GoVersion,omitempty"`
}

// AddMissing adds missing dependencies to the modfile
func (p *Package) addMissing(sess *session.Context) error {
	if p.IsInternal {
		return nil
	}
	// Get all dependencies with their module info in one command
	cmd := exec.Command("go", "list", "-deps", "-json", "./...")
	cmd.Dir = p.Dir
	output, err := cli.ExecRaw(sess, cmd)
	if err != nil {
		return fmt.Errorf("failed to get dependencies: %w", err)
	}

	// Parse JSON output to get module information
	moduleMap := make(map[string]string) // module path -> version
	decoder := jsontext.NewDecoder(strings.NewReader(string(output)))

	for {
		raw, err := decoder.ReadValue()
		if err != nil {
			break // end of stream or truncated trailing entry
		}

		var pkg struct {
			ImportPath string `json:"ImportPath"`
			Module     *struct {
				Path    string `json:"Path"`
				Version string `json:"Version"`
			} `json:"Module"`
		}

		if err := json.Unmarshal(raw, &pkg); err != nil {
			continue // Skip malformed entries
		}

		// Skip if no module info or if it's the main module
		if pkg.Module == nil || pkg.Module.Path == p.Import {
			continue
		}

		// Skip standard library (packages without dots in first segment)
		if isStandardLibrary(pkg.ImportPath) {
			continue
		}

		// Add to module map (automatically deduplicates)
		moduleMap[pkg.Module.Path] = pkg.Module.Version
	}

	// Add missing modules to modfile
	for modulePath, ver := range moduleMap {
		// Check if module is already in modfile
		if !isModuleInModfile(p.Modfile, modulePath) {
			// Handle empty version by getting latest
			if ver == "" {
				ver, _ = GetLatestVersion(modulePath, p.Dir)
			}
			if err := p.SetDep(modulePath, version.Version(ver)); err != nil {
				return err
			}
		}
	}

	return nil
}

func isStandardLibrary(pkg string) bool {
	if !strings.Contains(pkg, "/") {
		return true // Single element packages are stdlib
	}

	firstPart := strings.Split(pkg, "/")[0]
	// Standard library doesn't have dots in the first part
	return !strings.Contains(firstPart, ".")
}

// isModuleInModfile checks if a module is already present in the modfile
func isModuleInModfile(mf *modfile.File, modulePath string) bool {
	for _, req := range mf.Require {
		if req.Mod.Path == modulePath {
			return true
		}
	}
	return false
}
