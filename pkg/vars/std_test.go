// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package vars

import (
	"errors"
	"math"
	"testing"
)

func newDecimal(i uint64) *decimal {
	d := new(decimal)
	d.Assign(i)
	return d
}

type shiftTest struct {
	i     uint64
	shift int
	out   string
}

var shifttests = []shiftTest{
	{0, -100, "0"},
	{0, 100, "0"},
	{1, 100, "1267650600228229401496703205376"},
	{1, -100,
		"0.00000000000000000000000000000078886090522101180541" +
			"17285652827862296732064351090230047702789306640625",
	},
	{12345678, 8, "3160493568"},
	{12345678, -8, "48225.3046875"},
	{195312, 9, "99999744"},
	{1953125, 9, "1000000000"},
}

func TestDecimalShift(t *testing.T) {
	for i := 0; i < len(shifttests); i++ {
		test := &shifttests[i]
		d := newDecimal(test.i)
		d.Shift(test.shift)
		s := d.String()
		if s != test.out {
			t.Errorf("Decimal %v << %v = %v, want %v",
				test.i, test.shift, s, test.out)
		}
	}
}

type roundTest struct {
	i               uint64
	nd              int
	down, round, up string
	int             uint64
}

var roundtests = []roundTest{
	{0, 4, "0", "0", "0", 0},
	{12344999, 4, "12340000", "12340000", "12350000", 12340000},
	{12345000, 4, "12340000", "12340000", "12350000", 12340000},
	{12345001, 4, "12340000", "12350000", "12350000", 12350000},
	{23454999, 4, "23450000", "23450000", "23460000", 23450000},
	{23455000, 4, "23450000", "23460000", "23460000", 23460000},
	{23455001, 4, "23450000", "23460000", "23460000", 23460000},

	{99994999, 4, "99990000", "99990000", "100000000", 99990000},
	{99995000, 4, "99990000", "100000000", "100000000", 100000000},
	{99999999, 4, "99990000", "100000000", "100000000", 100000000},

	{12994999, 4, "12990000", "12990000", "13000000", 12990000},
	{12995000, 4, "12990000", "13000000", "13000000", 13000000},
	{12999999, 4, "12990000", "13000000", "13000000", 13000000},
}

func TestDecimalRound(t *testing.T) {
	for i := 0; i < len(roundtests); i++ {
		test := &roundtests[i]
		d := newDecimal(test.i)
		d.RoundDown(test.nd)
		s := d.String()
		if s != test.down {
			t.Errorf("Decimal %v RoundDown %d = %v, want %v",
				test.i, test.nd, s, test.down)
		}
		d = newDecimal(test.i)
		d.Round(test.nd)
		s = d.String()
		if s != test.round {
			t.Errorf("Decimal %v Round %d = %v, want %v",
				test.i, test.nd, s, test.down)
		}
		d = newDecimal(test.i)
		d.RoundUp(test.nd)
		s = d.String()
		if s != test.up {
			t.Errorf("Decimal %v RoundUp %d = %v, want %v",
				test.i, test.nd, s, test.up)
		}
	}
}

type roundIntTest struct {
	i     uint64
	shift int
	int   uint64
}

var roundinttests = []roundIntTest{
	{0, 100, 0},
	{512, -8, 2},
	{513, -8, 2},
	{640, -8, 2},
	{641, -8, 3},
	{384, -8, 2},
	{385, -8, 2},
	{383, -8, 1},
	{1, 100, 1<<64 - 1},
	{1000, 0, 1000},
}

func TestDecimalRoundedInteger(t *testing.T) {
	for i := 0; i < len(roundinttests); i++ {
		test := roundinttests[i]
		d := newDecimal(test.i)
		d.Shift(test.shift)
		int := d.RoundedInteger()
		if int != test.int {
			t.Errorf("Decimal %v >> %v RoundedInteger = %v, want %v",
				test.i, test.shift, int, test.int)
		}
	}
}

