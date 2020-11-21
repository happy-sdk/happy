package vars

import (
	"strconv"
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

func TestCollectionParseBool(t *testing.T) {
	collection := ParseFromBytes(genAtobTestBytes())
	for _, test := range atobTests {
		val := collection.Get(test.key)
		b, err := val.Bool()
		if test.wantErr != nil {
			if err == nil {
				t.Errorf("Value(%s).ParseBool(): expected %s but got nil", test.key, test.wantErr)
			} else {
				if test.wantErr != err.(*strconv.NumError).Err {
					t.Errorf("Value(%s).ParseBool(): expected %s but got %s", test.key, test.wantErr, err)
				}
			}
		} else {
			if err != nil {
				t.Errorf("Value(%s).ParseBool(): expected no error but got %s", test.key, err)
			}
			if b != test.want {
				t.Errorf("Value(%s).ParseBool(): = %t, want %t", test.key, b, test.want)
			}
		}
	}
}

func TestCollectionParseFloat(t *testing.T) {
	collection := ParseFromBytes(genAtofTestBytes())
	for _, test := range atofTests {
		val := collection.Get(test.key)
		out, err := val.Float(64)
		outs := strconv.FormatFloat(out, 'g', -1, 64)
		if test.wantErr != nil {
			if err == nil {
				t.Errorf("Value(%s).ParseFloat(64) = %v, err(%s) want %v, err(%s)",
					test.key, out, err, test.want, test.wantErr)
			} else {
				if test.wantErr != err.(*strconv.NumError).Err {
					t.Errorf("Value(%s).ParseFloat(64) = %v, err(%s) want %v, err(%s)",
						test.key, out, err, test.want, test.wantErr)
				}
			}
		}
		if outs != test.want {
			t.Errorf("Value(%s).ParseFloat(64) = %v, err(%s) want %v, err(%s)",
				test.key, out, err, test.want, test.wantErr)
		}

		if float64(float32(out)) == out {
			out, err := val.Float(32)
			out32 := float32(out)
			if float64(out32) != out {
				t.Errorf("Value(%s).ParseFloat(32) = %v, not a float32 (closest is %v)", test.key, out, float64(out32))
				continue
			}
			outs := strconv.FormatFloat(float64(out32), 'g', -1, 32)
			if outs != test.want {
				t.Errorf("Value(%s).ParseFloat(32) = %v, %s want %v, %s  # %v",
					test.key, out32, err, test.want, test.wantErr, out)
			}
		}
	}

}

func TestCollectionParseFloat32(t *testing.T) {
	collection := ParseFromBytes(genAtof32TestBytes())
	for _, test := range atof32Tests {
		val := collection.Get(test.key)
		out, err := val.Float(32)
		out32 := float32(out)
		if float64(out32) != out {
			t.Errorf("Value(%s).ParseFloat(32) = %v, not a float32 (closest is %v)",
				test.key, out, float64(out32))
			continue
		}
		outs := strconv.FormatFloat(float64(out32), 'g', -1, 32)
		if test.wantErr != nil {
			if err == nil {
				t.Errorf("Value(%s).ParseFloat(32) = %v, err(%s) want %v, err(%s)",
					test.key, out, err, test.want, test.wantErr)
			} else {
				if test.wantErr != err.(*strconv.NumError).Err {
					t.Errorf("Value(%s).ParseFloat(32) = %v, err(%s) want %v, err(%s)",
						test.key, out, err, test.want, test.wantErr)
				}
			}
		}
		if outs != test.want {
			t.Errorf("Value(%s).ParseFloat(32) = %v, err(%s) want %v, err(%s)",
				test.key, out, err, test.want, test.wantErr)
		}
	}
}

func TestCollectionParseUint64(t *testing.T) {
	collection := ParseFromBytes(genAtoui64TestBytes())
	for _, test := range atoui64Tests {
		val := collection.Get(test.key)
		out, err := val.Uint(10, 64)
		if test.wantErr != nil {
			if err == nil {
				t.Errorf("Value(%s).ParseUint(10, 64) = %v, err(%s) want %v, err(%s)",
					test.key, out, err, test.want, test.wantErr)
			} else {
				if test.wantErr != err.(*strconv.NumError).Err {
					t.Errorf("Value(%s).ParseUint(10, 64) = %v, err(%s) want %v, err(%s)",
						test.key, out, err, test.want, test.wantErr)
				}
			}
		}
		if out != test.want {
			t.Errorf("Value(%s).ParseUint(10, 64) = %v, err(%s) want %v, err(%s)",
				test.key, out, err, test.want, test.wantErr)
		}
	}
}

func TestCollectionParseUint64Base(t *testing.T) {
	collection := ParseFromBytes(genBtoui64TestBytes())
	for _, test := range btoui64Tests {
		val := collection.Get(test.key)
		out, err := val.Uint(0, 64)
		if test.wantErr != nil {
			if err == nil {
				t.Errorf("Value(%s).ParseUint(0, 64) = %v, err(%s) want %v, err(%s)",
					test.key, out, err, test.want, test.wantErr)
			} else {
				if test.wantErr != err.(*strconv.NumError).Err {
					t.Errorf("Value(%s).ParseUint(0, 64) = %v, err(%s) want %v, err(%s)",
						test.key, out, err, test.want, test.wantErr)
				}
			}
		}
		if out != test.want {
			t.Errorf("Value(%s).ParseUint(0, 64) = %v, err(%s) want %v, err(%s)",
				test.key, out, err, test.want, test.wantErr)
		}
	}
}

func TestCollectionParseInt64(t *testing.T) {
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
		if out != test.want {
			t.Errorf("Value(%s).ParseInt(10, 64) = %v, err(%s) want %v, err(%s)",
				test.key, out, err, test.want, test.wantErr)
		}
	}
}

