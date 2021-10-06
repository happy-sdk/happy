// Copyright 2016 Marko Kungla. All rights reserved.
// Use of this source code is governed by a The Apache-style
// license that can be found in the LICENSE file.

package varflag

import (
	"os"

	"github.com/mkungla/vars/v5"
)

// Name returns name of the flag set.
func (s *FlagSet) Name() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.name
}

// Add flag to flag set.
func (s *FlagSet) Add(flag ...Flag) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, f := range flag {
		f.BelongsTo(s.name)
	}
	s.flags = append(s.flags, flag...)
}

// AddSet adds flag set.
func (s *FlagSet) AddSet(set ...Flags) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sets = append(s.sets, set...)
}

// GetActiveSetName return last known command
// which was present while parsing. e.g subcommand.
func (s *FlagSet) GetActiveSetName() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var name string
	if s.Present() {
		name = s.name
		for _, set := range s.sets {
			if s.Present() {
				name = set.GetActiveSetName()
			}
		}
	}
	return name
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

// ////////////////////////////////////////////////////////////////////////////
// some shameful cognitive complexity
// ////////////////////////////////////////////////////////////////////////////

// Parse all flags recursively.
//nolint: gocognit, cyclop
func (s *FlagSet) Parse(args []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(args) > 0 && args[0] == os.Args[0] {
		args = args[1:]
	}

	var currargs []string
	if s.name != "/" && s.name != "*" {
		for i, arg := range args {
			if arg == s.name {
				s.pos = i
				currargs = args[i:]
				s.present = true
			}
		}
	} else {
		currargs = args
	}

	// first parse flags for current set
	for _, gflag := range s.flags {
		_, err := gflag.Parse(currargs)
		if err != nil {
			return err
		}
		// this flag need to be removed from sub command args
		if gflag.Present() {
			currargs = slicediff(currargs, gflag.input())
		}
	}

	// parse flags for sets
	for _, set := range s.sets {
		err := set.Parse(currargs)
		if err != nil {
			return err
		}
		if set.Present() {
			if s.name == "/" {
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
					// there is no args for cuurent set since
					// there is sub set which was first arg
					break includessubset
				}
			}
		}
	}

	// filter flags
	used := []string{}
	if args[0] == s.name {
		used = append(used, s.name)
	}
	// for _, flag := range s.flags {
	// 	if !flag.Present() {
	// 		// get the flag arg
	// 		// we can not use flag pos since flag was positioned after argument
	// 		if flag.Pos() == 0 {
	// 			return fmt.Errorf("%w: flag %s position was 0", ErrParse, flag.Name())
	// 		}
	// 		pose := flag.Pos() - 1
	// 		arg := args[pose]
	// 		// is flag key=val
	// 		if strings.Contains(arg, "=") {
	// 			used = append(used, args[pose])
	// 			continue
	// 		}
	// 		// bool flag
	// 		if _, ok := flag.(*BoolFlag); ok {
	// 			used = append(used, args[pose])
	// 			if len(args) > pose {
	// 				val := args[pose+1]
	// 				if val == "true" || val == "1" || val == "on" || val == "false" || val == "0" || val == "off" {
	// 					used = append(used, val)
	// 				}
	// 			}
	// 		} else {
	// 			// flag with value
	// 			// should be safe to use index +1 since
	// 			// flag was parsed correctly
	// 			used = append(used, args[pose], args[pose+1])
	// 		}
	// 	}
	// }
	sargs := slicediff(args, used)
	for _, arg := range sargs {
		s.args = append(s.args, vars.NewValue(arg))
	}
	return nil
}