type entry = struct {
	nlz, ntz, pop int
}

// tab contains results for all uint8 values
var tab [256]entry

func init() {
	tab[0] = entry{8, 8, 0}
	for i := 1; i < len(tab); i++ {
		// nlz
		x := i // x != 0
		n := 0
		for x&0x80 == 0 {
			n++
			x <<= 1
		}
		tab[i].nlz = n

		// ntz
		x = i // x != 0
		n = 0
		for x&1 == 0 {
			n++
			x >>= 1
		}
		tab[i].ntz = n

		// pop
		x = i // x != 0
		n = 0
		for x != 0 {
			n += int(x & 1)
			x >>= 1
		}
		tab[i].pop = n
	}
}

func TestTrailingZeros(t *testing.T) {
	for i := 0; i < 256; i++ {
		ntz := tab[i].ntz
		for k := 0; k < 64-8; k++ {
			x := uint64(i) << uint(k)
			want := ntz + k

			if x <= 1<<32-1 {
				got := bitsTrailingZeros32(uint32(x))
				if x == 0 {
					want = 32
				}
				if got != want {
					t.Fatalf("TrailingZeros32(%#08x) == %d; want %d", x, got, want)
				}
				if uintSize == 32 {
					got = bitsTrailingZeros(uint(x))
					if got != want {
						t.Fatalf("TrailingZeros(%#08x) == %d; want %d", x, got, want)
					}
				}
			}

			if x <= 1<<64-1 {
				got := bitsTrailingZeros64(uint64(x))
				if x == 0 {
					want = 64
				}
				if got != want {
					t.Fatalf("TrailingZeros64(%#016x) == %d; want %d", x, got, want)
				}
				if uintSize == 64 {
					got = bitsTrailingZeros(uint(x))
					if got != want {
						t.Fatalf("TrailingZeros(%#016x) == %d; want %d", x, got, want)
					}
				}
			}
		}
	}
}
func TestTrailingZeros32(t *testing.T) {
	if IsHost32bit() {
		t.Skip("curretly testing on 32bit")
		return
	}
	SetHost32bit()
	defer RestoreHost32bit()

	for i := 0; i < 256; i++ {
		ntz := tab[i].ntz
		for k := 0; k < 64-8; k++ {
			x := uint64(i) << uint(k)
			want := ntz + k

			if x <= 1<<32-1 {
				got := bitsTrailingZeros32(uint32(x))
				if x == 0 {
					want = 32
				}
				if got != want {
					t.Fatalf("TrailingZeros32(%#08x) == %d; want %d", x, got, want)
				}
				if uintSize == 32 || host32bit {
					got = bitsTrailingZeros(uint(x))
					if got != want {
						t.Fatalf("TrailingZeros(%#08x) == %d; want %d", x, got, want)
					}
				}
			}

			if x <= 1<<64-1 {
				got := bitsTrailingZeros64(uint64(x))
				if x == 0 {
					want = 64
				}
				if got != want {
					t.Fatalf("TrailingZeros64(%#016x) == %d; want %d", x, got, want)
				}
			}
		}
	}
}

func TestSpecial(t *testing.T) {
	tests := []struct {
		s       string
		wantF   float64
		wantN   int
		wantOk  bool
		wantErr bool
	}{
		// Test cases go here
		{
			s:       "INFI",
			wantF:   math.Inf(1),
			wantN:   3,
			wantOk:  true,
			wantErr: false,
		},
	}

	for _, test := range tests {
		f, n, ok := special(test.s)
		if ok && test.wantErr {
			t.Errorf("special(%q) = %v, %v, %v; want error", test.s, f, n, ok)
		}
		if !ok && !test.wantErr {
			t.Errorf("special(%q) = %v, %v, %v; want success", test.s, f, n, ok)
		}
		if f != test.wantF {
			t.Errorf("special(%q) = %v, %v, %v; want %v", test.s, f, n, ok, test.wantF)
		}
		if n != test.wantN {
			t.Errorf("special(%q) = %v, %v, %v; want %v", test.s, f, n, ok, test.wantN)
		}
		if ok != test.wantOk {
			t.Errorf("special(%q) = %v, %v, %v; want %v", test.s, f, n, ok, test.wantOk)
		}
	}
}

