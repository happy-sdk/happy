package color

import (
	"testing"
)

func TestColorFormatting(t *testing.T) {
	var tests = []struct {
		Color    string
		Str      string
		Expected string
	}{
		{"red", "foo", "\x1b[31mfoo\x1b[0m"},
		{"green", "foo", "\x1b[32mfoo\x1b[0m"},
		{"white", "foo", "\x1b[1;37mfoo\x1b[0m"},
		{"gray", "foo", "\x1b[0;37mfoo\x1b[0m"},
		{"darkgray", "foo", "\x1b[0;30mfoo\x1b[0m"},
		{"yellow", "foo", "\x1b[33mfoo\x1b[0m"},
		{"cyan", "foo", "\x1b[36mfoo\x1b[0m"},
		{"blue", "foo", "\x1b[34mfoo\x1b[0m"},
		{"black", "foo", "\x1b[30mfoo\x1b[0m"},
		{"", "foo", "\x1b[1;37mfoo\x1b[0m"},
	}
	for _, test := range tests {
		if res := Text(test.Color, test.Str); res != test.Expected {
			t.Errorf("expected color string(%s) got(%s)", test.Expected, res)
		}
	}
}
