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

func TestValueTypeAsRune(t *testing.T) {
	tests := []struct {
		val  Value
		want rune
	}{
		{NewValue("1"), 1},
		{NewValue("2"), 2},
		{NewValue("444434555"), 444434555},
	}
	for _, tt := range tests {
		got, _ := tt.val.Int(10, 0)
		if got != int64(tt.want) {
			t.Errorf("want: %d got %d", got, tt.want)
		}
	}
}

func TestValueTypeAsInt64(t *testing.T) {
	tests := []struct {
		val  Value
		want int64
	}{
		{NewValue("1"), 1},
		{NewValue("2"), 2},
		{NewValue("4444447777777834555"), 4444447777777834555},
	}
	for _, tt := range tests {
		got, _ := tt.val.Int(10, 0)
		if got != int64(tt.want) {
			t.Errorf("want: %d got %d", got, tt.want)
		}
	}
}

func TestValueTypeAsFloat32(t *testing.T) {
	tests := []struct {
		val  Value
		want float32
	}{
		{NewValue("1"), 1},
		{NewValue("2"), 2},
		{NewValue("4444447777777834555"), 4444447777777834555},
	}
	for _, tt := range tests {
		got, _ := tt.val.Float(32)
		if float32(got) != tt.want {
			t.Errorf("want: %f got %f", got, tt.want)
		}
	}
}

func TestValueTypeAsFloat64(t *testing.T) {
	tests := []struct {
		val  Value
		want float64
	}{
		{NewValue("1"), 1},
		{NewValue("2"), 2},
		{NewValue("443444777777834555"), 443444777777834555},
	}
	for _, tt := range tests {
		got, _ := tt.val.Float(64)
		if got != tt.want {
			t.Errorf("want: %f got %f", got, tt.want)
		}
	}
}

