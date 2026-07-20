// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package gomodule

import (
	"fmt"
	"strconv"
	"strings"
)

// BumpKind selects which version component a breaking change or Go version
// sync bumps - the release policy equivalent of "is this treated like a
// semver-major event, or does it stay within minor versions".
type BumpKind string

const (
	BumpKindMajor BumpKind = "major"
	BumpKindMinor BumpKind = "minor"
)

// ParseBumpKind validates s as a BumpKind.
func ParseBumpKind(s string) (BumpKind, error) {
	switch k := BumpKind(s); k {
	case BumpKindMajor, BumpKindMinor:
		return k, nil
	default:
		return "", fmt.Errorf("invalid bump kind %q: must be %q or %q", s, BumpKindMajor, BumpKindMinor)
	}
}

// BumpStrategy selects how far a BumpKind bump advances that component.
type BumpStrategy string

const (
	// BumpStrategySingle advances the component by 1, e.g. 55 -> 56.
	BumpStrategySingle BumpStrategy = "single"
	// BumpStrategyHundred advances the component to the next full multiple
	// of 100 strictly greater than its current value, e.g. 55 -> 100,
	// 100 -> 200.
	BumpStrategyHundred BumpStrategy = "hundred"
	// BumpStrategyThousand advances the component to the next full
	// multiple of 1000 strictly greater than its current value, e.g.
	// 55 -> 1000, 1000 -> 2000.
	BumpStrategyThousand BumpStrategy = "thousand"
)

// ParseBumpStrategy validates s as a BumpStrategy.
func ParseBumpStrategy(s string) (BumpStrategy, error) {
	switch st := BumpStrategy(s); st {
	case BumpStrategySingle, BumpStrategyHundred, BumpStrategyThousand:
		return st, nil
	default:
		return "", fmt.Errorf("invalid bump strategy %q: must be %q, %q, or %q", s, BumpStrategySingle, BumpStrategyHundred, BumpStrategyThousand)
	}
}

// nextBumpValue advances current per strategy. Always strictly greater than
// current, even when current already sits exactly on a "hundred"/"thousand"
// boundary, since a bump must always produce a new, higher version.
func nextBumpValue(current int, strategy BumpStrategy) int {
	switch strategy {
	case BumpStrategyHundred:
		return (current/100 + 1) * 100
	case BumpStrategyThousand:
		return (current/1000 + 1) * 1000
	default:
		return current + 1
	}
}

// bumpSignificant bumps tag's major or minor component (per kind), advancing
// it per strategy - the configurable equivalent of bumpMajor/bumpMinor used
// for breaking changes and Go version syncs, where a project may prefer to
// jump straight to the next full hundred/thousand minor instead of just
// incrementing by one (see the compatibility-block release policy).
func bumpSignificant(prefix, tag string, kind BumpKind, strategy BumpStrategy) (string, error) {
	clean := strings.TrimPrefix(tag, prefix+"v")
	parts := strings.Split(clean, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid version: %s", tag)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", err
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", err
	}

	if kind == BumpKindMinor {
		return fmt.Sprintf("%sv%d.%d.0", prefix, major, nextBumpValue(minor, strategy)), nil
	}
	return fmt.Sprintf("%sv%d.0.0", prefix, nextBumpValue(major, strategy)), nil
}
