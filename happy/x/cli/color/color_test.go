package color

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestColorFormatting(t *testing.T) {
	assert.Equal(t, "\x1b[31mfoo\x1b[0m", Text("red", "foo"))
	assert.Equal(t, "\x1b[32mfoo\x1b[0m", Text("green", "foo"))
	assert.Equal(t, "\x1b[1;37mfoo\x1b[0m", Text("white", "foo"))
	assert.Equal(t, "\x1b[0;37mfoo\x1b[0m", Text("gray", "foo"))
	assert.Equal(t, "\x1b[0;30mfoo\x1b[0m", Text("darkgray", "foo"))
	assert.Equal(t, "\x1b[33mfoo\x1b[0m", Text("yellow", "foo"))
	assert.Equal(t, "\x1b[36mfoo\x1b[0m", Text("cyan", "foo"))
	assert.Equal(t, "\x1b[34mfoo\x1b[0m", Text("blue", "foo"))
	assert.Equal(t, "\x1b[30mfoo\x1b[0m", Text("black", "foo"))
	assert.Equal(t, "\x1b[1;37mfoo\x1b[0m", Text("", "foo"))
}
