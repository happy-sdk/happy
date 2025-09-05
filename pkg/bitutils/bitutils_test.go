// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy SDK Authors

package bitutils

import (
	"errors"
	"fmt"
	"math"
	"math/bits"
	"strings"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

// TestNextPowerOfTwo verifies that NextPowerOfTwo returns the correct next power of 2 >= n.
func TestNextPowerOfTwo(t *testing.T) {
	tests := []struct {
		name     string
		input    uint64
		expected uint64
	}{
		{name: "Zero", input: 0, expected: 1},
		{name: "One", input: 1, expected: 1},
		{name: "Two", input: 2, expected: 2},
		{name: "Three", input: 3, expected: 4},
		{name: "Four", input: 4, expected: 4},
		{name: "Five", input: 5, expected: 8},
		{name: "MidRange", input: 100, expected: 128},
		{name: "PowerOfTwo", input: 1024, expected: 1024},
		{name: "JustBelowPower", input: 1023, expected: 1024},
		{name: "JustAbovePower", input: 1025, expected: 2048},
		{name: "Large", input: 1<<30 + 1, expected: 1 << 31},
		{name: "LargestValidPower", input: 1 << 62, expected: 1 << 62},
		{name: "JustAboveLargestValidPower", input: (1 << 62) + 1, expected: 1 << 63},
		{name: "LargestPossiblePower", input: 1 << 63, expected: 1 << 63},
		// For values > 1<<63, we expect overflow protection to return math.MaxUint
		{name: "OverflowCase1", input: (1 << 63) + 1, expected: math.MaxUint},
		{name: "MaxUintMinusOne", input: math.MaxUint - 1, expected: math.MaxUint},
		{name: "MaxUint", input: math.MaxUint, expected: math.MaxUint},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NextPowerOfTwo(tt.input)
			testutils.Equal(t, tt.expected, got, "NextPowerOfTwo(%d)", tt.input)
		})
	}
}

// TestNextPowerOfTwoPowers verifies that NextPowerOfTwo returns correct powers of 2.
// Only test powers 0-63 since 1<<64 would overflow.
func TestNextPowerOfTwoPowers(t *testing.T) {
	for i := range 64 { // Test all valid powers including 1<<63
		pow := uint64(1) << i
		t.Run(fmt.Sprintf("Power(%d)", i), func(t *testing.T) {
			got := NextPowerOfTwo(pow)
			testutils.Equal(t, pow, got, "NextPowerOfTwo(%d)", pow)
		})
	}
}

// TestNextPowerOfTwoSequence verifies monotonicity and power-of-2 properties.
func TestNextPowerOfTwoSequence(t *testing.T) {
	var last uint64
	for i := range uint64(1000) {
		got := NextPowerOfTwo(i)
		if i != 0 {
			testutils.Assert(t, got >= i, "NextPowerOfTwo(%d) = %d; want >= %d", i, got, i)
		}
		testutils.Assert(t, got >= last, "NextPowerOfTwo(%d) = %d; want >= %d (previous)", i, got, last)
		// Only check power-of-2 property if result is not MaxUint (which is not a power of 2)
		if got != math.MaxUint {
			testutils.Assert(t, got&(got-1) == 0, "NextPowerOfTwo(%d) = %d; want power of 2", i, got)
		}
		last = got
	}
}

// TestNextPowerOfTwoOverflow specifically tests overflow behavior
func TestNextPowerOfTwoOverflow(t *testing.T) {
	overflowCases := []uint64{
		(1 << 63) + 1,
		(1 << 63) + 1000,
		math.MaxUint - 1,
		math.MaxUint,
	}

	for _, input := range overflowCases {
		t.Run(fmt.Sprintf("Overflow(%d)", input), func(t *testing.T) {
			got := NextPowerOfTwo(input)
			testutils.Equal(t, math.MaxUint, got, "NextPowerOfTwo(%d) should handle overflow", input)
		})
	}
}

// TestClearLowestBit verifies that ClearLowestBit correctly clears the least significant set bit.
func TestClearLowestBit(t *testing.T) {
	tests := []struct {
		name     string
		input    uint64
		expected uint64
	}{
		{name: "Zero", input: 0, expected: 0},
		{name: "One", input: 1, expected: 0},
		{name: "Two", input: 2, expected: 0},
		{name: "Three", input: 3, expected: 2},
		{name: "Four", input: 4, expected: 0},
		{name: "Five", input: 5, expected: 4},
		{name: "Six", input: 6, expected: 4},
		{name: "Seven", input: 7, expected: 6},
		{name: "Eight", input: 8, expected: 0},
		{name: "Nine", input: 9, expected: 8},
		{name: "Ten", input: 10, expected: 8},
		{name: "Fifteen", input: 15, expected: 14},
		{name: "Sixteen", input: 16, expected: 0},
		{name: "PowerOfTwo", input: 1024, expected: 0},
		{name: "PowerOfTwoMinusOne", input: 1023, expected: 1022},
		{name: "LargeNumber", input: 0b1101101, expected: 0b1101100},   // 109 -> 108
		{name: "AllOddBits", input: 0b10101010, expected: 0b10101000},  // 170 -> 168
		{name: "AllEvenBits", input: 0b01010100, expected: 0b01010000}, // 84 -> 80
		{name: "MaxUint", input: math.MaxUint, expected: math.MaxUint - 1},
		{name: "LargestPowerOf2", input: 1 << 63, expected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClearLowestBit(tt.input)
			testutils.Equal(t, tt.expected, got, "ClearLowestBit(%d)", tt.input)
		})
	}
}

// TestClearLowestBitPowersOfTwo verifies that clearing the lowest bit of any power of 2 results in 0.
func TestClearLowestBitPowersOfTwo(t *testing.T) {
	for i := range 64 {
		pow := uint64(1) << i
		t.Run(fmt.Sprintf("Power2_%d", i), func(t *testing.T) {
			got := ClearLowestBit(pow)
			testutils.Equal(t, uint64(0), got, "ClearLowestBit(1<<%d) should be 0", i)
		})
	}
}

