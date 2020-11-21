package vars

import (
	"strings"
	"testing"
)

func TestParseFromString(t *testing.T) {
	slice := strings.Split(string(genStringTestBytes()), "\n")
	collection := ParseKeyValSlice(slice)
	for _, test := range stringTests {
		if actual := collection.Get(test.key); actual.String() != test.want {
			t.Errorf("Collection.Get(%q) = %q, want %q", test.key, actual.String(), test.want)
		}
	}

	collection2 := ParseKeyValSlice([]string{"X"})
	if actual := collection2.Get("x"); actual.String() != "" {
		t.Errorf("Collection.Get(\"X\") = %q, want \"\"", actual.String())
	}
}

func TestParseFromBytes(t *testing.T) {
	collection := ParseFromBytes(genStringTestBytes())
	for _, test := range stringTests {
		if actual := collection.Get(test.key); actual.String() != test.want {
			t.Errorf("Collection.Get(%q) = %q, want %q", test.key, actual.String(), test.want)
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
		{"STRING", "some-string", "some-string"},
		{"STRING", "some-string with space ", "some-string with space"},
		{"STRING", " some-string with space", "some-string with space"},
		{"STRING", "1234567", "1234567"},
		{"", "1234567", "1234567"},
	}
	for _, tt := range tests {
		if actual := collection.GetOrDefaultTo(tt.k, tt.defVal); actual.String() != tt.want {
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
