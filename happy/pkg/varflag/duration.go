// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package varflag

// DurationFlag defines a time.Duration flag with specified name.
// type DurationFlag struct {
// 	Common
// 	val time.Duration
// }

// // Duration returns new duration flag. Argument "a" can be any nr of aliases.
// func Duration(name string, value time.Duration, usage string, aliases ...string) (flag *DurationFlag, err error) {
// 	if !ValidFlagName(name) {
// 		return nil, fmt.Errorf("%w: flag name %q is not valid", ErrFlag, name)
// 	}

// 	flag = &DurationFlag{}
// 	flag.name = strings.TrimLeft(name, "-")
// 	flag.val = value
// 	flag.aliases = normalizeAliases(aliases)
// 	flag.usage = usage
// 	flag.defval, err = vars.ParseVariableAs(name, value, true, vars.KindDuration)
// 	if err != nil {
// 		return nil, err
// 	}
// 	flag.variable, err = vars.ParseVariableAs(name, value, false, vars.KindDuration)
// 	return flag, err
// }

// func DurationFunc(name string, value time.Duration, usage string, aliases ...string) CreateFlagFunc {
// 	return func() (Flag, error) {
// 		return Duration(name, value, usage, aliases...)
// 	}
// }

// // Parse duration flag.
// func (f *DurationFlag) Parse(args []string) (bool, error) {
// 	return f.parse(args, func(vv []vars.Variable) (err error) {
// 		if len(vv) > 0 {
// 			val, err := vars.ParseVariableAs(f.name, vv[0].String(), vars.KindDuration)
// 			if err != nil {
// 				return fmt.Errorf("%w: %s", ErrInvalidValue, err)
// 			}
// 			f.variable = val
// 			f.val = time.Duration(val.Int64())
// 		}
// 		return err
// 	})
// }

// // Value returns duration flag value, it returns default value if not present
// // or 0 if default is also not set.
// func (f *DurationFlag) Value() time.Duration {
// 	f.mu.RLock()
// 	defer f.mu.RUnlock()
// 	return f.val
// }

// // Unset the bool flag value.
// func (f *DurationFlag) Unset() {
// 	f.mu.Lock()
// 	defer f.mu.Unlock()
// 	f.variable = f.defval
// 	f.isPresent = false
// 	val, _ := time.ParseDuration(f.defval.String())
// 	f.val = val
// }