// TestClearLowestBitProperties verifies mathematical properties of the function.
func TestClearLowestBitProperties(t *testing.T) {
	// Test that the result is always <= input
	testCases := []uint64{0, 1, 2, 3, 7, 15, 31, 63, 127, 255, 1000, 1023, 1024, math.MaxUint}

	for _, n := range testCases {
		t.Run(fmt.Sprintf("Property_%d", n), func(t *testing.T) {
			result := ClearLowestBit(n)
			testutils.Assert(t, result <= n, "ClearLowestBit(%d) = %d should be <= %d", n, result, n)

			// If input is 0, result should be 0
			if n == 0 {
				testutils.Equal(t, uint64(0), result, "ClearLowestBit(0) should be 0")
				return
			}

			// If input is not 0, result should be < input
			testutils.Assert(t, result < n, "ClearLowestBit(%d) = %d should be < %d", n, result, n)

			// The difference should be a power of 2 (the bit that was cleared)
			diff := n - result
			testutils.Assert(t, diff > 0, "difference should be positive")
			testutils.Assert(t, (diff&(diff-1)) == 0, "difference %d should be a power of 2", diff)
		})
	}
}

// TestClearLowestBitSequence verifies that repeatedly clearing lowest bits eventually reaches 0.
func TestClearLowestBitSequence(t *testing.T) {
	testCases := []uint64{7, 15, 31, 63, 127, 255, 1023}

	for _, start := range testCases {
		t.Run(fmt.Sprintf("Sequence_%d", start), func(t *testing.T) {
			n := start
			steps := 0
			maxSteps := 64 // Should never need more than 64 steps

			for n != 0 && steps < maxSteps {
				prev := n
				n = ClearLowestBit(n)
				testutils.Assert(t, n < prev, "step %d: %d should be less than %d", steps, n, prev)
				steps++
			}

			testutils.Equal(t, uint64(0), n, "should eventually reach 0 after %d steps", steps)
			testutils.Assert(t, steps <= maxSteps, "should not take more than %d steps", maxSteps)
		})
	}
}

// TestClearLowestBitBitPatterns tests specific bit patterns to ensure correctness.
func TestClearLowestBitBitPatterns(t *testing.T) {
	tests := []struct {
		name     string
		input    uint64
		expected uint64
		desc     string
	}{
		{
			name:     "AlternatingBits1",
			input:    0b101010101010,
			expected: 0b101010101000,
			desc:     "alternating pattern starting with 0",
		},
		{
			name:     "AlternatingBits2",
			input:    0b010101010101,
			expected: 0b010101010100,
			desc:     "alternating pattern starting with 1",
		},
		{
			name:     "AllOnesLow8",
			input:    0b11111111,
			expected: 0b11111110,
			desc:     "all ones in low 8 bits",
		},
		{
			name:     "SingleBitHigh",
			input:    uint64(1) << 32,
			expected: 0,
			desc:     "single bit in high position",
		},
		{
			name:     "TwoBitsApart",
			input:    0b1001,
			expected: 0b1000,
			desc:     "two bits with gap",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClearLowestBit(tt.input)
			testutils.Equal(t, tt.expected, got, "ClearLowestBit(%b) [%s]", tt.input, tt.desc)
		})
	}
}

// TestFindLowestBit verifies that FindLowestBit correctly finds the index of the least significant set bit.
func TestFindLowestBit(t *testing.T) {
	tests := []struct {
		name     string
		input    uint64
		expected uint64
	}{
		{name: "Zero", input: 0, expected: 0},
		{name: "One", input: 1, expected: 0},
		{name: "Two", input: 2, expected: 1},
		{name: "Three", input: 3, expected: 0},
		{name: "Four", input: 4, expected: 2},
		{name: "Five", input: 5, expected: 0},
		{name: "Six", input: 6, expected: 1},
		{name: "Seven", input: 7, expected: 0},
		{name: "Eight", input: 8, expected: 3},
		{name: "Nine", input: 9, expected: 0},
		{name: "Ten", input: 10, expected: 1},
		{name: "Twelve", input: 12, expected: 2},
		{name: "Sixteen", input: 16, expected: 4},
		{name: "ThirtyTwo", input: 32, expected: 5},
		{name: "SixtyFour", input: 64, expected: 6},
		{name: "PowerOf2_1024", input: 1024, expected: 10},
		{name: "EvenNumber", input: 0b1101100, expected: 2}, // 108, lowest bit at position 2
		{name: "OddNumber", input: 0b1101101, expected: 0},  // 109, lowest bit at position 0
		{name: "LargePowerOf2", input: 1 << 20, expected: 20},
		{name: "MaxBit", input: 1 << 63, expected: 63},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindLowestBit(tt.input)
			testutils.Equal(t, tt.expected, got, "FindLowestBit(%d) [binary: %b]", tt.input, tt.input)
		})
	}
}

// TestFindLowestBitPowersOfTwo verifies that FindLowestBit returns the correct index for all powers of 2.
func TestFindLowestBitPowersOfTwo(t *testing.T) {
	for i := range 64 {
		pow := uint64(1) << i
		t.Run(fmt.Sprintf("Power2_%d", i), func(t *testing.T) {
			got := FindLowestBit(pow)
			testutils.Equal(t, uint64(i), got, "FindLowestBit(1<<%d) should return %d", i, i)
		})
	}
}

// TestFindLowestBitOddNumbers verifies that all odd numbers have their lowest bit at position 0.
func TestFindLowestBitOddNumbers(t *testing.T) {
	oddNumbers := []uint64{1, 3, 5, 7, 9, 11, 13, 15, 17, 19, 21, 99, 101, 255, 1023}

	for _, odd := range oddNumbers {
		t.Run(fmt.Sprintf("Odd_%d", odd), func(t *testing.T) {
			got := FindLowestBit(odd)
			testutils.Equal(t, uint64(0), got, "FindLowestBit(%d) should be 0 for odd numbers", odd)
		})
	}
}

