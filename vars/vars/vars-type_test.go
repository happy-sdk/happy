package vars

import . "github.com/onsi/ginkgo"

var _ = Describe("Testing with Ginkgo", func() {
	It("type_from_string", func() {

		val := NewValue("string var")
		val2 := Value("string var")
		if val != val2 {
			GinkgoT().Errorf("want: ValueFromString == Value got: ValueFromString = %q, val2 = %q", val, val2)
		}
	})
	It("type_as_bool", func() {

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
				GinkgoT().Errorf("want: %t got %t", got, tt.want)
			}
		}
	})
	It("type_from_bool", func() {

		tests := []struct {
			val  Value
			want string
		}{
			{ValueFromBool(true), "true"},
			{ValueFromBool(false), "false"},
		}
		for _, tt := range tests {
			if tt.val.String() != tt.want {
				GinkgoT().Errorf("want: %q got %q", tt.val.String(), tt.want)
			}
		}
	})
	It("type_as_uint", func() {

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
				GinkgoT().Errorf("want: %d got %d", got, tt.want)
			}
		}
	})
	It("type_as_uint8", func() {

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
				GinkgoT().Errorf("want: %d got %d", got, tt.want)
			}
		}
	})
	It("type_as_byte", func() {

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
				GinkgoT().Errorf("want: %d got %d", got, tt.want)
			}
		}
	})
	It("type_as_uint16", func() {

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
				GinkgoT().Errorf("want: %d got %d", got, tt.want)
			}
		}
	})
	It("type_as_uint32", func() {

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
				GinkgoT().Errorf("want: %d got %d", got, tt.want)
			}
		}
	})
	It("type_as_uint64", func() {

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
				GinkgoT().Errorf("want: %d got %d", got, tt.want)
			}
		}
	})
	It("type_as_uintptr", func() {

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
				GinkgoT().Errorf("want: %d got %d", got, tt.want)
			}
		}
	})
	It("type_as_int", func() {

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
				GinkgoT().Errorf("want: %d got %d", got, tt.want)
			}
		}
	})
	It("type_as_int8", func() {

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
				GinkgoT().Errorf("want: %d got %d", got, tt.want)
			}
		}
	})
	It("type_as_int16", func() {

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
				GinkgoT().Errorf("want: %d got %d", got, tt.want)
			}
		}
	})
	It("type_as_int32", func() {

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
				GinkgoT().Errorf("want: %d got %d", got, tt.want)
			}
		}
	})
	It("type_as_rune", func() {

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
				GinkgoT().Errorf("want: %d got %d", got, tt.want)
			}
		}
	})
	It("type_as_int64", func() {

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
				GinkgoT().Errorf("want: %d got %d", got, tt.want)
			}
		}
	})
	It("type_as_float32", func() {

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
				GinkgoT().Errorf("want: %f got %f", got, tt.want)
			}
		}
	})
	It("type_as_float64", func() {

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
				GinkgoT().Errorf("want: %f got %f", got, tt.want)
			}
		}
	})
	It("type_as_string_slice", func() {
	})
	It("type_as_complex64", func() {

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
				GinkgoT().Errorf("want: %f got %f", got, tt.want)
			}
		}
	})
	It("type_as_complex128", func() {

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
				GinkgoT().Errorf("want: %f got %f", got, tt.want)
			}
		}
	})
})
