package testutils

import (
	"errors"
	"testing"
)

func TestShouldSucceed(t *testing.T) {
	var testErr = errors.New("test error")
	True(t, true, "ecpected true")
	False(t, false, "ecpected false")
	NoError(t, nil, "ecpected no error")
	ErrorIs(t, testErr, testErr, "ecpected error to be testErr")
	Equal(t, 1, 1)
	Equal(t, true, true)
	Equal(t, "nil", "nil")
}
