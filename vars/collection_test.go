// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vars_test

import (
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/mkungla/vars/v5"

	"github.com/stretchr/testify/assert"
)

var tests = []struct {
	k       string
	defVal  string
	wantLen int
}{
	{"STRING", "one two", 2},
	{"STRING", "one two three four ", 4},
	{"STRING", " one two three four ", 4},
	{"STRING", "1 2 3 4 5 6 7 8.1", 8},
	{"", "", 0},
}

func TestCollectionParseFields(t *testing.T) {
	collection := vars.ParseFromBytes([]byte{})

	for _, tt := range tests {
		val := collection.Get(tt.k, tt.defVal)
		actual := len(val.Fields())
		if actual != tt.wantLen {
			t.Errorf("Value.(%q).ParseFields() len = %d, want %d", tt.k, actual, tt.wantLen)
		}
	}
}

func TestCollectionSet(t *testing.T) {
	collection := new(vars.Collection)
	for _, tt := range tests {
		v := fmt.Sprintf("%v", tt.defVal)
		collection.Store(tt.k, v)
		assert.Equal(t, v, collection.Get(tt.k).String())
		assert.True(t, collection.Has(tt.k))
	}
}

func TestCollectionEnvFile(t *testing.T) {
	content, err := ioutil.ReadFile("testdata/dot_env")
	if err != nil {
		t.Error(err)
	}
	collection := vars.ParseFromBytes(content)
	if val := collection.Get("GOARCH"); val.String() != "amd64" {
		t.Errorf("expected GOARCH to equal amd64 got %s", val)
	}
}

func TestCollectionKeyNoSpaces(t *testing.T) {
	collection := new(vars.Collection)
	collection.Set("valid", true)
	collection.Set(" invalid", true)

	invalid := collection.Get(" invalid")
	valid := collection.Get("valid")

	if invalid.Bool() {
		t.Errorf("Collection key should not accept pfx/sfx  spaces ")
	}
	if !valid.Bool() {
		t.Errorf("Collection key should be true")
	}
}

func TestCollectionParseInt64(t *testing.T) {
	collection := vars.ParseFromBytes(genAtoi64TestBytes())
	for _, test := range intTests {
		val := collection.Get(test.key)
		out := val.Int64()
		if out != test.int64 {
			t.Errorf("2. Value(%s).Int64() = %v) want %v",
				test.key, out, test.int64)
		}
	}
}

func TestCollectionParseUint64(t *testing.T) {
	collection := vars.ParseFromBytes(genAtoui64TestBytes())
	for _, test := range uintTests {
		val := collection.Get(test.key)
		out := val.Uint64()
		if out != test.uint64 {
			t.Errorf("2. Value(%s).Uint64() = %v) want %v",
				test.key, out, test.uint64)
		}
	}
}

func TestCollectionParseFloat32(t *testing.T) {
	collection := vars.ParseFromBytes(genAtof32TestBytes())
	for _, test := range float32Tests {
		val := collection.Get(test.key)
		out := val.Float32()
		if out != test.wantFloat32 {
			if math.IsNaN(float64(out)) && math.IsNaN(float64(test.wantFloat32)) {
				continue
			}
			t.Errorf("2. Value(%s).Float64() = %v) want %v",
				test.key, out, test.wantFloat32)
		}
	}
}

func TestCollectionParseFloat(t *testing.T) {
	collection := vars.ParseFromBytes(genAtofTestBytes())
	for _, test := range float64Tests {
		val := collection.Get(test.key)
		out := val.Float64()

		if val.String() != test.in {
			t.Errorf("1. Value(%s).Float64() = %q) want %q",
				test.key, val.String(), test.in)
		}

		if out != test.wantFloat64 {
			if math.IsNaN(out) && math.IsNaN(test.wantFloat64) {
				continue
			}
			t.Errorf("2. Value(%s).Float64() = %v) want %v",
				test.key, out, test.wantFloat64)
		}
	}
}

func TestCollectionParseBool(t *testing.T) {
	collection := vars.ParseFromBytes(genAtobTestBytes())
	for _, test := range boolTests {
		val := collection.Get(test.key)
		b := val.Bool()
		_, err := vars.NewTyped(test.key, val.String(), vars.TypeBool)
		if test.err != nil {
			if err == nil {
				t.Errorf("Value(%s).ParseBool(): expected %s but got nil", test.key, test.err)
			}
		} else {
			if err != nil {
				t.Errorf("Value(%s).ParseBool(): expected no error but got %s", test.key, err)
			}
			if b != test.want {
				t.Errorf("Value(%s).ParseBool(): = %t, want %t", test.key, b, test.want)
			}
		}
	}
}

func TestCollectionGetWithPrefix(t *testing.T) {
	collection := vars.ParseFromBytes(genStringTestBytes())
	p := collection.GetWithPrefix("CGO")

	if p.Len() != 6 {
		t.Errorf("Collection.GetsWithPrefix(\"CGO\") = %d, want (6)", p.Len())
	}
}

func TestCollectionGetOrDefaultTo(t *testing.T) {
	collection := vars.ParseFromBytes([]byte{})
	tests := []struct {
		k      string
		defVal string
		want   string
	}{
		{"STRING_1", "some-string", "some-string"},
		{"STRING_2", "some-string with space ", "some-string with space "},
		{"STRING_3", " some-string with space", " some-string with space"},
		{"STRING_4", "1234567", "1234567"},
		{"", "1234567", ""},
	}
	for _, tt := range tests {
		if actual := collection.Get(tt.k, tt.defVal); actual.String() != tt.want {
			t.Errorf("Collection.GetOrDefaultTo(%q, %q) = %q, want %q", tt.k, tt.defVal, actual, tt.want)
		}
	}
}