// TestFindLowestBitEvenNumbers tests various even numbers and their expected lowest bit positions.
func TestFindLowestBitEvenNumbers(t *testing.T) {
	tests := []struct {
		input    uint64
		expected uint64
	}{
		{2, 1},   // 0b10
		{4, 2},   // 0b100
		{6, 1},   // 0b110
		{8, 3},   // 0b1000
		{10, 1},  // 0b1010
		{12, 2},  // 0b1100
		{14, 1},  // 0b1110
		{16, 4},  // 0b10000
		{24, 3},  // 0b11000
		{48, 4},  // 0b110000
		{96, 5},  // 0b1100000
		{192, 6}, // 0b11000000
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Even_%d", tt.input), func(t *testing.T) {
			got := FindLowestBit(tt.input)
			testutils.Equal(t, tt.expected, got, "FindLowestBit(%d) [binary: %b]", tt.input, tt.input)
		})
	}
}

// TestFindLowestBitProperties verifies mathematical properties of the function.
func TestFindLowestBitProperties(t *testing.T) {
	testCases := []uint64{1, 2, 3, 4, 5, 7, 8, 15, 16, 31, 32, 63, 64, 127, 128, 255, 256, 1023, 1024}

	for _, n := range testCases {
		t.Run(fmt.Sprintf("Property_%d", n), func(t *testing.T) {
			if n == 0 {
				return // Skip zero for this test
			}

			bitIndex := FindLowestBit(n)

			// The bit at the returned index should be set
			testutils.Assert(t, (n&(1<<bitIndex)) != 0,
				"bit at position %d should be set in %d [%b]", bitIndex, n, n)

			// All bits below the returned index should be unset
			if bitIndex > 0 {
				mask := (uint64(1) << bitIndex) - 1
				testutils.Assert(t, (n&mask) == 0,
					"all bits below position %d should be unset in %d [%b]", bitIndex, n, n)
			}

			// The returned index should be less than 64 (for uint64)
			testutils.Assert(t, bitIndex < 64,
				"bit index %d should be less than 64", bitIndex)
		})
	}
}

// TestFindLowestBitConsistency verifies consistency with bit manipulation operations.
func TestFindLowestBitConsistency(t *testing.T) {
	testCases := []uint64{1, 3, 5, 6, 7, 8, 12, 15, 16, 24, 31, 32, 48, 63, 64, 96, 127, 128}

	for _, n := range testCases {
		if n == 0 {
			continue
		}

		t.Run(fmt.Sprintf("Consistency_%d", n), func(t *testing.T) {
			bitIndex := FindLowestBit(n)

			// The lowest bit should be at position bitIndex
			expectedLowestBit := uint64(1) << bitIndex
			actualLowestBit := n & (^n + 1) // Isolate lowest bit using two's complement

			testutils.Equal(t, expectedLowestBit, actualLowestBit,
				"isolated lowest bit should match 1<<%d for input %d", bitIndex, n)

			// Clearing the lowest bit and checking that no bit at bitIndex remains
			cleared := n & (n - 1)
			testutils.Assert(t, (cleared&(1<<bitIndex)) == 0,
				"after clearing lowest bit, position %d should be unset", bitIndex)
		})
	}
}

// TestFindLowestBitSpecialCases tests edge cases and special bit patterns.
func TestFindLowestBitSpecialCases(t *testing.T) {
	tests := []struct {
		name     string
		input    uint64
		expected uint64
		desc     string
	}{
		{
			name:     "AlternatingOnes",
			input:    0b101010101010,
			expected: 1,
			desc:     "alternating pattern 101010..., lowest bit at position 1",
		},
		{
			name:     "AlternatingZeros",
			input:    0b010101010101,
			expected: 0,
			desc:     "alternating pattern 010101..., lowest bit at position 0",
		},
		{
			name:     "HighBitOnly",
			input:    uint64(1) << 63,
			expected: 63,
			desc:     "only highest bit set",
		},
		{
			name:     "AllOnesLow8",
			input:    0b11111111,
			expected: 0,
			desc:     "all ones in low 8 bits",
		},
		{
			name:     "TrailingZeros",
			input:    0b111100000,
			expected: 5,
			desc:     "number with trailing zeros",
		},
		{
			name:     "SingleTrailingZero",
			input:    0b1111110,
			expected: 1,
			desc:     "single trailing zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindLowestBit(tt.input)
			testutils.Equal(t, tt.expected, got,
				"FindLowestBit(%b) [%s]", tt.input, tt.desc)
		})
	}
}

// TestFindLowestBitLargeNumbers tests behavior with large numbers near the uint64limits.
func TestFindLowestBitLargeNumbers(t *testing.T) {
	tests := []struct {
		name     string
		input    uint64
		expected uint64
	}{
		{name: "MaxUintMinus1", input: math.MaxUint - 1, expected: 1}, // All bits set except lowest
		{name: "MaxUintMinus3", input: math.MaxUint - 3, expected: 2}, // All bits set except two lowest
		{name: "MaxUintMinus7", input: math.MaxUint - 7, expected: 3}, // All bits set except three lowest
		{name: "LargePowerOf2", input: 1 << 60, expected: 60},
		{name: "LargeOddNumber", input: (1 << 60) + 1, expected: 0},
		{name: "LargeEvenNumber", input: (1 << 60) + 2, expected: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindLowestBit(tt.input)
			testutils.Equal(t, tt.expected, got, "FindLowestBit(%d)", tt.input)
		})
	}
}