func TestStdErrorCases(t *testing.T) {
	if unicodeIsControl(unicodeMaxLatin1 + 1) {
		t.Errorf("TestUnicodeIsControl expected false")
	}
	if r, s := utf8DecodeLastRuneInString(""); r != utf8RuneError || s != 0 {
		t.Errorf("utf8DecodeLastRuneInString expected utf8RuneError and size 0, got %d, %d", r, s)
	}
}

func TestFormatBits(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic due to invalid base")
		}
	}()
	formatBits([]byte{}, 0, 0, false, false)
}

type parseUint64Test struct {
	in  string
	out uint64
	err error
}

var parseUint64Tests = []parseUint64Test{
	{"", 0, ErrSyntax},
	{"0", 0, nil},
	{"1", 1, nil},
	{"12345", 12345, nil},
	{"012345", 12345, nil},
	{"12345x", 0, ErrSyntax},
	{"98765432100", 98765432100, nil},
	{"18446744073709551615", 1<<64 - 1, nil},
	{"18446744073709551616", 1<<64 - 1, ErrRange},
	{"18446744073709551620", 1<<64 - 1, ErrRange},
	{"1_2_3_4_5", 0, ErrSyntax}, // base=10 so no underscores allowed
	{"_12345", 0, ErrSyntax},
	{"1__2345", 0, ErrSyntax},
	{"12345_", 0, ErrSyntax},
	{"-0", 0, ErrSyntax},
	{"-1", 0, ErrSyntax},
	{"+1", 0, ErrSyntax},
}

func TestParseUint64(t *testing.T) {
	for i := range parseUint64Tests {
		test := &parseUint64Tests[i]
		out, err := strconvParseUint(test.in, 10, 64)
		if test.out != out || !errors.Is(err, test.err) {
			t.Errorf("ParseUint(%q, 10, 64) = %v, %v want %v, %v",
				test.in, out, err, test.out, test.err)
		}
		if test.err == nil {
			if _, err := strconvParseUint(test.in, -1, 64); err == nil {
				t.Errorf("expected incvalid base err when parsing %s", test.in)
			}

			if _, err := strconvParseUint(test.in, 10, 65); err == nil {
				t.Errorf("expected incvalid base err when parsing %s", test.in)
			}
		}
	}

}

type parseUint64BaseTest struct {
	in   string
	base int
	out  uint64
	err  error
}

