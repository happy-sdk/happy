package vars_test

import (
	"testing"

	"github.com/howi-lib/vars/v3"
)

func TestNewValue(t *testing.T) {
	var tests = []struct {
		val  interface{}
		want string
	}{
		{nil, ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := vars.NewValue(tt.val).String()
		if got != tt.want {
			t.Errorf("want: %s got %s", tt.want, got)
		}
	}
}