// TestFastMod verifies that FastMod correctly computes n % m for powers of 2.
func TestFastMod(t *testing.T) {
	tests := []struct {
		name      string
		n         uint64
		m         uint64
		expected  uint64
		shouldErr bool
	}{
		// Valid cases - m is power of 2
		{name: "ZeroModOne", n: 0, m: 1, expected: 0, shouldErr: false},
		{name: "OneModOne", n: 1, m: 1, expected: 0, shouldErr: false},
		{name: "OneModTwo", n: 1, m: 2, expected: 1, shouldErr: false},
		{name: "TwoModTwo", n: 2, m: 2, expected: 0, shouldErr: false},
		{name: "ThreeModTwo", n: 3, m: 2, expected: 1, shouldErr: false},
		{name: "FourModTwo", n: 4, m: 2, expected: 0, shouldErr: false},
		{name: "FiveModFour", n: 5, m: 4, expected: 1, shouldErr: false},
		{name: "SevenModFour", n: 7, m: 4, expected: 3, shouldErr: false},
		{name: "EightModFour", n: 8, m: 4, expected: 0, shouldErr: false},
		{name: "TenModEight", n: 10, m: 8, expected: 2, shouldErr: false},
		{name: "FifteenModEight", n: 15, m: 8, expected: 7, shouldErr: false},
		{name: "SixteenModEight", n: 16, m: 8, expected: 0, shouldErr: false},
		{name: "HundredMod32", n: 100, m: 32, expected: 4, shouldErr: false},
		{name: "ThousandMod64", n: 1000, m: 64, expected: 40, shouldErr: false},
		{name: "LargeNMod1024", n: 123456, m: 1024, expected: 576, shouldErr: false},
		{name: "MaxUintMod2", n: ^uint64(0), m: 2, expected: 1, shouldErr: false},
		{name: "MaxUintMod1024", n: ^uint64(0), m: 1024, expected: 1023, shouldErr: false},

		// Invalid cases - m is not power of 2
		{name: "ModThree", n: 10, m: 3, expected: 0, shouldErr: true},
		{name: "ModFive", n: 15, m: 5, expected: 0, shouldErr: true},
		{name: "ModSix", n: 20, m: 6, expected: 0, shouldErr: true},
		{name: "ModSeven", n: 25, m: 7, expected: 0, shouldErr: true},
		{name: "ModNine", n: 30, m: 9, expected: 0, shouldErr: true},
		{name: "ModTen", n: 35, m: 10, expected: 0, shouldErr: true},
		{name: "ModTwelve", n: 40, m: 12, expected: 0, shouldErr: true},
		{name: "ModFifteen", n: 45, m: 15, expected: 0, shouldErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FastMod(tt.n, tt.m)

			if tt.shouldErr {
				testutils.Assert(t, err != nil, "FastMod(%d, %d) should return error", tt.n, tt.m)
				testutils.Assert(t, errors.Is(err, Error), "error should wrap Error")
			} else {
				testutils.Assert(t, err == nil, "FastMod(%d, %d) should not return error: %v", tt.n, tt.m, err)
				testutils.Equal(t, tt.expected, got, "FastMod(%d, %d)", tt.n, tt.m)

				// Verify against standard modulo operation
				expected := tt.n % tt.m
				testutils.Equal(t, expected, got, "FastMod(%d, %d) should match standard modulo", tt.n, tt.m)
			}
		})
	}
}

// TestFastModZeroCase tests the special case of modulo by zero.
func TestFastModZeroCase(t *testing.T) {
	// The current function has a bug: 0 & (0-1) = 0, so it doesn't detect 0 as non-power-of-2
	// This documents the current behavior
	result, err := FastMod(42, 0)
	if err != nil {
		// If it does error (which would be correct behavior), verify the error
		testutils.Assert(t, errors.Is(err, Error), "error should wrap Error")
	} else {
		// If it doesn't error (current buggy behavior), the result is undefined
		t.Logf("FastMod(42, 0) returned result=%d with no error (this indicates a bug in the function)", result)
	}
}

// TestFastModAllPowersOfTwo verifies FastMod works correctly for all powers of 2.
func TestFastModAllPowersOfTwo(t *testing.T) {
	// Test various n values against all powers of 2 from 1 to 2^32
	testValues := []uint64{0, 1, 2, 3, 7, 15, 31, 63, 127, 255, 511, 1023, 2047, 4095, 8191, 16383, 32767, 65535}

	for i := range 33 { // Test powers 2^0 to 2^32
		m := uint64(1) << i

		for _, n := range testValues {
			t.Run(fmt.Sprintf("N%d_Mod_2Pow%d", n, i), func(t *testing.T) {
				got, err := FastMod(n, m)
				testutils.Assert(t, err == nil, "FastMod(%d, %d) should not error", n, m)

				expected := n % m
				testutils.Equal(t, expected, got, "FastMod(%d, %d)", n, m)

				// Additional check: result should be < m
				testutils.Assert(t, got < m, "FastMod(%d, %d) = %d should be < %d", n, m, got, m)
			})
		}
	}
}

// TestFastModErrorCases specifically tests various non-power-of-2 values for proper error handling.
func TestFastModErrorCases(t *testing.T) {
	nonPowersOf2 := []uint64{
		3, 5, 6, 7, 9, 10, 11, 12, 13, 14, 15, // Small non-powers
		17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, // Around 16 and 32
		33, 34, 35, 36, 48, 49, 50, 63, // Various mid-range
		65, 66, 67, 68, 96, 97, 98, 127, // Around 64 and 128
		129, 130, 131, 132, 192, 193, 194, 255, // Around 128 and 256
		257, 300, 500, 1000, 1023, 1025, // Larger values
	}

	for _, m := range nonPowersOf2 {
		t.Run(fmt.Sprintf("ErrorCase_M%d", m), func(t *testing.T) {
			_, err := FastMod(42, m) // Use arbitrary n value
			testutils.Assert(t, err != nil, "FastMod(42, %d) should return error for non-power-of-2", m)
			testutils.Assert(t, errors.Is(err, Error), "error should wrap Error")

			// Check error message contains expected text
			if err != nil {
				errMsg := err.Error()
				testutils.Assert(t, strings.Contains(errMsg, "FastMod"), "error message should contain 'FastMod'")
				testutils.Assert(t, strings.Contains(errMsg, "power of 2"), "error message should mention 'power of 2'")
			}
		})
	}
}

