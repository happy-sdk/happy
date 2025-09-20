// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package vars_test

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"reflect"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"github.com/happy-sdk/happy/pkg/vars"
)

func TestMapParseFields(t *testing.T) {
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

	collection, err := vars.ParseMapFromBytes([]byte{})
	testutils.NoError(t, err)

	for _, tt := range tests {
		val, _ := collection.LoadOrDefault(tt.k, tt.defVal)
		actual := len(val.Fields())
		if actual != tt.wantLen {
			t.Errorf("Value.(%q).ParseFields() len = %d, want %d", tt.k, actual, tt.wantLen)
		}
	}
}

func TestMapSet(t *testing.T) {
	var tests = []struct {
		k       string
		defVal  string
		wantLen int
	}{
		{"STRING1", "one two", 2},
		{"STRING2", "one two three four ", 4},
		{"STRING3", " one two three four ", 4},
		{"STRING4", "1 2 3 4 5 6 7 8.1", 8},
		{"", "", 0},
	}

	collection := vars.NewMap()
	for _, tt := range tests {
		if tt.wantLen == 0 {
			continue
		}
		v := fmt.Sprintf("%v", tt.defVal)
		err := collection.StoreReadOnly(tt.k, v, true)
		testutils.NoError(t, err)
		testutils.Equal(t, v, collection.Get(tt.k).String())
		testutils.Assert(t, collection.Has(tt.k))

		err2 := collection.StoreReadOnly(tt.k, v, true)
		testutils.ErrorIs(t, err2, vars.ErrReadOnly)

		err3 := collection.StoreReadOnly("$"+tt.k, v, true)
		testutils.ErrorIs(t, err3, vars.ErrKeyHasIllegalChar)
		testutils.ErrorIs(t, err3, vars.ErrKey)

	}
	_, loaded := collection.LoadOrDefault("pre_", "")
	testutils.Equal(t, false, loaded)
	_, loaded2 := collection.LoadOrStore("$pre_", "")
	testutils.Equal(t, false, loaded2)
	_, loaded3 := collection.LoadOrStore("pre_3", "")
	testutils.Equal(t, false, loaded3)

	testutils.Equal(t, 5, collection.Len())
	testutils.Equal(t, 5, len(slices.Collect(collection.All())))
}

func TestMapEnvFile(t *testing.T) {
	content, err := os.ReadFile("testdata/dot_env")
	if err != nil {
		t.Error(err)
	}
	collection, err := vars.ParseMapFromBytes(content)
	testutils.NoError(t, err)
	if val := collection.Get("GOARCH"); val.String() != "amd64" {
		t.Errorf("expected GOARCH to equal amd64 got %s", val)
	}
}

func TestMapKeyNoSpaces(t *testing.T) {
	collection := vars.Map{}
	testutils.NoError(t, collection.Store("valid", true))
	testutils.NoError(t, collection.Store(" invalid", true))

	invalid := collection.Get(" invalid")
	valid := collection.Get("valid")

	if !invalid.Bool() {
		t.Error("Map key should correct pfx/sfx spaces")
	}
	if !valid.Bool() {
		t.Error("Map key should be true")
	}
}

func genAtoi64TestBytes() []byte {
	var out []byte
	for _, data := range getIntTests() {
		line := fmt.Sprintf(`%s="%s"`+"\n", data.Key, data.Val)
		out = append(out, []byte(line)...)
	}
	return out
}

func TestMapParseInt64(t *testing.T) {
	collection, err := vars.ParseMapFromBytes(genAtoi64TestBytes())
	testutils.NoError(t, err)
	for _, test := range getIntTests() {
		val := collection.Get(test.Key)
		out := val.Int64()
		if out != test.Int64 {
			t.Errorf("2. Value(%s).Int64() = %v) want %v",
				test.Key, out, test.Int64)
		}
	}
}

func genAtoui64TestBytes() []byte {
	var out []byte
	for _, data := range getUintTests() {
		line := fmt.Sprintf(`%s="%s"`+"\n", data.Key, data.Val)
		out = append(out, []byte(line)...)
	}
	return out
}

