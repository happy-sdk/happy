package vars

import "testing"

func TestNewValue(t *testing.T) {
	var tests = []struct {
		val  interface{}
		want string
	}{
		{nil, ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := NewValue(tt.val).String()
		if got != tt.want {
			t.Errorf("want: %s got %s", tt.want, got)
		}
	}
}

func TestValueFromString(t *testing.T) {
	tests := []struct {
		name string
		val  string
		want string
	}{
		{"STRING", "some-string", "some-string"},
		{"STRING", "some-string with space ", "some-string with space"},
		{"STRING", " some-string with space", "some-string with space"},
		{"STRING", "1234567", "1234567"},
	}
	for _, tt := range tests {
		if got := NewValue(tt.val); got.String() != tt.want {
			t.Errorf("ValueFromString() = %q, want %q", got.String(), tt.want)
		}
		if rv := NewValue(tt.val); string(rv.Rune()) != tt.want {
			t.Errorf("Value.Rune() = %q, want %q", string(rv.Rune()), tt.want)
		}
	}
}
