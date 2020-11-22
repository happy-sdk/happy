package vars

import (
	"strconv"
	"testing"
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

func TestValueParseInt64(t *testing.T) {
	val := Value("200")
	iout, erri1 := val.AsInt()
	if iout != 200 {
		t.Errorf("Value(11).AsInt() = %d, err(%v) want 200", iout, erri1)
	}

	val2 := Value("x")
	iout2, erri2 := val2.AsInt()
	if iout2 != 0 || erri2 == nil {
		t.Errorf("Value(11).AsInt() = %d, err(%v) want 0 and err", iout2, erri2)
	}
}

func TestValueTypeFromString(t *testing.T) {
	val := NewValue("string var")
	val2 := Value("string var")
	if val != val2 {
		t.Errorf("want: ValueFromString == Value got: ValueFromString = %q, val2 = %q", val, val2)
	}
}

func TestValueTypeAsBool(t *testing.T) {
	tests := []struct {
		val  Value
		want bool
	}{
		{NewValue("true"), true},
		{NewValue("false"), false},
	}
	for _, tt := range tests {
		got, _ := tt.val.Bool()
		if got != tt.want {
			t.Errorf("want: %t got %t", got, tt.want)
		}
	}
}

func TestValueTypeFromBool(t *testing.T) {
	tests := []struct {
		val  Value
		want string
	}{
		{ValueFromBool(true), "true"},
		{ValueFromBool(false), "false"},
	}
	for _, tt := range tests {
		if tt.val.String() != tt.want {
			t.Errorf("want: %q got %q", tt.val.String(), tt.want)
		}
	}
}

func TestValueTypeAsUint(t *testing.T) {
	tests := []struct {
		val  Value
		want uint
	}{
		{NewValue("1"), 1},
		{NewValue("2"), 2},
		{NewValue("2000000000000"), 2000000000000},
	}
	for _, tt := range tests {
		got, _ := tt.val.Uint(10, 0)
		if got != uint64(tt.want) {
			t.Errorf("want: %d got %d", got, tt.want)
		}
	}
}

func TestValueTypeAsUint8(t *testing.T) {
	tests := []struct {
		val  Value
		want uint8
	}{
		{NewValue("1"), 1},
		{NewValue("2"), 2},
		{NewValue("200"), 200},
	}
	for _, tt := range tests {
		got, _ := tt.val.Uint(10, 0)
		if got != uint64(tt.want) {
			t.Errorf("want: %d got %d", got, tt.want)
		}
	}
}

func TestValueTypeAsByte(t *testing.T) {
	tests := []struct {
		val  Value
		want byte
	}{
		{NewValue("1"), 1},
		{NewValue("2"), 2},
		{NewValue("200"), 200},
	}
	for _, tt := range tests {
		got, _ := tt.val.Uint(10, 0)
		if got != uint64(tt.want) {
			t.Errorf("want: %d got %d", got, tt.want)
		}
	}
}

func TestValueTypeAsInt(t *testing.T) {
	switch strconv.IntSize {
	case 32:
		collection := ParseFromBytes(genAtoi32TestBytes())
		for _, test := range atoi32tests {
			val := collection.Get(test.key)
			out, err := val.Int(10, 0)
			if test.wantErr != nil {
				if err == nil {
					t.Errorf("Value(%s).ParseInt(10, 0) = %v, err(%s) want %v, err(%s)",
						test.key, out, err, test.want, test.wantErr)
				} else {
					if test.wantErr != err.(*strconv.NumError).Err {
						t.Errorf("Value(%s).ParseInt(10, 0)= %v, err(%s) want %v, err(%s)",
							test.key, out, err, test.want, test.wantErr)
					}
				}
			}
			if int32(out) != test.want {
				t.Errorf("Value(%s).ParseInt(10, 64) = %v, err(%s) want %v, err(%s)",
					test.key, out, err, test.want, test.wantErr)
			}
		}
	case 64:
		collection := ParseFromBytes(genAtoi64TestBytes())
		for _, test := range atoi64Tests {
			val := collection.Get(test.key)
			out, err := val.Int(10, 64)
			if test.wantErr != nil {
				if err == nil {
					t.Errorf("Value(%s).ParseInt(10, 64) = %v, err(%s) want %v, err(%s)",
						test.key, out, err, test.want, test.wantErr)
				} else {
					if test.wantErr != err.(*strconv.NumError).Err {
						t.Errorf("Value(%s).ParseInt(10, 64) = %v, err(%s) want %v, err(%s)",
							test.key, out, err, test.want, test.wantErr)
					}
				}
			}
			if int64(out) != test.want {
				t.Errorf("Value(%s).ParseInt(10, 64) = %v, err(%s) want %v, err(%s)",
					test.key, out, err, test.want, test.wantErr)
			}
		}
	}
}

func TestValueParseAsComplex64(t *testing.T) {
	collection := ParseFromBytes(genComplex64TestBytes())
	for _, test := range complex64Tests {
		val := collection.Get(test.key)
		out, err := val.Complex64()
		if test.wantErr != nil {
			if err == nil {
				t.Errorf("Value(%s).ParseComplex64() = %v, err(%s) want %v, err(%s)",
					test.key, out, err, test.want, test.wantErr)
			}
		}
		if out != test.want {
			t.Errorf("Value(%s).ParseComplex64() = %v, err(%s) want %v, err(%s)",
				test.key, out, err, test.want, test.wantErr)
		}
	}
}

func TestValueAsUint16(t *testing.T) {
	tests := []struct {
		val  Value
		want uint16
	}{
		{NewValue("1"), 1},
		{NewValue("2"), 2},
		{NewValue("20000"), 20000},
	}
	for _, tt := range tests {
		got, _ := tt.val.Uint(10, 16)
		if got != uint64(tt.want) {
			t.Errorf("want: %d got %d", got, tt.want)
		}
	}
}

func TestValueAsUint32(t *testing.T) {
	tests := []struct {
		val  Value
		want uint32
	}{
		{NewValue("1"), 1},
		{NewValue("2"), 2},
		{NewValue("2000000000"), 2000000000},
	}
	for _, tt := range tests {
		got, _ := tt.val.Uint(10, 32)
		if got != uint64(tt.want) {
			t.Errorf("want: %d got %d", got, tt.want)
		}
	}
}

func TestValueAsUint64(t *testing.T) {
	tests := []struct {
		val  Value
		want uint16
	}{
		{NewValue("1"), 1},
		{NewValue("2"), 2},
		{NewValue("20000"), 20000},
	}
	for _, tt := range tests {
		got, _ := tt.val.Uint(10, 16)
		if got != uint64(tt.want) {
			t.Errorf("want: %d got %d", got, tt.want)
		}
	}
}

func TestValueAsUintptr(t *testing.T) {
	tests := []struct {
		val  Value
		want uintptr
	}{
		{NewValue("1"), 1},
		{NewValue("2"), 2},
		{NewValue("9000000000000000000"), 9000000000000000000},
	}
	for _, tt := range tests {
		got, _ := tt.val.Uintptr()
		if uintptr(got) != uintptr(tt.want) {
			t.Errorf("want: %d got %d", got, tt.want)
		}
	}
}