func TestMapParseUint64(t *testing.T) {
	collection, err := vars.ParseMapFromBytes(genAtoui64TestBytes())
	testutils.NoError(t, err)
	for _, test := range getUintTests() {
		val := collection.Get(test.Key)
		out := val.Uint64()
		if out != test.Uint64 {
			t.Errorf("2. Value(%s).Uint64() = %v) want %v",
				test.Key, out, test.Uint64)
		}
	}
}

func genAtof32TestBytes() []byte {
	var out []byte
	for _, data := range getFloat32Tests() {
		line := fmt.Sprintf(`%s="%s"`+"\n", data.Key, data.In)
		out = append(out, []byte(line)...)
	}
	return out
}

func TestMapParseFloat32(t *testing.T) {
	collection, err := vars.ParseMapFromBytes(genAtof32TestBytes())
	testutils.NoError(t, err)

	for _, test := range getFloat32Tests() {
		val := collection.Get(test.Key)
		out := val.Float32()
		if out != test.WantFloat32 {
			if math.IsNaN(float64(out)) && math.IsNaN(float64(test.WantFloat32)) {
				continue
			}
			t.Errorf("2. Value(%s).Float64() = %v) want %v",
				test.Key, out, test.WantFloat32)
		}
	}
}

func genAtofTestBytes() []byte {
	var out []byte
	for _, data := range getFloat64Tests() {
		line := fmt.Sprintf(`%s="%s"`+"\n", data.Key, data.In)
		out = append(out, []byte(line)...)
	}
	return out
}

func TestMapParseFloat(t *testing.T) {
	collection, err := vars.ParseMapFromBytes(genAtofTestBytes())
	testutils.NoError(t, err)
	for _, test := range getFloat64Tests() {
		val := collection.Get(test.Key)
		out := val.Float64()

		if val.String() != test.In {
			t.Errorf("1. Value(%s).Float64() = %q) want %q",
				test.Key, val.String(), test.In)
		}

		if out != test.WantFloat64 {
			if math.IsNaN(out) && math.IsNaN(test.WantFloat64) {
				continue
			}
			t.Errorf("2. Value(%s).Float64() = %v) want %v",
				test.Key, out, test.WantFloat64)
		}
	}
}

func TestMapGetWithPrefix(t *testing.T) {
	collection, err := vars.ParseMapFromBytes(genStringTestBytes())
	testutils.NoError(t, err)
	p, loaded := collection.LoadWithPrefix("CGO")
	if !loaded {
		t.Error("expected loaded to be true")
	}
	if p.Len() != 6 {
		t.Errorf("Map.GetsWithPrefix(\"CGO\") = %d, want (6)", p.Len())
	}

	p2 := collection.ExtractWithPrefix("CGO")
	if p2.Len() != 6 {
		t.Errorf("Map.GetsWithPrefix(\"CGO\") = %d, want (6)", p2.Len())
	}
}

func TestMapGetOrDefault(t *testing.T) {
	collection, err := vars.ParseMapFromBytes([]byte{})
	testutils.NoError(t, err)
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
		if actual, _ := collection.LoadOrDefault(tt.k, tt.defVal); actual.String() != tt.want {
			t.Errorf("Map.GetOrDefaultTo(%q, %q) = %q, want %q", tt.k, tt.defVal, actual, tt.want)
		}
	}
}

func TestMapGetOrDefaultValue(t *testing.T) {
	collection, err := vars.ParseMapFromBytes([]byte{})
	testutils.NoError(t, err)
	tests := []struct {
		k      string
		defVal vars.Value
		want   string
	}{
		{"STRING_1", newUnsafeValue("some-string"), "some-string"},
		{"STRING_2", newUnsafeValue("some-string with space "), "some-string with space "},
		{"STRING_3", newUnsafeValue(" some-string with space"), " some-string with space"},
		{"STRING_4", newUnsafeValue("1234567"), "1234567"},
		{"", newUnsafeValue("1234567"), ""},
	}
	for _, tt := range tests {
		if actual, _ := collection.LoadOrDefault(tt.k, tt.defVal); actual.String() != tt.want {
			t.Errorf("Map.LoadOrDefault(%q, %q) = %q, want %q got ", tt.k, tt.defVal, actual, tt.want)
		}
	}
}