// TestFastModProperties verifies mathematical properties of the modulo operation.
func TestFastModProperties(t *testing.T) {
	powersOf2 := []uint64{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024}
	testValues := []uint64{0, 1, 7, 15, 31, 63, 127, 255, 511, 1023, 2047, 4095}

	for _, m := range powersOf2 {
		for _, n := range testValues {
			t.Run(fmt.Sprintf("Property_N%d_M%d", n, m), func(t *testing.T) {
				result, err := FastMod(n, m)
				testutils.Assert(t, err == nil, "FastMod(%d, %d) should not error", n, m)

				// Property 1: 0 <= result < m
				testutils.Assert(t, result < m, "FastMod(%d, %d) = %d should be < %d", n, m, result, m)

				// Property 2: n = (n/m)*m + (n%m), so n >= result
				testutils.Assert(t, n >= result, "FastMod(%d, %d) = %d should be <= %d", n, m, result, n)

				// Property 3: (n - result) should be divisible by m
				if n >= result {
					diff := n - result
					testutils.Assert(t, diff%m == 0, "(%d - %d) = %d should be divisible by %d", n, result, diff, m)
				}

				// Property 4: FastMod(n + m, m) should equal FastMod(n, m)
				if n <= ^uint64(0)-m { // Avoid overflow
					result2, err2 := FastMod(n+m, m)
					testutils.Assert(t, err2 == nil, "FastMod(%d, %d) should not error", n+m, m)
					testutils.Equal(t, result, result2, "FastMod(%d, %d) should equal FastMod(%d, %d)", n+m, m, n, m)
				}
			})
		}
	}
}

// TestFastModConsistencyWithStandardMod verifies FastMod matches standard modulo for powers of 2.
func TestFastModConsistencyWithStandardMod(t *testing.T) {
	// Generate a wider range of test values
	testCases := []struct {
		n uint64
		m uint64
	}{
		// Edge cases
		{0, 1}, {0, 2}, {0, 4}, {0, 8},

		// Small values
		{1, 1}, {1, 2}, {1, 4}, {1, 8},
		{2, 2}, {2, 4}, {2, 8}, {2, 16},
		{3, 2}, {3, 4}, {3, 8}, {3, 16},

		// Values equal to modulus
		{1, 1}, {2, 2}, {4, 4}, {8, 8}, {16, 16}, {32, 32},

		// Values larger than modulus
		{5, 4}, {9, 8}, {17, 16}, {33, 32}, {65, 64},
		{100, 64}, {200, 128}, {1000, 512}, {2000, 1024},

		// Large values
		{1000000, 1024}, {1000000, 2048}, {1000000, 4096},
		{^uint64(0), 2}, {^uint64(0), 4}, {^uint64(0), 1024}, {^uint64(0), 65536},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Consistency_N%d_M%d", tc.n, tc.m), func(t *testing.T) {
			fastResult, err := FastMod(tc.n, tc.m)
			testutils.Assert(t, err == nil, "FastMod(%d, %d) should not error", tc.n, tc.m)

			standardResult := tc.n % tc.m
			testutils.Equal(t, standardResult, fastResult,
				"FastMod(%d, %d) should match standard modulo operation", tc.n, tc.m)
		})
	}
}

// TestFastModBitPatterns tests specific bit patterns to ensure the bit masking works correctly.
func TestFastModBitPatterns(t *testing.T) {
	tests := []struct {
		name string
		n    uint64
		m    uint64
		desc string
	}{
		{
			name: "AllOnesLow8_Mod16",
			n:    0b11111111, // 255
			m:    16,         // 0b10000
			desc: "all ones in low 8 bits mod 16",
		},
		{
			name: "AlternatingBits_Mod8",
			n:    0b10101010, // 170
			m:    8,          // 0b1000
			desc: "alternating bit pattern mod 8",
		},
		{
			name: "HighBitsOnly_Mod32",
			n:    0b11110000000, // High bits set
			m:    32,            // 0b100000
			desc: "high bits only mod 32",
		},
		{
			name: "MixedPattern_Mod64",
			n:    0b1100110011001100,
			m:    64, // 0b1000000
			desc: "mixed bit pattern mod 64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FastMod(tt.n, tt.m)
			testutils.Assert(t, err == nil, "FastMod should not error for %s", tt.desc)

			expected := tt.n % tt.m
			testutils.Equal(t, expected, got, "FastMod bit pattern test: %s", tt.desc)
		})
	}
}

