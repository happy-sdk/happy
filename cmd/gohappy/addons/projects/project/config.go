// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package project

import (
	"fmt"
	"path/filepath"

	"github.com/happy-sdk/happy/cmd/gohappy/pkg/git"
	"github.com/happy-sdk/happy/pkg/fsutils"
	"github.com/happy-sdk/happy/pkg/options"
)

func newConfig() (config *options.Options, err error) {
	spec, err := options.New("project")
	if err != nil {
		return nil, err
	}

	var sections = []func(*options.Spec) error{
		addConfigSpecsLocal,
		addConfigSpecsGit,
	}

	for _, section := range sections {
		if err = section(spec); err != nil {
			return nil, err
		}
	}

	if config, err = spec.Seal(); err != nil {
		return nil, err
	}
	return config, err
}

func addConfigSpecsLocal(config *options.Spec) error {
	localConfig, err := options.New("local",
		options.NewOption("wd", ".").
			Description("Local working directory").
			Parser(func(opt options.Option, newval options.Value) (parsed options.Value, err error) {
				currentPath, err := filepath.Abs(newval.String())
				if err != nil {
					return opt.Default(), err
				}
				return options.NewValue(currentPath)
			}).
			Validator(func(opt options.Option) error {
				if !fsutils.IsDir(opt.String()) {
					return fmt.Errorf("%w(%s): %s is not a directory", Error, opt.Key(), opt.String())
				}
				return nil
			}),
	)
	if err != nil {
		return err
	}
	goConfig, err := options.New("go",
		options.NewOption("module.count", 0).
			Description("Number of go modules").
			Validator(func(opt options.Option) error {
				if count := opt.Variable().Int(); count < 0 {
					return fmt.Errorf("%w(%s): %d is not a positive integer", Error, opt.Key(), count)
				}
				return nil
			}),
		options.NewOption("monorepo", false),
	)

	if err != nil {
		return err
	}

	envConfig, err := options.New("env")
	if err != nil {
		return err
	}
	envConfig.AllowWildcard()
	return config.Extend(
		localConfig,
		goConfig,
		envConfig,
	)
}

func addConfigSpecsGit(config *options.Spec) error {
	gitopts, err := git.NewConfig()
	if err != nil {
		return err
	}
	return config.Extend(gitopts)
}
