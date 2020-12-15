// Copyright 2012 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package vars

import (
	"io/ioutil"
	"math"
	"strings"
	"testing"
)

func TestCollectionParseFromString(t *testing.T) {
	slice := strings.Split(string(genStringTestBytes()), "\n")
	collection := ParseKeyValSlice(slice)
	for _, test := range stringTests {
		if actual := collection.Get(test.key); actual.String() != test.val {
			t.Errorf("Collection.Get(%q) = %q, want %q", test.key, actual.String(), test.val)
		}
	}

	collection2 := ParseKeyValSlice([]string{"X"})
	if actual := collection2.Get("x"); actual.String() != "" {
		t.Errorf("Collection.Get(\"X\") = %q, want \"\"", actual.String())
	}
}

func TestCollectionParseFromBytes(t *testing.T) {
	collection := ParseFromBytes(genStringTestBytes())
	for _, test := range stringTests {
		if actual := collection.Get(test.key); actual.String() != test.val {
			t.Errorf("Collection.Get(%q) = %q, want %q", test.key, actual.String(), test.val)
		}
	}
}

func TestCollectionGetOrDefaultTo(t *testing.T) {
	collection := ParseFromBytes([]byte{})
	tests := []struct {
		k      string
		defVal string
		want   string
	}{
		{"STRING_1", "some-string", "some-string"},
		{"STRING_2", "some-string with space ", "some-string with space "},
		{"STRING_3", " some-string with space", " some-string with space"},
		{"STRING_4", "1234567", "1234567"},
		{"", "1234567", "1234567"},
	}
	for _, tt := range tests {
		if actual := collection.Get(tt.k, tt.defVal); actual.String() != tt.want {
			t.Errorf("Collection.GetOrDefaultTo(%q, %q) = %q, want %q", tt.k, tt.defVal, actual, tt.want)
		}
	}
}

func TestCollectionGetWithPrefix(t *testing.T) {
	collection := ParseFromBytes(genStringTestBytes())
	p := collection.GetWithPrefix("CGO")
	if len(p) != 6 {
		t.Errorf("Collection.GetsWithPrefix(\"CGO\") = %d, want (6)", len(p))
	}
}

func TestCollectionParseBool(t *testing.T) {
	collection := ParseFromBytes(genAtobTestBytes())
	for _, test := range boolTests {
		val := collection.Get(test.key)
		b := val.Bool()
		_, err := NewTyped(test.key, val.String(), TypeBool)
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

func TestCollectionParseFloat(t *testing.T) {
	collection := ParseFromBytes(genAtofTestBytes())
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

func TestCollectionParseFloat32(t *testing.T) {
	collection := ParseFromBytes(genAtof32TestBytes())
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

func TestCollectionParseUint64(t *testing.T) {
	collection := ParseFromBytes(genAtoui64TestBytes())
	for _, test := range uintTests {
		val := collection.Get(test.key)
		out := val.Uint64()
		if out != test.uint64 {
			t.Errorf("2. Value(%s).Uint64() = %v) want %v",
				test.key, out, test.uint64)
		}
	}
}

func TestCollectionParseInt64(t *testing.T) {
	collection := ParseFromBytes(genAtoi64TestBytes())
	for _, test := range intTests {
		val := collection.Get(test.key)
		out := val.Int64()
		if out != test.int64 {
			t.Errorf("2. Value(%s).Int64() = %v) want %v",
				test.key, out, test.int64)
		}
	}
}

func TestCollectionParseFields(t *testing.T) {
	collection := ParseFromBytes([]byte{})
	tests := []struct {
		k       string
		defVal  string
		wantLen int
	}{
		{"STRING", "one two", 2},
		{"STRING", "one two three four ", 4},
		{"STRING", " one two three four ", 4},
		{"STRING", "1 2 3 4 5 6 7 8.1", 8},
	}
	for _, tt := range tests {
		val := collection.Get(tt.k, tt.defVal)
		actual := len(val.ParseFields())
		if actual != tt.wantLen {
			t.Errorf("Value.(%q).ParseFields() len = %d, want %d", tt.k, actual, tt.wantLen)
		}
	}
}

func TestCollectionSet(t *testing.T) {
	collection := NewCollection()
	collection.Set("STRING", "collection")
	if val := collection.Get("STRING"); val != "collection" {
		t.Errorf("expected collection but got %s", val)
	}
}

func TestCollectionEnvFile(t *testing.T) {
	content, err := ioutil.ReadFile("testdata/dot_env")
	if err != nil {
		t.Error(err)
	}
	collection := ParseFromBytes(content)
	if val := collection.Get("GOARCH"); val != "amd64" {
		t.Errorf("expected GOARCH to equal amd64 got %s", val)
	}
}
