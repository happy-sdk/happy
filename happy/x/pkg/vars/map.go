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

package vars

import (
	"sync"
)

// Collection is collection of Variables safe for concurrent use.
type Map struct {
	mu  sync.RWMutex
	len int64
	db  map[string]Variable
}

func (c *Map) All() (all []Variable) {
	c.Range(func(v Variable) bool {
		all = append(all)
		return true
	})
	return
}

// Delete deletes the value for a key.
func (c *Map) Delete(key string) {
	_, _ = c.LoadAndDelete(key)
}

// Get retrieves the value of the variable named by the key.
// It returns the value, which will be empty string if the variable is not set
// or value was empty.
func (c *Map) Get(key string) (v Variable) {
	k, err := parseKey(key)
	if err != nil {
		return
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	v, ok := c.db[k]
	if !ok {
		return EmptyVariable
	}
	return v
}

// GetWithPrefix return all variables with prefix if any as new Map.
func (c *Map) LoadWithPrefix(prfx string) (set *Map, loaded bool) {
	set = new(Map)
	c.Range(func(v Variable) bool {
		key := v.Key()
		if len(key) >= len(prfx) && key[0:len(prfx)] == prfx {
			set.Store(key, v)
			loaded = true
		}
		return true
	})
	return set, loaded
}

// GetWithPrefix return all variables with prefix if any as new Map
// and strip prefix from keys.
func (c *Map) ExtractWithPrefix(prfx string) *Map {
	vars := new(Map)
	c.Range(func(v Variable) bool {
		key := v.Key()
		if len(key) >= len(prfx) && key[0:len(prfx)] == prfx {
			vars.Store(key[len(prfx):], v)
		}
		return true
	})
	return vars
}

// Has reprts whether given variable  exists.
func (c *Map) Has(key string) bool {
	k, err := parseKey(key)
	if err != nil {
		return false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	_, ok := c.db[k]
	return ok
}

// Len of collection.
func (c *Map) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return int(c.len)
}

// Load returns the variable stored in the Collection for a key,
// or EmptyVar if no value is present.
// The ok result indicates whether variable was found in the Collection.
func (c *Map) Load(key string) (v Variable, ok bool) {
	if !c.Has(key) {
		return EmptyVariable, false
	}
	return c.Get(key), true
}

// LoadAndDelete deletes the value for a key, returning the previous value if any.
// The loaded result reports whether the key was present.
func (c *Map) LoadAndDelete(key string) (v Variable, loaded bool) {
	if !c.Has(key) {
		return EmptyVariable, false
	}

	v = c.Get(key)
	loaded = true
	c.mu.Lock()
	delete(c.db, v.Key())
	syncAtomicAddInt64(&c.len, -1)
	c.mu.Unlock()
	return
}

// LoadOrDefault returns the existing value for the key if present.
// Much like LoadOrStore, but second argument willl be returned as
// Value whithout being stored into Collection.
func (c *Map) LoadOrDefault(key string, value any) (v Variable, loaded bool) {
	k, err := parseKey(key)
	if err != nil {
		return EmptyVariable, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	// existing
	if val, ok := c.db[k]; ok {
		return val, true
	}
	if len(k) > 0 {
		if def, ok := value.(Variable); ok {
			return def, false
		}
	}
	v, err = NewVariable(k, value, false)
	if err != nil {

		return EmptyVariable, false
	}
	return v, false
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
func (c *Map) LoadOrStore(key string, value any) (actual Variable, loaded bool) {
	k, err := parseKey(key)
	if err != nil {
		return EmptyVariable, false
	}
	loaded = c.Has(k)
	if !loaded {
		c.Store(k, value)
	}
	return c.Get(k), loaded
}

// func (c *Collection) MarshalJSON() ([]byte, error) {
// 	pl := make(map[string]any)
// 	c.Range(func(v Variable) bool {
// 		pl[v.Key()] = v.val.raw
// 		return true
// 	})
// 	return json.Marshal(pl)
// }

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
//
// Range does not necessarily correspond to any consistent snapshot of the Map's
// contents: no key will be visited more than once, but if the value for any key
// is stored or deleted concurrently, Range may reflect any mapping for that key
// from any point during the Range call.
//
// Range may be O(N) with the number of elements in the map even if f returns
// false after a constant number of calls.
func (c *Map) Range(f func(v Variable) bool) {
	c.mu.RLock()
	for _, v := range c.db {
		c.mu.RUnlock()
		if !f(v) {
			break
		}
		c.mu.RLock()
	}
	c.mu.RUnlock()
}

// Store sets the value for a key.
// If it fails to parse val then key will be set to
// EmptyValue with TypeInvalid. Safest would be to
// store Value which error has been already checked.
func (c *Map) Store(k string, value any) {
	key, err := parseKey(k)
	if err != nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.db == nil {
		c.db = make(map[string]Variable)
	}

	if vv, ok := value.(Variable); ok && vv.Key() == key {
		c.db[key] = vv
	} else {
		c.db[key], _ = NewVariable(key, value, false)
	}

	syncAtomicAddInt64(&c.len, 1)
}

// ToBytes returns []byte containing
// key = "value"\n.
func (c *Map) ToBytes() []byte {
	s := c.ToKeyValSlice()

	p := getParser()
	defer p.free()

	for _, line := range s {
		p.fmt.string(line + "\n")
	}
	return p.buf
}

// ToKeyValSlice produces []string slice of strings in format key = "value".
func (c *Map) ToKeyValSlice() []string {
	r := []string{}
	c.Range(func(v Variable) bool {
		// we can do it directly on interface value since they all are Values
		// implementing Stringer
		r = append(r, v.Key()+"=\""+v.String()+"\"")
		return true
	})
	return r
}

type MapIface[VAR VariableIface[VAL], VAL ValueIface] interface {
	Store(key string, value any)
	Len() int
	Delete(key string)
	Get(key string) VAR
	Load(key string) (v VAR, ok bool)
	LoadAndDelete(key string) (v VAR, loaded bool)
	LoadOrDefault(key string, value any) (v VAR, loaded bool)
	LoadOrStore(key string, value any) (actual VAR, loaded bool)
	Range(f func(v VAR) bool)
	All() []VAR
}

type GenericVariableMap[
	MAP MapIface[VAR, VAL],
	VAR VariableIface[VAL],
	VAL ValueIface,
] struct {
	m *Map
}

func (m GenericVariableMap[MAP, VAR, VAL]) Len() int {
	return m.m.Len()
}
func (m GenericVariableMap[MAP, VAR, VAL]) Has(key string) bool {
	return m.m.Has(key)
}

func (m GenericVariableMap[MAP, VAR, VAL]) Delete(key string) {
	m.m.Delete(key)
}

func (m GenericVariableMap[MAP, VAR, VAL]) Store(key string, value any) {
	m.m.Store(key, value)
}

func (m GenericVariableMap[MAP, VAR, VAL]) Get(key string) VAR {
	return AsVariable[VAR, VAL](m.m.Get(key))
}

func (m GenericVariableMap[MAP, VAR, VAL]) LoadWithPrefix(prfx string) (set MAP, loaded bool) {
	rm, ok := m.m.LoadWithPrefix(prfx)
	loaded = ok
	return AsMap[MAP, VAR, VAL](rm), loaded
}

func (m GenericVariableMap[MAP, VAR, VAL]) ExtractWithPrefix(prfx string) MAP {
	rm := m.m.ExtractWithPrefix(prfx)
	return AsMap[MAP, VAR, VAL](rm)
}

func (m GenericVariableMap[MAP, VAR, VAL]) Load(key string) (v VAR, ok bool) {
	mm, ok := m.m.Load(key)
	return AsVariable[VAR, VAL](mm), ok
}

func (m GenericVariableMap[MAP, VAR, VAL]) LoadAndDelete(key string) (v VAR, loaded bool) {
	mm, ok := m.m.LoadAndDelete(key)
	return AsVariable[VAR, VAL](mm), ok
}

func (m GenericVariableMap[MAP, VAR, VAL]) LoadOrDefault(key string, value any) (v VAR, loaded bool) {
	mm, ok := m.m.LoadOrDefault(key, value)
	return AsVariable[VAR, VAL](mm), ok
}

func (m GenericVariableMap[MAP, VAR, VAL]) LoadOrStore(key string, value any) (actual VAR, loaded bool) {
	mm, ok := m.m.LoadOrStore(key, value)
	return AsVariable[VAR, VAL](mm), ok
}

func (m GenericVariableMap[MAP, VAR, VAL]) Range(f func(v VAR) bool) {
	m.m.Range(func(orig Variable) bool {
		v := AsVariable[VAR, VAL](orig)
		return f(v)
	})
}

func (m GenericVariableMap[MAP, VAR, VAL]) All() (all []VAR) {
	m.m.Range(func(orig Variable) bool {
		v := AsVariable[VAR, VAL](orig)
		all = append(all, v)
		return true
	})
	return
}
