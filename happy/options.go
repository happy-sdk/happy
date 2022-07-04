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

package happy

import (
	"fmt"

	"github.com/mkungla/happy/config"
)

func Title(title string) Option {
	return func(opts OptionSetter) error {
		if cnf, ok := opts.(*config.Config); ok {
			cnf.Title = title
			return nil
		}
		return opts.Set("app.title", title)
	}
}

func Slug(slug string) Option {
	return func(opts OptionSetter) error {
		if !config.ValidSlug(slug) {
			return fmt.Errorf("%w: %s", config.ErrInvalidSlug, slug)
		}
		if cnf, ok := opts.(*config.Config); ok {
			cnf.Slug = slug
			return nil
		}
		return opts.Set("app.slug", slug)
	}
}

func Description(desc string) Option {
	return func(opts OptionSetter) error {
		if cnf, ok := opts.(*config.Config); ok {
			cnf.Description = desc
			return nil
		}
		return opts.Set("app.description", desc)
	}
}

func CopyrightBy(copyrightBy string) Option {
	return func(opts OptionSetter) error {
		if cnf, ok := opts.(*config.Config); ok {
			cnf.CopyrightBy = copyrightBy
			return nil
		}
		return opts.Set("app.copyright.by", copyrightBy)
	}
}

func CopyrightSince(year int) Option {
	return func(opts OptionSetter) error {
		if cnf, ok := opts.(*config.Config); ok {
			cnf.CopyrightSince = year
			return nil
		}
		return opts.Set("app.copyright.since", year)
	}
}

func License(license string) Option {
	return func(opts OptionSetter) error {
		if cnf, ok := opts.(*config.Config); ok {
			cnf.License = license
			return nil
		}
		return opts.Set("app.license", license)
	}
}

func WithLogger(logger Logger) Option {
	return func(opts OptionSetter) error {
		if app, ok := opts.(Application); ok {
			return app.Set("logger", logger)
		}
		return opts.Set("logger", logger)
	}
}