func TestCollectionParseInt64Base(t *testing.T) {
	collection := ParseFromBytes(genBtoi64TestBytes())
	for _, test := range btoi64Tests {
		val := collection.Get(test.key)
		out, err := val.Int(test.base, 64)
		if test.wantErr != nil {
			if err == nil {
				t.Errorf("Value(%s).ParseInt(%d, 64) = %v, err(%v) want %v, err(%v)",
					test.key, test.base, out, err, test.want, test.wantErr)
			} else {
				if test.wantErr != err.(*strconv.NumError).Err {
					t.Errorf("Value(%s).ParseInt(%d, 64) = %v, err(%v) want %v, err(%v)",
						test.key, test.base, out, err, test.want, test.wantErr)
				}
			}
		}

		if out != test.want {
			t.Errorf("Value(%s).ParseInt(%d, 64) = %v, err(%v) want %v, err(%v)",
				test.key, test.base, out, err, test.want, test.wantErr)
		}
	}
}

func TestCollectionParseUint(t *testing.T) {
	switch strconv.IntSize {
	case 32:
		collection := ParseFromBytes(genAtoui32TestBytes())
		for _, test := range atoui32Tests {
			val := collection.Get(test.key)
			out, err := val.Uint(10, 0)
			if test.wantErr != nil {
				if err == nil {
					t.Errorf("Value(%s).ParseUint(10, 0) = %v, err(%s) want %v, err(%s)",
						test.key, out, err, test.want, test.wantErr)
				} else {
					if test.wantErr != err.(*strconv.NumError).Err {
						t.Errorf("Value(%s).ParseUint(10, 0) = %v, err(%s) want %v, err(%s)",
							test.key, out, err, test.want, test.wantErr)
					}
				}
			}
			if uint32(out) != test.want {
				t.Errorf("Value(%s).ParseUint(10, 0) = %v, err(%s) want %v, err(%s)",
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
					t.Errorf("Value(%s).ParseUint(10, 0) = %v, err(%s) want %v, err(%s)",
						test.key, out, err, test.want, test.wantErr)
				} else {
					if test.wantErr != err.(*strconv.NumError).Err {
						t.Errorf("Value(%s).ParseUint(10, 0) = %v, err(%s) want %v, err(%s)",
							test.key, out, err, test.want, test.wantErr)
					}
				}
			}
			if uint64(out) != test.want {
				t.Errorf("Value(%s).ParseUint(10, 0) = %v, err(%s) want %v, err(%s)",
					test.key, out, err, test.want, test.wantErr)
			}
		}
	}
}
