package vars

import (
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("Testing with Ginkgo", func() {
	It("parse from strings", func() {
		slice := strings.Split(string(genStringTestBytes()), "\n")
		collection := ParseKeyValSlice(slice)
		for _, test := range stringTests {
			if actual := collection.Get(test.key); actual.String() != test.want {
				GinkgoT().Errorf("Collection.Get(%q) = %q, want %q", test.key, actual.String(), test.want)
			}
		}

		collection2 := ParseKeyValSlice([]string{"X"})
		if actual := collection2.Get("x"); actual.String() != "" {
			GinkgoT().Errorf("Collection.Get(\"X\") = %q, want \"\"", actual.String())
		}
	})

	It("parse from bytes", func() {
		collection := ParseFromBytes(genStringTestBytes())
		for _, test := range stringTests {
			if actual := collection.Get(test.key); actual.String() != test.want {
				GinkgoT().Errorf("Collection.Get(%q) = %q, want %q", test.key, actual.String(), test.want)
			}
		}
	})

	It("value from string", func() {
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
				GinkgoT().Errorf("ValueFromString() = %q, want %q", got.String(), tt.want)
			}
			if rv := NewValue(tt.val); string(rv.Rune()) != tt.want {
				GinkgoT().Errorf("Value.Rune() = %q, want %q", string(rv.Rune()), tt.want)
			}
		}
	})

	It("collection_ get or default to", func() {
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
				GinkgoT().Errorf("Collection.GetOrDefaultTo(%q, %q) = %q, want %q", tt.k, tt.defVal, actual, tt.want)
			}
		}
	})

	It("collection_ get with prefix", func() {
		collection := ParseFromBytes(genStringTestBytes())
		p := collection.GetWithPrefix("CGO")
		if len(p) != 6 {
			GinkgoT().Errorf("Collection.GetsWithPrefix(\"CGO\") = %d, want (6)", len(p))
		}
	})

	It("value_ parse bool", func() {
		collection := ParseFromBytes(genAtobTestBytes())
		for _, test := range atobTests {
			val := collection.Get(test.key)
			b, err := val.Bool()
			if test.wantErr != nil {
				if err == nil {
					GinkgoT().Errorf("Value(%s).ParseBool(): expected %s but got nil", test.key, test.wantErr)
				} else {
					if test.wantErr != err.(*strconv.NumError).Err {
						GinkgoT().Errorf("Value(%s).ParseBool(): expected %s but got %s", test.key, test.wantErr, err)
					}
				}
			} else {
				if err != nil {
					GinkgoT().Errorf("Value(%s).ParseBool(): expected no error but got %s", test.key, err)
				}
				if b != test.want {
					GinkgoT().Errorf("Value(%s).ParseBool(): = %t, want %t", test.key, b, test.want)
				}
			}
		}
	})

	It("value_ parse float", func() {
		collection := ParseFromBytes(genAtofTestBytes())
		for _, test := range atofTests {
			val := collection.Get(test.key)
			out, err := val.Float(64)
			outs := strconv.FormatFloat(out, 'g', -1, 64)
			if test.wantErr != nil {
				if err == nil {
					GinkgoT().Errorf("Value(%s).ParseFloat(64) = %v, err(%s) want %v, err(%s)",
						test.key, out, err, test.want, test.wantErr)
				} else {
					if test.wantErr != err.(*strconv.NumError).Err {
						GinkgoT().Errorf("Value(%s).ParseFloat(64) = %v, err(%s) want %v, err(%s)",
							test.key, out, err, test.want, test.wantErr)
					}
				}
			}
			if outs != test.want {
				GinkgoT().Errorf("Value(%s).ParseFloat(64) = %v, err(%s) want %v, err(%s)",
					test.key, out, err, test.want, test.wantErr)
			}

			if float64(float32(out)) == out {
				out, err := val.Float(32)
				out32 := float32(out)
				if float64(out32) != out {
					GinkgoT().Errorf("Value(%s).ParseFloat(32) = %v, not a float32 (closest is %v)", test.key, out, float64(out32))
					continue
				}
				outs := strconv.FormatFloat(float64(out32), 'g', -1, 32)
				if outs != test.want {
					GinkgoT().Errorf("Value(%s).ParseFloat(32) = %v, %s want %v, %s  # %v",
						test.key, out32, err, test.want, test.wantErr, out)
				}
			}
		}
	})

	It("value_ parse float32", func() {
		collection := ParseFromBytes(genAtof32TestBytes())
		for _, test := range atof32Tests {
			val := collection.Get(test.key)
			out, err := val.Float(32)
			out32 := float32(out)
			if float64(out32) != out {
				GinkgoT().Errorf("Value(%s).ParseFloat(32) = %v, not a float32 (closest is %v)",
					test.key, out, float64(out32))
				continue
			}
			outs := strconv.FormatFloat(float64(out32), 'g', -1, 32)
			if test.wantErr != nil {
				if err == nil {
					GinkgoT().Errorf("Value(%s).ParseFloat(32) = %v, err(%s) want %v, err(%s)",
						test.key, out, err, test.want, test.wantErr)
				} else {
					if test.wantErr != err.(*strconv.NumError).Err {
						GinkgoT().Errorf("Value(%s).ParseFloat(32) = %v, err(%s) want %v, err(%s)",
							test.key, out, err, test.want, test.wantErr)
					}
				}
			}
			if outs != test.want {
				GinkgoT().Errorf("Value(%s).ParseFloat(32) = %v, err(%s) want %v, err(%s)",
					test.key, out, err, test.want, test.wantErr)
			}
		}
	})

	It("value_ parse uint64", func() {
		collection := ParseFromBytes(genAtoui64TestBytes())
		for _, test := range atoui64Tests {
			val := collection.Get(test.key)
			out, err := val.Uint(10, 64)
			if test.wantErr != nil {
				if err == nil {
					GinkgoT().Errorf("Value(%s).ParseUint(10, 64) = %v, err(%s) want %v, err(%s)",
						test.key, out, err, test.want, test.wantErr)
				} else {
					if test.wantErr != err.(*strconv.NumError).Err {
						GinkgoT().Errorf("Value(%s).ParseUint(10, 64) = %v, err(%s) want %v, err(%s)",
							test.key, out, err, test.want, test.wantErr)
					}
				}
			}
			if out != test.want {
				GinkgoT().Errorf("Value(%s).ParseUint(10, 64) = %v, err(%s) want %v, err(%s)",
					test.key, out, err, test.want, test.wantErr)
			}
		}
	})

	It("value_ parse uint64 base", func() {
		collection := ParseFromBytes(genBtoui64TestBytes())
		for _, test := range btoui64Tests {
			val := collection.Get(test.key)
			out, err := val.Uint(0, 64)
			if test.wantErr != nil {
				if err == nil {
					GinkgoT().Errorf("Value(%s).ParseUint(0, 64) = %v, err(%s) want %v, err(%s)",
						test.key, out, err, test.want, test.wantErr)
				} else {
					if test.wantErr != err.(*strconv.NumError).Err {
						GinkgoT().Errorf("Value(%s).ParseUint(0, 64) = %v, err(%s) want %v, err(%s)",
							test.key, out, err, test.want, test.wantErr)
					}
				}
			}
			if out != test.want {
				GinkgoT().Errorf("Value(%s).ParseUint(0, 64) = %v, err(%s) want %v, err(%s)",
					test.key, out, err, test.want, test.wantErr)
			}
		}
	})

	It("value_ parse int64", func() {
		val := Value("200")
		iout, erri1 := val.AsInt()
		if iout != 200 {
			GinkgoT().Errorf("Value(11).AsInt() = %d, err(%v) want 200", iout, erri1)
		}

		val2 := Value("x")
		iout2, erri2 := val2.AsInt()
		if iout2 != 0 || erri2 == nil {
			GinkgoT().Errorf("Value(11).AsInt() = %d, err(%v) want 0 and err", iout2, erri2)
		}

		collection := ParseFromBytes(genAtoi64TestBytes())
		for _, test := range atoi64Tests {
			val := collection.Get(test.key)
			out, err := val.Int(10, 64)
			if test.wantErr != nil {
				if err == nil {
					GinkgoT().Errorf("Value(%s).ParseInt(10, 64) = %v, err(%s) want %v, err(%s)",
						test.key, out, err, test.want, test.wantErr)
				} else {
					if test.wantErr != err.(*strconv.NumError).Err {
						GinkgoT().Errorf("Value(%s).ParseInt(10, 64) = %v, err(%s) want %v, err(%s)",
							test.key, out, err, test.want, test.wantErr)
					}
				}
			}
			if out != test.want {
				GinkgoT().Errorf("Value(%s).ParseInt(10, 64) = %v, err(%s) want %v, err(%s)",
					test.key, out, err, test.want, test.wantErr)
			}
		}
	})

	It("value_ parse int64 base", func() {
		collection := ParseFromBytes(genBtoi64TestBytes())
		for _, test := range btoi64Tests {
			val := collection.Get(test.key)
			out, err := val.Int(test.base, 64)
			if test.wantErr != nil {
				if err == nil {
					GinkgoT().Errorf("Value(%s).ParseInt(%d, 64) = %v, err(%v) want %v, err(%v)",
						test.key, test.base, out, err, test.want, test.wantErr)
				} else {
					if test.wantErr != err.(*strconv.NumError).Err {
						GinkgoT().Errorf("Value(%s).ParseInt(%d, 64) = %v, err(%v) want %v, err(%v)",
							test.key, test.base, out, err, test.want, test.wantErr)
					}
				}
			}

			if out != test.want {
				GinkgoT().Errorf("Value(%s).ParseInt(%d, 64) = %v, err(%v) want %v, err(%v)",
					test.key, test.base, out, err, test.want, test.wantErr)
			}
		}
	})

	It("value_ test parse uint", func() {
		switch strconv.IntSize {
		case 32:
			collection := ParseFromBytes(genAtoui32TestBytes())
			for _, test := range atoui32Tests {
				val := collection.Get(test.key)
				out, err := val.Uint(10, 0)
				if test.wantErr != nil {
					if err == nil {
						GinkgoT().Errorf("Value(%s).ParseUint(10, 0) = %v, err(%s) want %v, err(%s)",
							test.key, out, err, test.want, test.wantErr)
					} else {
						if test.wantErr != err.(*strconv.NumError).Err {
							GinkgoT().Errorf("Value(%s).ParseUint(10, 0) = %v, err(%s) want %v, err(%s)",
								test.key, out, err, test.want, test.wantErr)
						}
					}
				}
				if uint32(out) != test.want {
					GinkgoT().Errorf("Value(%s).ParseUint(10, 0) = %v, err(%s) want %v, err(%s)",
						test.key, out, err, test.want, test.wantErr)
				}
			}
		case 64:
			collection := ParseFromBytes(genAtoui64TestBytes())
			for _, test := range atoui64Tests {
				val := collection.Get(test.key)
				out, err := val.Uint(10, 0)
				if test.wantErr != nil {
					if err == nil {
						GinkgoT().Errorf("Value(%s).ParseUint(10, 0) = %v, err(%s) want %v, err(%s)",
							test.key, out, err, test.want, test.wantErr)
					} else {
						if test.wantErr != err.(*strconv.NumError).Err {
							GinkgoT().Errorf("Value(%s).ParseUint(10, 0) = %v, err(%s) want %v, err(%s)",
								test.key, out, err, test.want, test.wantErr)
						}
					}
				}
				if uint64(out) != test.want {
					GinkgoT().Errorf("Value(%s).ParseUint(10, 0) = %v, err(%s) want %v, err(%s)",
						test.key, out, err, test.want, test.wantErr)
				}
			}
		}
	})

	It("value_ test parse int", func() {
		switch strconv.IntSize {
		case 32:
			collection := ParseFromBytes(genAtoi32TestBytes())
			for _, test := range atoi32tests {
				val := collection.Get(test.key)
				out, err := val.Int(10, 0)
				if test.wantErr != nil {
					if err == nil {
						GinkgoT().Errorf("Value(%s).ParseInt(10, 0) = %v, err(%s) want %v, err(%s)",
							test.key, out, err, test.want, test.wantErr)
					} else {
						if test.wantErr != err.(*strconv.NumError).Err {
							GinkgoT().Errorf("Value(%s).ParseInt(10, 0)= %v, err(%s) want %v, err(%s)",
								test.key, out, err, test.want, test.wantErr)
						}
					}
				}
				if int32(out) != test.want {
					GinkgoT().Errorf("Value(%s).ParseInt(10, 64) = %v, err(%s) want %v, err(%s)",
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
						GinkgoT().Errorf("Value(%s).ParseInt(10, 64) = %v, err(%s) want %v, err(%s)",
							test.key, out, err, test.want, test.wantErr)
					} else {
						if test.wantErr != err.(*strconv.NumError).Err {
							GinkgoT().Errorf("Value(%s).ParseInt(10, 64) = %v, err(%s) want %v, err(%s)",
								test.key, out, err, test.want, test.wantErr)
						}
					}
				}
				if int64(out) != test.want {
					GinkgoT().Errorf("Value(%s).ParseInt(10, 64) = %v, err(%s) want %v, err(%s)",
						test.key, out, err, test.want, test.wantErr)
				}
			}
		}
	})

	It("value_ parse fields", func() {
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
				GinkgoT().Errorf("Value.(%q).ParseFields() len = %d, want %d", tt.k, actual, tt.wantLen)
			}
		}
	})

	It("value_ parse complex64", func() {
		collection := ParseFromBytes(genComplex64TestBytes())
		for _, test := range complex64Tests {
			val := collection.Get(test.key)
			out, err := val.Complex64()
			if test.wantErr != nil {
				if err == nil {
					GinkgoT().Errorf("Value(%s).ParseComplex64() = %v, err(%s) want %v, err(%s)",
						test.key, out, err, test.want, test.wantErr)
				}
			}
			if out != test.want {
				GinkgoT().Errorf("Value(%s).ParseComplex64() = %v, err(%s) want %v, err(%s)",
					test.key, out, err, test.want, test.wantErr)
			}
		}
	})

	It("value_ parse complex128", func() {
		collection := ParseFromBytes(genComplex128TestBytes())
		for _, test := range complex128Tests {
			val := collection.Get(test.key)
			out, err := val.Complex128()
			if test.wantErr != nil {
				if err == nil {
					GinkgoT().Errorf("Value(%s).ParseComplex128() = %v, err(%s) want %v, err(%s)",
						test.key, out, err, test.want, test.wantErr)
				}
			}

			if out != test.want {
				GinkgoT().Errorf("Value(%s).ParseComplex128() = %v, err(%s) want %v, err(%s)",
					test.key, out, err, test.want, test.wantErr)
			}
		}
	})

	It("value_ len", func() {
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
				GinkgoT().Errorf("Value.(%q).Len() len = %d, want %d", tt.k, actual, tt.wantLen)
			}
			if tt.defVal == "" && !val.Empty() {
				GinkgoT().Errorf("Value.(%q).Empty() = %t for value(%q), want true", tt.k, val.Empty(), val.String())
			}
			if tt.defVal != "" && val.Empty() {
				GinkgoT().Errorf("Value.(%q).Empty() = %t for value(%q), want true", tt.k, val.Empty(), val.String())
			}
		}
	})

	It("parse from string", func() {
		key, val := ParseKeyVal("X=1")
		if key != "X" {
			GinkgoT().Errorf("Key should be X got %q", key)
		}
		if val.Empty() {
			GinkgoT().Error("Val should be 1")
		}
		if i, err := val.Int(0, 10); i != 1 || err != nil {
			GinkgoT().Error("ParseInt should be 1")
		}
	})

	It("parse key val empty", func() {
		ek, ev := ParseKeyVal("")
		if ek != "" || ev != "" {
			GinkgoT().Errorf("TestParseKeyValEmpty(\"\") = %q=%q, want ", ek, ev)
		}
	})

	It("parse key val empty val", func() {
		key, val := ParseKeyVal("X")
		if key != "X" {
			GinkgoT().Errorf("Key should be X got %q", key)
		}
		if !val.Empty() {
			GinkgoT().Error("Val should be empty")
		}
	})
})