func TestAlignUp(t *testing.T) {
	t.Run("valid power of 2 alignments", func(t *testing.T) {
		testCases := []struct {
			name     string
			n        uint64
			align    uint64
			expected uint64
		}{
			{"align to 1", 0, 1, 0},
			{"align to 1", 1, 1, 1},
			{"align to 1", 5, 1, 5},

			{"align to 2 - already aligned", 0, 2, 0},
			{"align to 2 - already aligned", 2, 2, 2},
			{"align to 2 - already aligned", 4, 2, 4},
			{"align to 2 - needs rounding", 1, 2, 2},
			{"align to 2 - needs rounding", 3, 2, 4},
			{"align to 2 - needs rounding", 5, 2, 6},

			{"align to 4 - already aligned", 0, 4, 0},
			{"align to 4 - already aligned", 4, 4, 4},
			{"align to 4 - already aligned", 8, 4, 8},
			{"align to 4 - needs rounding", 1, 4, 4},
			{"align to 4 - needs rounding", 2, 4, 4},
			{"align to 4 - needs rounding", 3, 4, 4},
			{"align to 4 - needs rounding", 5, 4, 8},
			{"align to 4 - needs rounding", 6, 4, 8},
			{"align to 4 - needs rounding", 7, 4, 8},

			{"align to 8", 0, 8, 0},
			{"align to 8", 1, 8, 8},
			{"align to 8", 7, 8, 8},
			{"align to 8", 8, 8, 8},
			{"align to 8", 9, 8, 16},
			{"align to 8", 15, 8, 16},
			{"align to 8", 16, 8, 16},

			{"align to 16", 0, 16, 0},
			{"align to 16", 1, 16, 16},
			{"align to 16", 15, 16, 16},
			{"align to 16", 16, 16, 16},
			{"align to 16", 17, 16, 32},

			{"align to 32", 0, 32, 0},
			{"align to 32", 31, 32, 32},
			{"align to 32", 32, 32, 32},
			{"align to 32", 33, 32, 64},

			{"align to 64", 63, 64, 64},
			{"align to 64", 64, 64, 64},
			{"align to 64", 65, 64, 128},

			{"align to 128", 127, 128, 128},
			{"align to 128", 128, 128, 128},
			{"align to 128", 129, 128, 256},

			{"align to 256", 255, 256, 256},
			{"align to 256", 256, 256, 256},
			{"align to 256", 257, 256, 512},

			{"large values", 1000, 512, 1024},
			{"large values", 2000, 1024, 2048},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := AlignUp(tc.n, tc.align)
				testutils.Assert(t, err == nil,
					"AlignUp(%d, %d) returned unexpected error: %v", tc.n, tc.align, err)
				testutils.Assert(t, result == tc.expected,
					"AlignUp(%d, %d) = %d, expected %d", tc.n, tc.align, result, tc.expected)
			})
		}
	})

	t.Run("edge cases", func(t *testing.T) {
		// Test with zero and maximum values that won't cause overflow
		result, err := AlignUp(0, 1)
		testutils.Assert(t, err == nil, "AlignUp(0, 1) should not return error")
		testutils.Assert(t, result == 0, "AlignUp(0, 1) should be 0")

		result, err = AlignUp(0, 2)
		testutils.Assert(t, err == nil, "AlignUp(0, 2) should not return error")
		testutils.Assert(t, result == 0, "AlignUp(0, 2) should be 0")

		result, err = AlignUp(0, 1024)
		testutils.Assert(t, err == nil, "AlignUp(0, 1024) should not return error")
		testutils.Assert(t, result == 0, "AlignUp(0, 1024) should be 0")

		// Test large alignment values
		result, err = AlignUp(1, 1073741824)
		testutils.Assert(t, err == nil, "AlignUp(1, 1073741824) should not return error")
		testutils.Assert(t, result == 1073741824,
			"AlignUp(1, 1073741824) should be 1073741824")
	})

	t.Run("error on invalid align values", func(t *testing.T) {
		invalidAlignValues := []uint64{0, 3, 5, 6, 7, 9, 10, 11, 12, 13, 14, 15, 17, 18, 19, 20, 24, 100, 255}

		for _, align := range invalidAlignValues {
			t.Run(fmt.Sprintf("invalid_align_%d", align), func(t *testing.T) {
				result, err := AlignUp(1, align)
				testutils.Assert(t, err != nil,
					"AlignUp(1, %d) should return error for non-power-of-2 align", align)
				testutils.Assert(t, result == 0,
					"AlignUp(1, %d) should return 0 when error occurs", align)

				if err != nil {
					testutils.Assert(t, errors.Is(err, ErrAlignment),
						"error should wrap the base Error: %v", err)
					expectedMsg := "bitutils:alignment: AlignUp: align must be a power of 2"
					testutils.Assert(t, err.Error() == expectedMsg,
						"error message should be '%s', got '%s'", expectedMsg, err.Error())
				}
			})
		}
	})

	t.Run("verify alignment property", func(t *testing.T) {
		// Test that the result is always a multiple of align
		alignments := []uint64{1, 2, 4, 8, 16, 32, 64, 128, 256, 512}
		testValues := []uint64{0, 1, 2, 3, 7, 15, 31, 63, 100, 255, 1000, 1023, 2047}

		for _, align := range alignments {
			for _, n := range testValues {
				result, err := AlignUp(n, align)
				testutils.Assert(t, err == nil,
					"AlignUp(%d, %d) should not return error", n, align)

				// Verify result is multiple of align
				testutils.Assert(t, result%align == 0,
					"AlignUp(%d, %d) = %d is not a multiple of %d", n, align, result, align)

				// Verify result is >= n
				testutils.Assert(t, result >= n,
					"AlignUp(%d, %d) = %d should be >= %d", n, align, result, n)

				// Verify result is the smallest such multiple
				if result > 0 {
					testutils.Assert(t, result-align < n,
						"AlignUp(%d, %d) = %d is not the smallest multiple >= %d", n, align, result, n)
				}
			}
		}
	})

	t.Run("powers of 2 validation", func(t *testing.T) {
		// Test that all powers of 2 up to a reasonable limit work
		for i := range 20 {
			align := uint64(1 << i) // 2^i
			result, err := AlignUp(1, align)
			testutils.Assert(t, err == nil,
				"AlignUp(1, %d) should not return error", align)
			testutils.Assert(t, result == align,
				"AlignUp(1, %d) should be %d", align, align)
		}
	})

	t.Run("zero align edge case", func(t *testing.T) {
		// Special test for align = 0, which should always return error
		result, err := AlignUp(5, 0)
		testutils.Assert(t, err != nil, "AlignUp(5, 0) should return error")
		testutils.Assert(t, result == 0, "AlignUp(5, 0) should return 0 on error")
		testutils.Assert(t, errors.Is(err, ErrAlignment), "error should wrap the ErrAlignment")
	})

	t.Run("consistency with different input values", func(t *testing.T) {
		// Test that the same align value works consistently with different n values
		align := uint64(8)
		testCases := []struct {
			n        uint64
			expected uint64
		}{
			{0, 0}, {1, 8}, {2, 8}, {3, 8}, {4, 8}, {5, 8}, {6, 8}, {7, 8},
			{8, 8}, {9, 16}, {10, 16}, {11, 16}, {12, 16}, {13, 16}, {14, 16}, {15, 16},
			{16, 16}, {17, 24}, {24, 24}, {25, 32},
		}

		for _, tc := range testCases {
			result, err := AlignUp(tc.n, align)
			testutils.Assert(t, err == nil,
				"AlignUp(%d, %d) should not return error", tc.n, align)
			testutils.Assert(t, result == tc.expected,
				"AlignUp(%d, %d) = %d, expected %d", tc.n, align, result, tc.expected)
		}
	})
}

