// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package gomodule

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"

	"github.com/happy-sdk/happy/pkg/version"
)

func Count(wd string) (int, error) {
	totalmodules := 0
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
		totalmodules++
		return nil
	}); err != nil {
		return 0, err
	}
	return totalmodules, nil
}

// TopologicalReleaseQueue performs a topological sort on the dependency graph
// and returns a slice of package names in release order while also
// modifying the pkgs slice.
func TopologicalReleaseQueue(pkgs []*Package) ([]string, error) {
	pkgMap := make(map[string]*Package)
	for i := range pkgs {
		pkgMap[pkgs[i].Import] = pkgs[i]
	}

	// Build dependency graph
	depGraph := make(map[string][]string)

	// Initialize all packages in the graph
	for _, pkg := range pkgs {
		depGraph[pkg.Import] = []string{}
	}

	for _, pkg := range pkgs {
		for _, require := range pkg.Modfile.Require {
			if dep, exists := pkgMap[require.Mod.Path]; exists {
				// Add edge from dependency to dependent (dep -> pkg)
				depGraph[dep.Import] = append(depGraph[dep.Import], pkg.Import)
				basever := path.Base(dep.NextReleaseTag)
				if basever == "" || basever == "." {
					basever = path.Base(dep.LastReleaseTag)
				}
				depNextRelease, err := version.Parse(basever)
				if err != nil {
					return nil, err
				}
				if err := pkg.SetDep(dep.Import, depNextRelease); err != nil {
					return nil, err
				}
			}
		}
	}

	// Topological Sort with detailed debugging
	var queue []string
	visited := make(map[string]bool)
	visiting := make(map[string]bool)

	var visit func(string, []string) error
	visit = func(n string, path []string) error {
		// Add current node to path for cycle detection
		currentPath := append(path, n)

		if visiting[n] {
			fmt.Printf("CYCLE DETECTED: %v -> %s\n", currentPath, n)
			// Find where the cycle starts
			for i, p := range currentPath {
				if p == n {
					fmt.Printf("Cycle: %v\n", currentPath[i:])
					break
				}
			}
			return fmt.Errorf("circular dependency detected: %v", currentPath)
		}
		if visited[n] {
			return nil
		}

		visiting[n] = true
		for _, m := range depGraph[n] {
			if err := visit(m, currentPath); err != nil {
				return err
			}
		}
		visiting[n] = false
		visited[n] = true
		queue = append(queue, n)
		return nil
	}

	// Visit all packages
	for _, pkg := range pkgs {
		if !visited[pkg.Import] {
			if err := visit(pkg.Import, []string{}); err != nil {
				return nil, fmt.Errorf("dependency resolution error: %v", err)
			}
		}
	}
	slices.Reverse(queue)

	// Create index map for sorting
	importToIndex := make(map[string]int, len(queue))
	for i, impr := range queue {
		importToIndex[impr] = i
	}

	// Sort pkgs according to topological order
	slices.SortFunc(pkgs, func(a, b *Package) int {
		return importToIndex[a.Import] - importToIndex[b.Import]
	})

	return queue, nil
}

func GetCommonDeps(pkgs []*Package) ([]Dependency, error) {
	// Map to hold the count of each external dependency
	deps := make(map[string]Dependency)

	// Map to quickly check if a dependency is internal
	internalDeps := make(map[string]struct{})
	for _, pkg := range pkgs {
		internalDeps[pkg.Import] = struct{}{}
	}

	// Collect and count external dependencies
	for _, pkg := range pkgs {
		for _, require := range pkg.Modfile.Require {
			if _, internal := internalDeps[require.Mod.Path]; !internal {
				requireModVersion, err := version.Parse(require.Mod.Version)
				if err != nil {
					return nil, err
				}

				if dep, exists := deps[require.Mod.Path]; !exists {
					deps[require.Mod.Path] = Dependency{
						Import:     require.Mod.Path,
						UsedBy:     []string{pkg.Import},
						MaxVersion: requireModVersion,
						MinVersion: requireModVersion,
					}
				} else {
					if version.Compare(requireModVersion, dep.MaxVersion) == 1 {
						dep.MaxVersion = requireModVersion
					}
					if version.Compare(requireModVersion, dep.MinVersion) == -1 {
						dep.MinVersion = requireModVersion
					}
					dep.UsedBy = append(dep.UsedBy, pkg.Import)
					deps[require.Mod.Path] = dep
				}
			}
		}
	}

	// Filter dependencies that are referenced by at least two packages
	var commonDeps []Dependency
	for _, dep := range deps {
		if len(dep.UsedBy) >= 2 {
			commonDeps = append(commonDeps, dep)
		}
	}
	return commonDeps, nil
}
