// Copyright 2012 Marko Kungla.
// Source code is provider under MIT License.

package vars

import (
	"bytes"
	"fmt"
	"strings"
)

// Collection holds collection of variables
type Collection map[string]Value

// Get retrieves the value of the variable named by the key.
// It returns the value, which will be empty string if the variable is not set
// or value was empty.
func (c Collection) Get(k string, defval ...interface{}) (val Value) {
	val, ok := c[k]
	if len(k) == 0 || !ok || len(val) == 0 {
		if len(defval) > 0 {
			s, _ := ParseValue(defval[0])
			return s
		}
	}
	return val
}

// Set updates key value pair in collection. If key does not exist then appends
// key wth given value
func (c Collection) Set(k string, v interface{}) {
	k = strings.TrimSpace(k)
	v = strings.TrimSpace(fmt.Sprint(v))
	c[k], _ = ParseValue(v)
}

// GetWithPrefix return all variables with prefix if any as map[]
func (c Collection) GetWithPrefix(prfx string) (vars Collection) {
	vars = make(Collection)
	for k, v := range c {
		if len(k) >= len(prfx) && k[0:len(prfx)] == prfx {
			vars[k] = v
		}
	}
	return
}

// ToKeyValSlice produces []string slice of strings in format key = "value"
func (c Collection) ToKeyValSlice() []string {
	r := []string{}
	for k, v := range c {
		r = append(r, fmt.Sprintf("%s = %q", k, v))
	}
	return r
}

// ToBytes returns []byte containing
// key = "value"\n
func (c Collection) ToBytes() []byte {
	s := c.ToKeyValSlice()
	b := bytes.Buffer{}
	for _, line := range s {
		b.WriteString(line + "\n")
	}
	return b.Bytes()
}