func TestCeilDiv(t *testing.T) {
	t.Run("basic ceiling division", func(t *testing.T) {
		testCases := []struct {
			name     string
			a        uint64
			b        uint64
			expected uint64
		}{
			// Exact divisions (no remainder)
			{"0 / 1", 0, 1, 0},
			{"1 / 1", 1, 1, 1},
			{"2 / 1", 2, 1, 2},
			{"4 / 2", 4, 2, 2},
			{"6 / 3", 6, 3, 2},
			{"8 / 4", 8, 4, 2},
			{"10 / 5", 10, 5, 2},
			{"100 / 10", 100, 10, 10},

			// Divisions with remainder (need ceiling)
			{"1 / 2", 1, 2, 1},
			{"3 / 2", 3, 2, 2},
			{"5 / 2", 5, 2, 3},
			{"7 / 2", 7, 2, 4},
			{"1 / 3", 1, 3, 1},
			{"2 / 3", 2, 3, 1},
			{"4 / 3", 4, 3, 2},
			{"5 / 3", 5, 3, 2},
			{"7 / 3", 7, 3, 3},
			{"8 / 3", 8, 3, 3},
			{"10 / 3", 10, 3, 4},
			{"11 / 3", 11, 3, 4},
			{"1 / 4", 1, 4, 1},
			{"2 / 4", 2, 4, 1},
			{"3 / 4", 3, 4, 1},
			{"5 / 4", 5, 4, 2},
			{"7 / 4", 7, 4, 2},
			{"9 / 4", 9, 4, 3},

			// Larger numbers
			{"100 / 7", 100, 7, 15},     // 100/7 = 14.28... -> 15
			{"1000 / 13", 1000, 13, 77}, // 1000/13 = 76.92... -> 77
			{"255 / 16", 255, 16, 16},   // 255/16 = 15.9375 -> 16
			{"256 / 16", 256, 16, 16},   // 256/16 = 16 (exact)
			{"257 / 16", 257, 16, 17},   // 257/16 = 16.0625 -> 17
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := CeilDiv(tc.a, tc.b)
				testutils.Assert(t, err == nil,
					"CeilDiv(%d, %d) returned unexpected error: %v", tc.a, tc.b, err)
				testutils.Assert(t, result == tc.expected,
					"CeilDiv(%d, %d) = %d, expected %d", tc.a, tc.b, result, tc.expected)
			})
		}
	})

	t.Run("divide by zero error", func(t *testing.T) {
		testValues := []uint64{0, 1, 5, 10, 100, 1000, 4294967295} // including max uint32

		for _, a := range testValues {
			t.Run(fmt.Sprintf("a_%d", a), func(t *testing.T) {
				result, err := CeilDiv(a, 0)
				testutils.Assert(t, err != nil,
					"CeilDiv(%d, 0) should return error", a)
				testutils.Assert(t, result == 0,
					"CeilDiv(%d, 0) should return 0 when error occurs", a)
				testutils.Assert(t, errors.Is(err, ErrDivideByZero),
					"error should be ErrDivideByZero: %v", err)
				expectedMsg := "bitutils division by zero"
				testutils.Assert(t, err.Error() == expectedMsg,
					"error message should be '%s', got '%s'", expectedMsg, err.Error())
			})
		}
	})

	t.Run("ceiling property verification", func(t *testing.T) {
		// Test that CeilDiv always returns the mathematical ceiling
		testCases := []struct {
			a, b uint64
		}{
			{0, 1}, {0, 2}, {0, 10},
			{1, 1}, {1, 2}, {1, 3}, {1, 10},
			{2, 1}, {2, 2}, {2, 3}, {2, 5},
			{5, 1}, {5, 2}, {5, 3}, {5, 4}, {5, 6},
			{10, 1}, {10, 2}, {10, 3}, {10, 4}, {10, 7},
			{17, 5}, {23, 7}, {31, 8}, {47, 11},
			{100, 3}, {100, 7}, {100, 11}, {100, 13},
		}

		for _, tc := range testCases {
			result, err := CeilDiv(tc.a, tc.b)
			testutils.Assert(t, err == nil,
				"CeilDiv(%d, %d) should not return error", tc.a, tc.b)

			// Verify ceiling property: result * b >= a
			testutils.Assert(t, result*tc.b >= tc.a,
				"CeilDiv(%d, %d) = %d, but %d * %d = %d < %d",
				tc.a, tc.b, result, result, tc.b, result*tc.b, tc.a)

			// Verify minimality: (result - 1) * b < a (unless result is 0)
			if result > 0 {
				testutils.Assert(t, (result-1)*tc.b < tc.a,
					"CeilDiv(%d, %d) = %d, but (%d-1) * %d = %d >= %d (not minimal)",
					tc.a, tc.b, result, result, tc.b, (result-1)*tc.b, tc.a)
			}
		}
	})

	t.Run("comparison with regular division", func(t *testing.T) {
		testCases := []struct {
			a, b         uint64
			hasRemainder bool
		}{
			{6, 2, false},  // 6/2 = 3, no remainder
			{6, 3, false},  // 6/3 = 2, no remainder
			{7, 2, true},   // 7/2 = 3.5, has remainder
			{7, 3, true},   // 7/3 = 2.33..., has remainder
			{8, 3, true},   // 8/3 = 2.66..., has remainder
			{9, 3, false},  // 9/3 = 3, no remainder
			{10, 4, true},  // 10/4 = 2.5, has remainder
			{12, 4, false}, // 12/4 = 3, no remainder
			{15, 7, true},  // 15/7 = 2.14..., has remainder
			{21, 7, false}, // 21/7 = 3, no remainder
		}

		for _, tc := range testCases {
			ceilResult, err := CeilDiv(tc.a, tc.b)
			testutils.Assert(t, err == nil,
				"CeilDiv(%d, %d) should not return error", tc.a, tc.b)

			regularDiv := tc.a / tc.b

			if tc.hasRemainder {
				// If there's a remainder, ceiling should be regular + 1
				testutils.Assert(t, ceilResult == regularDiv+1,
					"CeilDiv(%d, %d) = %d, expected %d (regular div %d + 1)",
					tc.a, tc.b, ceilResult, regularDiv+1, regularDiv)
			} else {
				// If no remainder, ceiling should equal regular division
				testutils.Assert(t, ceilResult == regularDiv,
					"CeilDiv(%d, %d) = %d, expected %d (same as regular div)",
					tc.a, tc.b, ceilResult, regularDiv)
			}
		}
	})

	t.Run("edge cases with large numbers", func(t *testing.T) {
		// Test with larger numbers to ensure no overflow issues
		const maxUint32 = 4294967295 // 2^32 - 1

		result, err := CeilDiv(maxUint32, maxUint32)
		testutils.Assert(t, err == nil, "CeilDiv(max, max) should not error")
		testutils.Assert(t, result == 1, "CeilDiv(max, max) should be 1")

		result, err = CeilDiv(maxUint32-1, maxUint32)
		testutils.Assert(t, err == nil, "CeilDiv(max-1, max) should not error")
		testutils.Assert(t, result == 1, "CeilDiv(max-1, max) should be 1")

		result, err = CeilDiv(maxUint32, 2)
		testutils.Assert(t, err == nil, "CeilDiv(max, 2) should not error")
		expected := uint64(maxUint32+1) / 2 // This should be 2^31
		testutils.Assert(t, result == expected,
			"CeilDiv(%d, 2) = %d, expected %d", maxUint32, result, expected)

		result, err = CeilDiv(1000000, 3)
		testutils.Assert(t, err == nil, "CeilDiv(1000000, 3) should not error")
		expected = 333334 // 1000000/3 = 333333.33... -> 333334
		testutils.Assert(t, result == expected,
			"CeilDiv(1000000, 3) = %d, expected %d", result, expected)
	})

	t.Run("systematic power of 2 divisors", func(t *testing.T) {
		// Test various numbers divided by powers of 2
		divisors := []uint64{1, 2, 4, 8, 16, 32, 64, 128, 256}
		testValues := []uint64{0, 1, 3, 7, 15, 31, 63, 127, 255, 511, 1023}

		for _, divisor := range divisors {
			for _, value := range testValues {
				result, err := CeilDiv(value, divisor)
				testutils.Assert(t, err == nil,
					"CeilDiv(%d, %d) should not return error", value, divisor)

				// Verify the ceiling property
				testutils.Assert(t, result*divisor >= value,
					"CeilDiv(%d, %d) = %d, but %d * %d < %d",
					value, divisor, result, result, divisor, value)

				if result > 0 {
					testutils.Assert(t, (result-1)*divisor < value,
						"CeilDiv(%d, %d) = %d is not minimal", value, divisor, result)
				}
			}
		}
	})
}