func TestMapParseMapFromBytes(t *testing.T) {
	bytes := genStringTestBytes()
	collection, err := vars.ParseMapFromBytes(bytes)
	testutils.NoError(t, err)
	for _, test := range getStringTests() {
		if actual := collection.Get(test.Key); actual.String() != test.Val || actual.Any() != test.Val {
			t.Errorf("Map.Get(%q) = %q, want %q", test.Key, actual.String(), test.Val)
		}
	}
	slices.Sort(bytes)
	bytes2 := collection.ToBytes()
	slices.Sort(bytes2)
	testutils.EqualAny(t, bytes, bytes2)
}

func TestMapParseFromString(t *testing.T) {
	slice := strings.Split(string(genStringTestBytes()), "\n")
	collection, err := vars.ParseMapFromSlice(slice)
	testutils.NoError(t, err)
	for _, test := range getStringTests() {
		if actual := collection.Get(test.Key); actual.String() != test.Val || actual.Any() != test.Val {
			t.Errorf("Map.Get(%q) = %q, want %q", test.Key, actual.String(), test.Val)
		}
	}

	sort.Strings(slice)
	slice2 := collection.ToKeyValSlice()
	slice2 = append(slice2, "")
	sort.Strings(slice2)
	testutils.EqualAny(t, slice, slice2)

	collection2, err := vars.ParseMapFromSlice([]string{"X"})
	testutils.NoError(t, err)
	if actual := collection2.Get("x"); actual.String() != "" {
		t.Errorf("Map.Get(\"X\") = %q, want \"\"", actual.String())
	}

	collection3, err := vars.ParseMapFromSlice([]string{})
	testutils.NoError(t, err)
	if l := collection3.Len(); l != 0 {
		t.Errorf("Map.Len() == %d, want \"0\"", l)
	}
}

