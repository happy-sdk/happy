// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy SDK Authors

package bitutils

import (
	"errors"
	"fmt"
	"math"
	"math/bits"
)

var (
	Error           = errors.New("bitutils")
	ErrAlignment    = fmt.Errorf("%w:alignment", Error)
	ErrDivideByZero = fmt.Errorf("%w division by zero", Error)
)

// NextPowerOfTwo returns the next power of 2 >= n.
func NextPowerOfTwo(n uint64) uint64 {
	if n == 0 {
		return 1
	}

	// Handle the largest valid power of 2 (1<<63) as a special case
	// because the bit manipulation algorithm would overflow
	if n == (1 << 63) {
		return 1 << 63
	}

	// Handle overflow cases - anything larger than 1<<63
	if n > (1 << 63) {
		return math.MaxUint
	}

	// Standard bit manipulation algorithm for values < (1<<63)
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	n++
	return n
}

// ClearLowestBit clears the least significant set bit in n and returns the result.
func ClearLowestBit(n uint64) uint64 {
	return n & (n - 1)
}

// FindLowestBit returns the index of the least significant set bit in n (0-based).
// Returns 0 if n is 0.
func FindLowestBit(n uint64) uint64 {
	if n == 0 {
		return 0
	}
	return uint64(bits.TrailingZeros64(n))
}

// FastMod computes n % m efficiently when m is a power of 2.
func FastMod(n, m uint64) (uint64, error) {
	if m&(m-1) != 0 {
		return 0, fmt.Errorf("%w: FastMod: m must be a power of 2", Error)
	}
	return n & (m - 1), nil
}

// AlignUp rounds n up to the next multiple of align, where align is a power of 2.
func AlignUp(n, align uint64) (uint64, error) {
	if align == 0 || align&(align-1) != 0 {
		return 0, fmt.Errorf("%w: AlignUp: align must be a power of 2", ErrAlignment)
	}
	return (n + align - 1) &^ (align - 1), nil
}

// CeilDiv computes the ceiling of a/b without floating-point arithmetic.
func CeilDiv(a, b uint64) (uint64, error) {
	if b == 0 {
		return 0, ErrDivideByZero
	}
	return (a + b - 1) / b, nil
}

// Log2Ceil returns the ceiling of the base-2 logarithm of n.
// Returns 0 if n is 0.
func Log2Ceil(n uint64) uint64 {
	if n <= 1 {
		return 0
	}
	return uint64(bits.Len64(n - 1))
}

// ReverseBits reverses the bits of n.
const (
	mask1  = 0x5555555555555555
	mask2  = 0x3333333333333333
	mask4  = 0x0F0F0F0F0F0F0F0F
	mask8  = 0x00FF00FF00FF00FF
	mask16 = 0x0000FFFF0000FFFF
)

// ReverseBits reverses the bits of a 64-bit unsigned integer.
func ReverseBits(n uint64) uint64 {
	n = ((n >> 1) & mask1) | ((n & mask1) << 1)
	n = ((n >> 2) & mask2) | ((n & mask2) << 2)
	n = ((n >> 4) & mask4) | ((n & mask4) << 4)
	n = ((n >> 8) & mask8) | ((n & mask8) << 8)
	n = ((n >> 16) & mask16) | ((n & mask16) << 16)
	n = (n >> 32) | (n << 32)
	return n
}