func TestCollectionGetOrDefaultValue(t *testing.T) {
	collection := vars.ParseFromBytes([]byte{})
	tests := []struct {
		k      string
		defVal vars.Value
		want   string
	}{
		{"STRING_1", vars.NewValue("some-string"), "some-string"},
		{"STRING_2", vars.NewValue("some-string with space "), "some-string with space "},
		{"STRING_3", vars.NewValue(" some-string with space"), " some-string with space"},
		{"STRING_4", vars.NewValue("1234567"), "1234567"},
		{"", vars.NewValue("1234567"), ""},
	}
	for _, tt := range tests {
		if actual := collection.Get(tt.k, tt.defVal); actual.String() != tt.want {
			t.Errorf("Collection.GetOrDefaultTo(%q, %q) = %q, want %q", tt.k, tt.defVal, actual, tt.want)
		}
	}
}

func TestCollectionParseFromBytes(t *testing.T) {
	bytes := genStringTestBytes()
	collection := vars.ParseFromBytes(bytes)
	for _, test := range stringTests {
		if actual := collection.Get(test.key); actual.String() != test.val {
			t.Errorf("Collection.Get(%q) = %q, want %q", test.key, actual.String(), test.val)
		}
	}

	sort.Slice(bytes, func(i int, j int) bool { return bytes[i] < bytes[j] })
	bytes2 := collection.ToBytes()
	sort.Slice(bytes2, func(i int, j int) bool { return bytes2[i] < bytes2[j] })
	assert.Equal(t, bytes, bytes2)
}

func TestCollectionParseFromString(t *testing.T) {
	slice := strings.Split(string(genStringTestBytes()), "\n")
	collection := vars.ParseKeyValSlice(slice)
	for _, test := range stringTests {
		if actual := collection.Get(test.key); actual.String() != test.val {
			t.Errorf("Collection.Get(%q) = %q, want %q", test.key, actual.String(), test.val)
		}
	}

	sort.Strings(slice)
	slice2 := collection.ToKeyValSlice()
	slice2 = append(slice2, "")
	sort.Strings(slice2)
	assert.Equal(t, slice, slice2)

	collection2 := vars.ParseKeyValSlice([]string{"X"})
	if actual := collection2.Get("x"); actual.String() != "" {
		t.Errorf("Collection.Get(\"X\") = %q, want \"\"", actual.String())
	}

	collection3 := vars.ParseKeyValSlice([]string{})
	if l := collection3.Len(); l != 0 {
		t.Errorf("Collection.Len() == %d, want \"0\"", l)
	}
}

func TestConcurrentRange(t *testing.T) {
	const mapSize = 1 << 10

	m := new(vars.Collection)
	for n := int64(1); n <= mapSize; n++ {
		m.Store(strconv.Itoa(int(n)), int64(n))
	}

	done := make(chan struct{})
	var wg sync.WaitGroup
	defer func() {
		close(done)
		wg.Wait()
	}()
	for g := int64(runtime.GOMAXPROCS(0)); g > 0; g-- {
		r := rand.New(rand.NewSource(g))
		wg.Add(1)
		go func(g int64) {
			defer wg.Done()
			for i := int64(0); ; i++ {
				select {
				case <-done:
					return
				default:
				}
				for n := int64(1); n < mapSize; n++ {
					key := strconv.Itoa(int(n))
					if r.Int63n(mapSize) == 0 {
						m.Store(strconv.Itoa(int(n)), n*i*g)
					} else {
						m.Load(key)
					}
				}
			}
		}(g)
	}

	iters := 1 << 10
	if testing.Short() {
		iters = 16
	}
	for n := iters; n > 0; n-- {
		seen := make(map[int64]bool, mapSize)

		m.Range(func(ki string, vi vars.Value) bool {
			pk, err := strconv.Atoi(ki)
			k := int64(pk)
			assert.NoError(t, err)
			v := vi.Int64()
			if v%k != 0 {
				t.Fatalf("while Storing multiples of %v, Range saw value %v", k, v)
			}
			if seen[k] {
				t.Fatalf("Range visited key %v twice", k)
			}
			seen[k] = true
			return true
		})

		if len(seen) != mapSize {
			t.Fatalf("Range visited %v elements of %v-element Map", len(seen), mapSize)
		}
	}
}

func TestMissCounting(t *testing.T) {
	m := new(vars.Collection)

	// Since the miss-counting in missLocked (via Delete)
	// compares the miss count with len(m.dirty),
	// add an initial entry to bias len(m.dirty) above the miss count.
	m.Store("", struct{}{})

	var finalized uint32

	// Set finalizers that count for collected keys. A non-zero count
	// indicates that keys have not been leaked.
	for atomic.LoadUint32(&finalized) == 0 {
		p := new(int)
		key := strconv.Itoa(*p)
		runtime.SetFinalizer(p, func(*int) {
			atomic.AddUint32(&finalized, 1)
		})

		m.Store(key, struct{}{})
		m.Delete(key)
		runtime.GC()
	}
}
