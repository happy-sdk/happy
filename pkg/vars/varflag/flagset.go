// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package varflag

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/happy-sdk/happy/pkg/vars"
)

// FlagSet holds collection of flags for parsing
// e.g. global, sub command or custom flag collection.
type FlagSet struct {
	mu      sync.RWMutex
	name    string
	argn    int
	present bool
	flags   []Flag
	sets    []Flags
	args    []vars.Value
	pos     int
	parsed  bool
	aliases map[string]string
}

var ErrInvalidFlagSetName = errors.New("invalid flag set name")

// NewFlagSet is wrapper to parse flags together.
// e.g. under specific command. Where "name" is command name
// to search before parsing the flags under this set.
// argsn is number of command line arguments allowed within this set.
// If argsn is -gt 0 then parser will stop after finding argsn+1 argument
// which is not a flag.
func NewFlagSet(name string, argn int) (*FlagSet, error) {
	if name == "/" {
		name = filepath.Base(os.Args[0])
	} else if !ValidFlagName(name) {
		return nil, fmt.Errorf("%w: %q is not valid name for flag set", ErrInvalidFlagSetName, name)
	}
	return &FlagSet{
		name:    name,
		argn:    argn,
		aliases: make(map[string]string),
	}, nil
}

func SetArgcMax(flags Flags, max int) error {
	s, ok := flags.(*FlagSet)
	if !ok {
		return fmt.Errorf("%w: can not call SetArgcMax for %T", ErrFlag, flags)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.parsed {
		return fmt.Errorf("%w: %s", ErrFlagAlreadyParsed, s.name)
	}
	s.argn = max
	return nil
}

// Name returns name of the flag set.
func (s *FlagSet) Name() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.name
}

// Len returns nuber of flags in set.
func (s *FlagSet) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.flags)
}

// Add flag to flag set.
func (s *FlagSet) Add(flag ...Flag) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, f := range flag {
		_, err := s.get(f.Name())
		if !errors.Is(err, ErrNoNamedFlag) {
			return fmt.Errorf("%w: %s", ErrFlagExists, f.Name())
		}
		if err := s.checkAliasShadowing(f); err != nil {
			return err
		}
		f.AttachTo(s.name)
		for _, alias := range f.Aliases() {
			s.aliases[alias] = f.Name()
		}
	}

	s.flags = append(s.flags, flag...)
	return nil
}

// Get flag of current set.
func (s *FlagSet) Get(name string) (Flag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.get(name)
}
func (s *FlagSet) get(name string) (Flag, error) {
	for _, f := range s.flags {
		if f.Name() == name {
			return f, nil
		}
	}
	return nil, ErrNoNamedFlag
}

// AddSet adds flag set.
func (s *FlagSet) AddSet(set ...Flags) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sets = append(s.sets, set...)
	return nil
}

// GetSetTree
func (s *FlagSet) GetActiveSets() []Flags {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tree []Flags
	tree = append(tree, s)
	if s.Present() {
		for _, set := range s.sets {
			if set.Present() {
				tree = append(tree, set.GetActiveSets()...)
			}
		}
	}

	return tree
}

// Pos returns flagset position from it's parent.
func (s *FlagSet) Pos() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pos
}

// Present returns true if this set was parsed.
func (s *FlagSet) Present() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.present
}

// Args returns parsed arguments for this flag set.
func (s *FlagSet) Args() []vars.Value {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.args
}

// Args returns parsed arguments for this flag set.
func (s *FlagSet) AcceptsArgs() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.argn > 0
}

// Flags returns slice of flags in this set.
func (s *FlagSet) Flags() []Flag {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.flags
}

// Sets retruns subsets of flags under this flagset.
func (s *FlagSet) Sets() []Flags {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sets
}

// Parse all flags recursively.
func (s *FlagSet) Parse(args []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.parsed {
		return fmt.Errorf("%w: %s", ErrFlagAlreadyParsed, s.name)
	}

	var currargs []string
	if s.name != "*" && s.name != filepath.Base(os.Args[0]) {
		for i, arg := range args {
			if arg == s.name {
				s.pos = i
				currargs = args[i:]
				s.present = true
			}
		}
	} else {
		currargs = args
		// root cmd is always considered as present
		s.present = true
	}

	// if set is not present it is not an error
	if !s.present {
		return nil
	}

	// first parse flags for current set
	for _, gflag := range s.flags {
		_, err := gflag.Parse(currargs)
		if err != nil && !errors.Is(err, ErrFlagAlreadyParsed) {
			return fmt.Errorf("%s %w", s.name, err)
		}
		// this flag need to be removed from sub command args
		if gflag.Present() {
			currargs = slicediff(currargs, gflag.Input())
		}
	}

	// parse flags for sets
	for _, set := range s.sets {
		err := set.Parse(currargs)
		if err != nil {
			return err
		}
		if set.Present() {
			if s.name == filepath.Base(os.Args[0]) {
				// update global flag command names
				for _, flag := range s.flags {
					if !flag.Present() {
						continue
					}
				}
			}
			break
		}
	}

	// since we did not have errors we can look up args
	return s.extractArgs(currargs)
}

func (s *FlagSet) extractArgs(args []string) error {
	if len(args) == 0 {
		return nil
	}

	// rm subcmds
includessubset:
	for _, set := range s.sets {
		if set.Present() {
			for i, arg := range args {
				if arg == set.Name() {
					args = args[:i]
					if i == 0 {
						return nil
					}
					// there is no args for current set since
					// there is sub set which was first arg
					break includessubset
				}
			}
		}
	}

	// filter flags
	used := []string{}
	if args[0] == s.name || args[0] == os.Args[0] {
		used = append(used, args[0])
	}

	sargs := slicediff(args, used)

	if s.argn == 0 && len(sargs) > 0 {
		if strings.HasPrefix(sargs[0], "-") {
			return fmt.Errorf("%w: %s does not accept flag %s", ErrInvalidArguments, s.name, sargs[0])
		}

		return errors.Join(ErrInvalidArguments, ErrInvalidCommandOrArgs.WithArgs(s.name, sargs[0]))
	}

	for _, arg := range sargs {
		a, err := vars.NewValue(arg)
		if err != nil {
			return err
		}
		if s.argn == -1 || len(s.args) <= s.argn {
			s.args = append(s.args, a)
		} else {
			break
		}
	}
	return nil
}

func (s *FlagSet) checkAliasShadowing(flag Flag) error {
	if flag.Aliases() == nil {
		return nil
	}
	for _, alias := range flag.Aliases() {
		if flagName, ok := s.aliases[alias]; ok && flag.Name() != flagName {
			return fmt.Errorf("%w: --%s alias -%s shadows --%s alias --%s", ErrAliasShadow, flag.Name(), alias, flagName, alias)
		}
	}
	return nil
}