func TestValueTypeAsComplex64(t *testing.T) {
	tests := []struct {
		val  Value
		want complex64
	}{
		{NewValue("1.000000059604644775390626 2"), complex64(complex(1.0000001, 2))},
		{NewValue("1x -0"), complex64(0)},
	}
	for _, tt := range tests {
		got, _ := tt.val.Complex64()
		if got != tt.want {
			t.Errorf("want: %f got %f", got, tt.want)
		}
	}
}
func TestValueTypeAsComplex128(t *testing.T) {
	tests := []struct {
		val  Value
		want complex128
	}{
		{NewValue("123456700 1e-100"), complex(1.234567e+08, 1e-100)},
		{NewValue("99999999999999974834176 100000000000000000000001"), complex128(complex(9.999999999999997e+22, 1.0000000000000001e+23))},
		{NewValue("100000000000000008388608 100000000000000016777215"), complex128(complex(1.0000000000000001e+23, 1.0000000000000001e+23))},
		{NewValue("1e-20 625e-3"), complex128(complex(1e-20, 0.625))},
	}
	for _, tt := range tests {
		got, _ := tt.val.Complex128()
		if got != tt.want {
			t.Errorf("want: %f got %f", got, tt.want)
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

func TestValueAsInt(t *testing.T) {
	tests := []struct {
		val  Value
		want int
	}{
		{NewValue("1"), 1},
		{NewValue("2"), 2},
		{NewValue("444444444444"), 444444444444},
	}
	for _, tt := range tests {
		got, _ := tt.val.Int(10, 0)
		if got != int64(tt.want) {
			t.Errorf("want: %d got %d", got, tt.want)
		}
	}
}

func TestValueAsInt8(t *testing.T) {
	tests := []struct {
		val  Value
		want int8
	}{
		{NewValue("1"), 1},
		{NewValue("2"), 2},
		{NewValue("44"), 44},
	}
	for _, tt := range tests {
		got, _ := tt.val.Int(10, 0)
		if got != int64(tt.want) {
			t.Errorf("want: %d got %d", got, tt.want)
		}
	}
}

func TestValueAsInt16(t *testing.T) {
	tests := []struct {
		val  Value
		want int16
	}{
		{NewValue("1"), 1},
		{NewValue("2"), 2},
		{NewValue("4444"), 4444},
	}
	for _, tt := range tests {
		got, _ := tt.val.Int(10, 0)
		if got != int64(tt.want) {
			t.Errorf("want: %d got %d", got, tt.want)
		}
	}
}

func TestValueAsInt32(t *testing.T) {
	tests := []struct {
		val  Value
		want int32
	}{
		{NewValue("1"), 1},
		{NewValue("2"), 2},
		{NewValue("444434555"), 444434555},
	}
	for _, tt := range tests {
		got, _ := tt.val.Int(10, 0)
		if got != int64(tt.want) {
			t.Errorf("want: %d got %d", got, tt.want)
		}
	}
}

func TestValueAsInt64(t *testing.T) {
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

func TestValueAsIntMulti(t *testing.T) {
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

func TestValueParseAsFields(t *testing.T) {
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
		val := collection.GetOrDefaultTo(tt.k, tt.defVal)
		actual := len(val.ParseFields())
		if actual != tt.wantLen {
			t.Errorf("Value.(%q).ParseFields() len = %d, want %d", tt.k, actual, tt.wantLen)
		}
	}
}

func TestValueParseAsComplex128(t *testing.T) {
	collection := ParseFromBytes(genComplex128TestBytes())
	for _, test := range complex128Tests {
		val := collection.Get(test.key)
		out, err := val.Complex128()
		if test.wantErr != nil {
			if err == nil {
				t.Errorf("Value(%s).ParseComplex128() = %v, err(%s) want %v, err(%s)",
					test.key, out, err, test.want, test.wantErr)
			}
		}

		if out != test.want {
			t.Errorf("Value(%s).ParseComplex128() = %v, err(%s) want %v, err(%s)",
				test.key, out, err, test.want, test.wantErr)
		}
	}
}

func TestValueLen(t *testing.T) {
	collection := ParseKeyValSlice([]string{})
	tests := []struct {
		k       string
		defVal  string
		wantLen int
	}{
		{"STRING", "one two", 2},
		{"STRING", "one two three four ", 4},
		{"STRING", " one two three four ", 4},
		{"STRING", "1 2 3 4 5 6 7 8.1", 8},
		{"STRING", "", 0},
	}
	for _, tt := range tests {
		val := collection.GetOrDefaultTo(tt.k, tt.defVal)
		actual := len(val.String())
		if actual != val.Len() {
			t.Errorf("Value.(%q).Len() len = %d, want %d", tt.k, actual, tt.wantLen)
		}
		if tt.defVal == "" && !val.Empty() {
			t.Errorf("Value.(%q).Empty() = %t for value(%q), want true", tt.k, val.Empty(), val.String())
		}
		if tt.defVal != "" && val.Empty() {
			t.Errorf("Value.(%q).Empty() = %t for value(%q), want true", tt.k, val.Empty(), val.String())
		}
	}
}

func TestParseFromString(t *testing.T) {
	key, val := ParseKeyVal("X=1")
	if key != "X" {
		t.Errorf("Key should be X got %q", key)
	}
	if val.Empty() {
		t.Error("Val should be 1")
	}
	if i, err := val.Int(0, 10); i != 1 || err != nil {
		t.Error("ParseInt should be 1")
	}
}

func TestParseFromEmpty(t *testing.T) {

	if ek, ev := ParseKeyVal(""); ek != "" || ev != "" {
		t.Errorf("TestParseKeyValEmpty(\"\") = %q=%q, want ", ek, ev)
	}
	key, val := ParseKeyVal("X")
	if key != "X" {
		t.Errorf("Key should be X got %q", key)
	}
	if !val.Empty() {
		t.Error("Val should be empty")
	}
}