func TestConcurrentRange(t *testing.T) {
	const mapSize = 1 << 10

	m := vars.Map{}
	for n := int64(1); n <= mapSize; n++ {
		testutils.NoError(t, m.Store("k"+strconv.Itoa(int(n)), n))
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
					key := "k" + strconv.Itoa(int(n))
					if r.Int63n(mapSize) == 0 {
						testutils.NoError(t, m.Store(key, n*i*g))
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
		seen := make(map[string]bool, mapSize)

		m.Range(func(vi vars.Variable) bool {
			pk, err := strconv.Atoi(vi.Name()[1:])
			k := int64(pk)
			testutils.NoError(t, err)
			v := vi.Int64()
			if v%k != 0 {
				t.Fatalf("while Storing multiples of %v, Range saw value %v", k, v)
			}
			if seen[vi.Name()] {
				t.Fatalf("Range visited key %v twice", k)
			}
			seen[vi.Name()] = true
			return true
		})

		if len(seen) != mapSize {
			t.Fatalf("Range visited %v elements of %v-element Map", len(seen), mapSize)
		}
	}
}

func TestMissCounting(t *testing.T) {
	m := vars.Map{}

	// Since the miss-counting in missLocked (via Delete)
	// compares the miss count with len(m.dirty),
	// add an initial entry to bias len(m.dirty) above the miss count.
	_ = m.Store("", struct{}{})

	var finalized uint32

	// Set finalizers that count for collected keys. A non-zero count
	// indicates that keys have not been leaked.
	for atomic.LoadUint32(&finalized) == 0 {
		p := new(int)
		key := "k" + strconv.Itoa(*p)
		runtime.SetFinalizer(p, func(*int) {
			atomic.AddUint32(&finalized, 1)
		})

		testutils.NoError(t, m.Store(key, struct{}{}))
		m.Delete(key)
		runtime.GC()
	}
}

func TestMapRangeNestedCall(t *testing.T) {
	var c vars.Map
	for i, v := range [3]string{"hello", "world", "Go"} {
		testutils.NoError(t, c.Store(fmt.Sprintf("k%d", i), v))
	}
	c.Range(func(v vars.Variable) bool {
		c.Range(func(v vars.Variable) bool {
			// We should be able to load the key offered in the Range callback,
			// because there are no concurrent Delete involved in this tested map.
			if vv, ok := c.Load(v.Name()); !ok || !reflect.DeepEqual(vv.Value(), v.Value()) {
				t.Fatalf("Nested Range loads unexpected value, got %+v want %+v", v, v.Value())
			}

			// We didn't keep 42 and a value into the map before, if somehow we loaded
			// a value from such a key, meaning there must be an internal bug regarding
			// nested range in the Map.
			if vv, loaded := c.LoadOrStore("k42", "dummy"); loaded {
				t.Fatalf("Nested Range loads unexpected value, want store a new value %q = %q", vv.Name(), vv.String())
			}

			// Try to Store then LoadAndDelete the corresponding value with the key
			// 42 to the Map. In this case, the key 42 and associated value should be
			// removed from the Map. Therefore any future range won't observe key 42
			// as we checked in above.
			val := "vars.Map"
			testutils.NoError(t, c.Store("k42", val))
			if vv, loaded := c.LoadAndDelete("k42"); !loaded || !reflect.DeepEqual(vv.Any(), val) {
				t.Fatalf("Nested Range loads unexpected value, got %v, want %v", vv, val)
			}
			return true
		})
		// Remove key from Map on-the-fly.
		c.Delete(v.Name())
		return true
	})
	// After a Range of Delete, all keys should be removed and any
	// further Range won't invoke the callback. Hence length remains 0.
	length := 0
	c.Range(func(v vars.Variable) bool {
		length++
		return true
	})
	if length != 0 {
		t.Fatalf("Unexpected vars.Map size, got %v want %v", length, 0)
	}
}

// func TestExpectedEmptyVars(t *testing.T) {
// 	var c vars.Map
// 	if v, loaded := c.Load("test"); loaded || v.Kind() != vars.KindInvalid {
// 		t.Fatalf("Load: did not expect value in new collection got %q", v)
// 	}
// 	if v, loaded := c.LoadAndDelete("test"); loaded || v.Kind() != vars.KindInvalid {
// 		t.Fatalf("LoadAndDelete: did not expect value in new collection got %q", v)
// 	}

// 	val1 := "test1"
// 	testutils.NoError(t, c.Store("test", val1))
// 	if v, loaded := c.LoadOrDefault("test", "test2"); !loaded || v.String() != val1 {
// 		t.Fatalf("LoadOrDefault: unexpected value %q", v)
// 	}

// 	if v, loaded := c.LoadOrDefault("test2", c.Get("test")); loaded {
// 		t.Fatalf("LoadOrDefault: unexpected value %q", v)
// 	}
// 	testutils.NoError(t, c.Store("test2", c.Get("test")))

// 	c.Delete("test2")

// 	if c.Has("test2") {
// 		t.Error("var with key test2 was not deleted")
// 	}
// 	if v, loaded := c.LoadOrStore("test", "test3"); !loaded {
// 		t.Fatalf("LoadOrStore: unexpected value %q", v)
// 	}

// 	if c.Has(" ") {
// 		t.Fatal("empty key lookup returned true")
// 	}
// 	if v := c.Get(" "); !v.Empty() {
// 		t.Fatal("empty value lookup returned value")
// 	}
// }

// func TestJSON(t *testing.T) {
// 	m := vars.Map{}
// 	testutils.NoError(t, m.Store("key1", "value1"))
// 	testutils.NoError(t, m.Store("key2", "value2"))
// 	jsonData, err := json.Marshal(&m)
// 	testutils.NoError(t, err)

// 	var newmap vars.Map
// 	err = json.Unmarshal(jsonData, &newmap)
// 	testutils.NoError(t, err)
// 	testutils.Equal(t, "value1", newmap.Get("key1").String())
// 	testutils.Equal(t, "value2", newmap.Get("key2").String())
// }

// func genAtobTestBytes() []byte {
// 	var out []byte
// 	for _, data := range getBoolTests() {
// 		line := fmt.Sprintf(`%s="%s"`+"\n", data.Key, data.In)
// 		out = append(out, []byte(line)...)
// 	}
// 	return out
// }

// func TestMapParseBool(t *testing.T) {
// 	collection, err := vars.ParseMapFromBytes(genAtobTestBytes())
// 	testutils.NoError(t, err)
// 	for _, test := range getBoolTests() {
// 		val := collection.Get(test.Key)

// 		if b := val.Bool(); b != test.Want {
// 			t.Errorf("Value(%s).ParseBool(): = %t, want %t", test.Key, b, test.Want)
// 		}
// 	}
// }

// func TestMap_Range(t *testing.T) {
// 	collection := vars.NewMap()
// 	testutils.NoError(t, collection.Store("key1", "value1"))
// 	testutils.NoError(t, collection.Store("key2", "value2"))

// 	var keys []string
// 	collection.Range(func(value vars.Variable) bool {
// 		keys = append(keys, value.Name())
// 		return true
// 	})

// 	testutils.Equal(t, 0, slices.Compare(keys, []string{"key1", "key2"}))
// }

// func TestMap_ToGoMap(t *testing.T) {
// 	collection := vars.NewMap()
// 	testutils.NoError(t, collection.Store("key1", "value1"))
// 	testutils.NoError(t, collection.Store("key2", "value2"))

// 	goMap := collection.ToGoMap()
// 	testutils.Equal(t, "value1", goMap["key1"])
// 	testutils.Equal(t, "value2", goMap["key2"])
// }

// func TestReadOnlyMap_From(t *testing.T) {
// 	collection := vars.NewMap()
// 	testutils.NoError(t, collection.Store("key1", "value1"))
// 	testutils.NoError(t, collection.Store("key2", "value2"))

// 	readOnlyMap, err := vars.ReadOnlyMapFrom(collection)
// 	testutils.NoError(t, err)
// 	testutils.Equal(t, "value1", readOnlyMap.Get("key1").String())
// 	testutils.Equal(t, "value2", readOnlyMap.Get("key2").String())
// 	testutils.True(t, readOnlyMap.Has("key1"))
// 	testutils.True(t, readOnlyMap.Has("key2"))
// 	testutils.False(t, readOnlyMap.Has("key3"))
// 	testutils.Len(t, readOnlyMap, 2)
// }

// func TestReadOnlyMap_All(t *testing.T) {
// 	collection := vars.NewMap()
// 	testutils.NoError(t, collection.Store("key1", "value1"))
// 	testutils.NoError(t, collection.Store("key2", "value2"))

// 	readOnlyMap, err := vars.ReadOnlyMapFrom(collection)
// 	testutils.NoError(t, err)
// 	vals := readOnlyMap.All()
// 	testutils.Len(t, vals, 2)
// 	testutils.Equal(t, "key1", vals[0].Name())
// 	testutils.Equal(t, "value1", vals[0].String())
// 	testutils.Equal(t, "key2", vals[1].Name())
// 	testutils.Equal(t, "value2", vals[1].String())
// }

// func TestReadOnlyMap_Load(t *testing.T) {
// 	collection := vars.NewMap()
// 	testutils.NoError(t, collection.Store("key1", "value1"))
// 	testutils.NoError(t, collection.Store("key2", "value2"))

// 	readOnlyMap, err := vars.ReadOnlyMapFrom(collection)
// 	testutils.NoError(t, err)
// 	testutils.Len(t, readOnlyMap, 2)
// 	v1, ok := readOnlyMap.Load("key1")
// 	testutils.True(t, ok)
// 	testutils.Equal(t, "value1", v1.String())
// 	v2, ok := readOnlyMap.Load("key2")
// 	testutils.True(t, ok)
// 	testutils.Equal(t, "value2", v2.String())
// 	v3, ok := readOnlyMap.Load("key3")
// 	testutils.False(t, ok)
// 	testutils.Equal(t, vars.EmptyValue.String(), v3.String())
// }

// func TestReadOnlyMap_LoadOrDefault(t *testing.T) {
// 	collection := vars.NewMap()
// 	testutils.NoError(t, collection.Store("key1", "value1"))
// 	testutils.NoError(t, collection.Store("key2", "value2"))

// 	readOnlyMap, err := vars.ReadOnlyMapFrom(collection)
// 	testutils.NoError(t, err)
// 	testutils.Len(t, readOnlyMap, 2)
// 	v1, ok := readOnlyMap.LoadOrDefault("key1", vars.EmptyValue)
// 	testutils.True(t, ok)
// 	testutils.Equal(t, "value1", v1.String())
// 	v2, ok := readOnlyMap.LoadOrDefault("key2", vars.EmptyValue)
// 	testutils.True(t, ok)
// 	testutils.Equal(t, "value2", v2.String())
// 	v3, ok := readOnlyMap.LoadOrDefault("key3", vars.EmptyValue)
// 	testutils.False(t, ok)
// 	testutils.Equal(t, vars.EmptyValue.String(), v3.String())
// 	v4, ok := readOnlyMap.LoadOrDefault("key4", "default_val")
// 	testutils.False(t, ok)
// 	testutils.Equal(t, "default_val", v4.String())
// }

// func TestReadOnlyMap_WithPrefix(t *testing.T) {
// 	t.Run("invalid", func(t *testing.T) {
// 		collection := vars.NewMap()
// 		testutils.NoError(t, collection.Store("key.1", "value1"))
// 		testutils.NoError(t, collection.Store("key.2", "value2"))
// 		testutils.NoError(t, collection.Store("other_key.1", "other_value1"))

// 		testutils.Len(t, collection, 3)
// 		readOnlyMap, err := vars.ReadOnlyMapFrom(collection)
// 		testutils.NoError(t, err)
// 		testutils.Len(t, readOnlyMap, 3)
// 		invalidCollection, err := readOnlyMap.WithPrefix("key.")
// 		testutils.Error(t, err)
// 		testutils.Len(t, invalidCollection, 0)
// 	})
// 	t.Run("valid", func(t *testing.T) {
// 		collection := vars.NewMap()
// 		testutils.NoError(t, collection.Store("key.v1", "value1"))
// 		testutils.NoError(t, collection.Store("key.v2", "value2"))
// 		testutils.NoError(t, collection.Store("other_key.v1", "other_value1"))

// 		testutils.Len(t, collection, 3)
// 		readOnlyMap, err := vars.ReadOnlyMapFrom(collection)
// 		testutils.NoError(t, err)
// 		testutils.Len(t, readOnlyMap, 3)
// 		validCollection, err := readOnlyMap.WithPrefix("key.")
// 		testutils.NoError(t, err)
// 		testutils.Len(t, validCollection, 2)
// 	})
// }

// func TestReadOnlyMap_LoadWithPrefix(t *testing.T) {
// 	t.Run("invalid", func(t *testing.T) {
// 		collection := vars.NewMap()
// 		testutils.NoError(t, collection.Store("key.1", "value1"))
// 		testutils.NoError(t, collection.Store("key.2", "value2"))
// 		testutils.NoError(t, collection.Store("other_key.1", "other_value1"))
// 		readOnlyMap, err := vars.ReadOnlyMapFrom(collection)
// 		testutils.NoError(t, err)
// 		validCollection, ok := readOnlyMap.LoadWithPrefix("key.")
// 		testutils.Len(t, validCollection, 0)
// 		testutils.False(t, ok)
// 	})
// 	t.Run("valid", func(t *testing.T) {
// 		collection := vars.NewMap()
// 		testutils.NoError(t, collection.Store("key.v1", "value1"))
// 		testutils.NoError(t, collection.Store("key.v2", "value2"))
// 		testutils.NoError(t, collection.Store("other_key.v1", "other_value1"))
// 		readOnlyMap, err := vars.ReadOnlyMapFrom(collection)
// 		testutils.NoError(t, err)
// 		validCollection, ok := readOnlyMap.LoadWithPrefix("key.")
// 		testutils.Len(t, validCollection, 2)
// 		testutils.True(t, ok)
// 		testutils.Equal(t, "", readOnlyMap.Get("other_key.v").String())
// 	})
// }

// func TestReadOnlyMap_ToBytes(t *testing.T) {
// 	t.Run("empty", func(t *testing.T) {
// 		collection := vars.NewMap()
// 		readOnlyMap, err := vars.ReadOnlyMapFrom(collection)
// 		testutils.NoError(t, err)
// 		testutils.True(t, bytes.Equal(readOnlyMap.ToBytes(), []byte("")))
// 	})
// 	t.Run("valid", func(t *testing.T) {
// 		collection := vars.NewMap()
// 		testutils.NoError(t, collection.Store("key.v1", "value1"))
// 		testutils.NoError(t, collection.Store("key.v2", "value2"))
// 		testutils.NoError(t, collection.Store("other_key.v1", "other_value1"))
// 		readOnlyMap, err := vars.ReadOnlyMapFrom(collection)
// 		testutils.NoError(t, err)
// 		testutils.True(t, bytes.Equal(readOnlyMap.ToBytes(), []byte("key.v1=value1\nkey.v2=value2\nother_key.v1=other_value1\n")))
// 	})
// }

// func TestReadOnlyMap_MarshalJSON(t *testing.T) {
// 	t.Run("empty", func(t *testing.T) {
// 		collection := vars.NewMap()
// 		readOnlyMap, err := vars.ReadOnlyMapFrom(collection)
// 		testutils.NoError(t, err)
// 		b, err := json.Marshal(readOnlyMap)
// 		testutils.NoError(t, err)
// 		testutils.True(t, bytes.Equal(b, []byte("null")))
// 	})
// 	t.Run("valid", func(t *testing.T) {
// 		collection := vars.NewMap()
// 		testutils.NoError(t, collection.Store("key.v1", "value1"))
// 		testutils.NoError(t, collection.Store("key.v2", "value2"))
// 		testutils.NoError(t, collection.Store("other_key.v1", "other_value1"))
// 		readOnlyMap, err := vars.ReadOnlyMapFrom(collection)
// 		testutils.NoError(t, err)
// 		b, err := json.Marshal(readOnlyMap)
// 		testutils.NoError(t, err)
// 		testutils.True(t, bytes.Equal(b, []byte(`{"key.v1":"value1","key.v2":"value2","other_key.v1":"other_value1"}`)))
// 	})
// }

// func TestReadOnlyMap_UnmarshalJSON(t *testing.T) {
// 	t.Run("empty", func(t *testing.T) {
// 		var readOnlyMap vars.ReadOnlyMap
// 		err := json.Unmarshal([]byte("null"), &readOnlyMap)
// 		testutils.NoError(t, err)
// 		testutils.Equal(t, 0, readOnlyMap.Len())
// 	})

// 	t.Run("valid", func(t *testing.T) {
// 		var readOnlyMap vars.ReadOnlyMap
// 		err := json.Unmarshal([]byte(`{"key.v1":"value1","key.v2":"value2","other_key.v1":"other_value1"}`), &readOnlyMap)
// 		testutils.NoError(t, err)
// 		testutils.Equal(t, 3, readOnlyMap.Len())
// 		testutils.Equal(t, "value1", readOnlyMap.Get("key.v1").String())
// 		testutils.Equal(t, "value2", readOnlyMap.Get("key.v2").String())
// 		testutils.Equal(t, "other_value1", readOnlyMap.Get("other_key.v1").String())
// 	})
// }

func genStringTestBytes() []byte {
	var out []byte
	for _, data := range getStringTests() {
		line := fmt.Sprintf(`%s=%s`+"\n", data.Key, data.Val)
		out = append(out, []byte(line)...)
	}
	// add empty line
	out = append(out, []byte("")...)
	return out
}

func newUnsafeValue(val any) vars.Value {
	v, _ := vars.NewValue(val)
	return v
}