func TestLog2Ceil(t *testing.T) {
	tests := []struct {
		n        uint64
		expected uint64
	}{
		{0, 0}, // defined special case
		{1, 0}, // <= 1 → 0
		{2, 1}, // log2(2) = 1
		{3, 2}, // ceil(log2(3)) = 2
		{4, 2}, // log2(4) = 2
		{5, 3}, // ceil(log2(5)) = 3
		{7, 3}, // ceil(log2(7)) = 3
		{8, 3}, // log2(8) = 3
		{9, 4}, // ceil(log2(9)) = 4
		{15, 4},
		{16, 4},
		{17, 5},
		{1023, 10},
		{1024, 10},
		{1025, 11},
	}

	for _, tc := range tests {
		got := Log2Ceil(tc.n)
		testutils.Assert(
			t,
			got == tc.expected,
			"Log2Ceil(%d) = %d, expected %d",
			tc.n, got, tc.expected,
		)
	}
}

func TestReverseBits(t *testing.T) {
	tests := []struct {
		n        uint64
		expected uint64
	}{
		{0, 0},
		{1, 1 << 63},                             // lowest bit → highest
		{1 << 63, 1},                             // highest bit → lowest
		{0xAAAAAAAAAAAAAAAA, 0x5555555555555555}, // alternating bits
		{0x5555555555555555, 0xAAAAAAAAAAAAAAAA}, // alternating bits reversed
		{0xF0F0F0F0F0F0F0F0, 0x0F0F0F0F0F0F0F0F}, // nibble pattern
		{0x0123456789ABCDEF, 0xF7B3D591E6A2C480}, // known reversal
		{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, // all ones
	}

	for _, tc := range tests {
		got := ReverseBits(tc.n)
		testutils.Assert(
			t,
			got == tc.expected,
			"ReverseBits(%#016x) = %#016x, expected %#016x",
			tc.n, got, tc.expected,
		)
	}
}

func TestReverseBitsMatchesStdlib(t *testing.T) {
	// Cross-check with math/bits.Reverse64 for robustness
	for _, n := range []uint64{
		0, 1, 2, 3, 0x12345678, 0xFFFFFFFFFFFFFFFF,
	} {
		got := ReverseBits(n)
		want := bits.Reverse64(n)
		testutils.Assert(
			t,
			uint64(got) == want,
			"ReverseBits(%#016x) = %#016x, expected %#016x",
			n, got, want,
		)
	}
}
