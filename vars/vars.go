// Copyright 2012 Marko Kungla.
// Source code is provider under MIT License.

package vars

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// NewValue trims spaces from provided string and returns new Value
func NewValue(val interface{}) Value {
	return Value(strings.TrimSpace(fmt.Sprintf("%v", val)))
}

// NewCollection returns new Collection
func NewCollection() Collection {
	return make(Collection)
}

// ValueFromBool returns Value parsed from bool
func ValueFromBool(val bool) Value {
	return Value(strconv.FormatBool(val))
}

// ParseKeyVal parses variable from single "key=val" pair and
// returns (key string, val Value)
func ParseKeyVal(kv string) (key string, val Value) {
	if len(kv) == 0 {
		return
	}
	reg := regexp.MustCompile(`"([^"]*)"`)

	kv = reg.ReplaceAllString(kv, "${1}")
	l := len(kv)
	for i := 0; i < l; i++ {
		if kv[i] == '=' {
			key = kv[:i]
			val = NewValue(kv[i+1:])
			if i < l {
				return
			}
		}
	}
	// VAR did not have any value
	key = kv[:l]
	val = ""
	return
}

// ParseKeyValSlice parses variables from any []"key=val" slice and
// returns Collection
func ParseKeyValSlice(kv []string) Collection {
	vars := make(Collection)
	if len(kv) == 0 {
		return vars
	}
	reg := regexp.MustCompile(`"([^"]*)"`)

NextVar:
	for _, v := range kv {
		v = reg.ReplaceAllString(v, "${1}")
		l := len(v)
		if l == 0 {
			continue
		}
		for i := 0; i < l; i++ {
			if v[i] == '=' {
				vars[v[:i]] = NewValue(v[i+1:])
				if i < l {
					continue NextVar
				}
			}
		}
		// VAR did not have any value
		vars[strings.TrimRight(v[:l], "=")] = ""
	}
	return vars
}

// ParseFromBytes parses []bytes to string, creates []string by new line
// and calls ParseFromStrings.
func ParseFromBytes(b []byte) Collection {
	slice := strings.Split(string(b[0:]), "\n")
	return ParseKeyValSlice(slice)
}