var parseUint64BaseTests = []parseUint64BaseTest{
	{"", 0, 0, ErrSyntax},
	{"0", 0, 0, nil},
	{"0x", 0, 0, ErrSyntax},
	{"0X", 0, 0, ErrSyntax},
	{"1", 0, 1, nil},
	{"12345", 0, 12345, nil},
	{"012345", 0, 012345, nil},
	{"0x12345", 0, 0x12345, nil},
	{"0X12345", 0, 0x12345, nil},
	{"12345x", 0, 0, ErrSyntax},
	{"0xabcdefg123", 0, 0, ErrSyntax},
	{"123456789abc", 0, 0, ErrSyntax},
	{"98765432100", 0, 98765432100, nil},
	{"18446744073709551615", 0, 1<<64 - 1, nil},
	{"18446744073709551616", 0, 1<<64 - 1, ErrRange},
	{"18446744073709551620", 0, 1<<64 - 1, ErrRange},
	{"0xFFFFFFFFFFFFFFFF", 0, 1<<64 - 1, nil},
	{"0x10000000000000000", 0, 1<<64 - 1, ErrRange},
	{"01777777777777777777777", 0, 1<<64 - 1, nil},
	{"01777777777777777777778", 0, 0, ErrSyntax},
	{"02000000000000000000000", 0, 1<<64 - 1, ErrRange},
	{"0200000000000000000000", 0, 1 << 61, nil},
	{"0b", 0, 0, ErrSyntax},
	{"0B", 0, 0, ErrSyntax},
	{"0b101", 0, 5, nil},
	{"0B101", 0, 5, nil},
	{"0o", 0, 0, ErrSyntax},
	{"0O", 0, 0, ErrSyntax},
	{"0o377", 0, 255, nil},
	{"0O377", 0, 255, nil},

	// underscores allowed with base == 0 only
	{"1_2_3_4_5", 0, 12345, nil}, // base 0 => 10
	{"_12345", 0, 0, ErrSyntax},
	{"1__2345", 0, 0, ErrSyntax},
	{"12345_", 0, 0, ErrSyntax},

	{"1_2_3_4_5", 10, 0, ErrSyntax}, // base 10
	{"_12345", 10, 0, ErrSyntax},
	{"1__2345", 10, 0, ErrSyntax},
	{"12345_", 10, 0, ErrSyntax},

	{"0x_1_2_3_4_5", 0, 0x12345, nil}, // base 0 => 16
	{"_0x12345", 0, 0, ErrSyntax},
	{"0x__12345", 0, 0, ErrSyntax},
	{"0x1__2345", 0, 0, ErrSyntax},
	{"0x1234__5", 0, 0, ErrSyntax},
	{"0x12345_", 0, 0, ErrSyntax},

	{"1_2_3_4_5", 16, 0, ErrSyntax}, // base 16
	{"_12345", 16, 0, ErrSyntax},
	{"1__2345", 16, 0, ErrSyntax},
	{"1234__5", 16, 0, ErrSyntax},
	{"12345_", 16, 0, ErrSyntax},

	{"0_1_2_3_4_5", 0, 012345, nil}, // base 0 => 8 (0377)
	{"_012345", 0, 0, ErrSyntax},
	{"0__12345", 0, 0, ErrSyntax},
	{"01234__5", 0, 0, ErrSyntax},
	{"012345_", 0, 0, ErrSyntax},

	{"0o_1_2_3_4_5", 0, 012345, nil}, // base 0 => 8 (0o377)
	{"_0o12345", 0, 0, ErrSyntax},
	{"0o__12345", 0, 0, ErrSyntax},
	{"0o1234__5", 0, 0, ErrSyntax},
	{"0o12345_", 0, 0, ErrSyntax},

	{"0_1_2_3_4_5", 8, 0, ErrSyntax}, // base 8
	{"_012345", 8, 0, ErrSyntax},
	{"0__12345", 8, 0, ErrSyntax},
	{"01234__5", 8, 0, ErrSyntax},
	{"012345_", 8, 0, ErrSyntax},

	{"0b_1_0_1", 0, 5, nil}, // base 0 => 2 (0b101)
	{"_0b101", 0, 0, ErrSyntax},
	{"0b__101", 0, 0, ErrSyntax},
	{"0b1__01", 0, 0, ErrSyntax},
	{"0b10__1", 0, 0, ErrSyntax},
	{"0b101_", 0, 0, ErrSyntax},

	{"1_0_1", 2, 0, ErrSyntax}, // base 2
	{"_101", 2, 0, ErrSyntax},
	{"1_01", 2, 0, ErrSyntax},
	{"10_1", 2, 0, ErrSyntax},
	{"101_", 2, 0, ErrSyntax},
}

func TestParseUint64Base(t *testing.T) {
	for i := range parseUint64BaseTests {
		test := &parseUint64BaseTests[i]
		out, err := strconvParseUint(test.in, test.base, 64)
		if test.out != out || !errors.Is(err, test.err) {
			t.Errorf("ParseUint(%q, %v, 64) = %v, %v want %v, %v",
				test.in, test.base, out, err, test.out, test.err)
		}
	}
}
