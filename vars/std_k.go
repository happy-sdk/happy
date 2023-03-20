// Copyright 2022 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package vars

import (
	"errors"
	"fmt"
	"sync"
)

func bug(msg string, args ...any) {
	panic(fmt.Sprintf(msg, args...))
}

const fastSmalls = true // enable fast path for small integers

const nSmalls = 100

const maxUint64 = 1<<64 - 1

func bitSizeError(fn, str string, bitSize int) *NumError {
	return &NumError{fn, cloneString(str), errors.New("invalid bit size " + formatIntFast(int64(bitSize), 10))}
}

func baseError(fn, str string, base int) *NumError {
	return &NumError{fn, cloneString(str), errors.New("invalid base " + formatIntFast(int64(base), 10))}
}

// FLOATS
// TODO: move elsewhere?

type decimalSlice struct {
	d      []byte
	nd, dp int
	neg    bool
}

// %b: -ddddddddp±ddd
func fmtB(dst []byte, neg bool, mant uint64, exp int, flt *floatInfo) []byte {
	// sign
	if neg {
		dst = append(dst, '-')
	}

	// mantissa
	dst, _ = formatBits(dst, mant, 10, false, true)

	// p
	dst = append(dst, 'p')

	// ±exponent
	exp -= int(flt.mantbits)
	if exp >= 0 {
		bug("fmtB")
		dst = append(dst, '+')
	}
	dst, _ = formatBits(dst, uint64(exp), 10, exp < 0, true)

	return dst
}

const (
	lowerhex = "0123456789abcdef"
	upperhex = "0123456789ABCDEF"
)

// ryuFtoaShortest formats mant*2^exp with prec decimal digits.
func ryuFtoaShortest(d *decimalSlice, mant uint64, exp int, flt *floatInfo) {
	if mant == 0 {
		d.nd, d.dp = 0, 0
		return
	}
	// If input is an exact integer with fewer bits than the mantissa,
	// the previous and next integer are not admissible representations.
	if exp <= 0 && bitsTrailingZeros64(mant) >= -exp {
		mant >>= uint(-exp)
		ryuDigits(d, mant, mant, mant, true, false)
		return
	}
	ml, mc, mu, e2 := computeBounds(mant, exp, flt)
	if e2 == 0 {
		ryuDigits(d, ml, mc, mu, true, false)
		return
	}
	// Find 10^q *larger* than 2^-e2
	q := mulByLog2Log10(-e2) + 1

	// We are going to multiply by 10^q using 128-bit arithmetic.
	// The exponent is the same for all 3 numbers.
	var dl, dc, du uint64
	var dl0, dc0, du0 bool
	if flt == &float32info {
		var dl32, dc32, du32 uint32
		dl32, _, dl0 = mult64bitPow10(uint32(ml), e2, q)
		dc32, _, dc0 = mult64bitPow10(uint32(mc), e2, q)
		du32, e2, du0 = mult64bitPow10(uint32(mu), e2, q)
		dl, dc, du = uint64(dl32), uint64(dc32), uint64(du32)
	} else {
		dl, _, dl0 = mult128bitPow10(ml, e2, q)
		dc, _, dc0 = mult128bitPow10(mc, e2, q)
		du, e2, du0 = mult128bitPow10(mu, e2, q)
	}
	if e2 >= 0 {
		bug("not enough significant bits after mult128bitPow10")
	}
	// Is it an exact computation?
	if q > 55 {
		// Large positive powers of ten are not exact
		dl0, dc0, du0 = false, false, false
	}
	if q < 0 && q >= -24 {
		// Division by a power of ten may be exact.
		// (note that 5^25 is a 59-bit number so division by 5^25 is never exact).
		if divisibleByPower5(ml, -q) {
			dl0 = true
		}
		if divisibleByPower5(mc, -q) {
			dc0 = true
		}
		if divisibleByPower5(mu, -q) {
			du0 = true
		}
	}
	// Express the results (dl, dc, du)*2^e2 as integers.
	// Extra bits must be removed and rounding hints computed.
	extra := uint(-e2)
	extraMask := uint64(1<<extra - 1)
	// Now compute the floored, integral base 10 mantissas.
	dl, fracl := dl>>extra, dl&extraMask
	dc, fracc := dc>>extra, dc&extraMask
	du, fracu := du>>extra, du&extraMask
	// Is it allowed to use 'du' as a result?
	// It is always allowed when it is truncated, but also
	// if it is exact and the original binary mantissa is even
	// When disallowed, we can subtract 1.
	uok := !du0 || fracu > 0
	if du0 && fracu == 0 {
		uok = mant&1 == 0
	}
	if !uok {
		du--
	}
	// Is 'dc' the correctly rounded base 10 mantissa?
	// The correct rounding might be dc+1
	cup := false // don't round up.
	if dc0 {
		// If we computed an exact product, the half integer
		// should round to next (even) integer if 'dc' is odd.
		cup = fracc > 1<<(extra-1) ||
			(fracc == 1<<(extra-1) && dc&1 == 1)
	} else {
		// otherwise, the result is a lower truncation of the ideal
		// result.
		cup = fracc>>(extra-1) == 1
	}
	// Is 'dl' an allowed representation?
	// Only if it is an exact value, and if the original binary mantissa
	// was even.
	lok := dl0 && fracl == 0 && (mant&1 == 0)
	if !lok {
		dl++
	}
	// We need to remember whether the trimmed digits of 'dc' are zero.
	c0 := dc0 && fracc == 0
	// render digits
	ryuDigits(d, dl, dc, du, c0, cup)
	d.dp -= q
}

func divisibleByPower5(m uint64, k int) bool {
	if m == 0 {
		bug("divisibleByPower5")
		return true
	}
	for i := 0; i < k; i++ {
		if m%5 != 0 {
			return false
		}
		m /= 5
	}
	return true
}

const smallsString = "00010203040506070809" +
	"10111213141516171819" +
	"20212223242526272829" +
	"30313233343536373839" +
	"40414243444546474849" +
	"50515253545556575859" +
	"60616263646566676869" +
	"70717273747576777879" +
	"80818283848586878889" +
	"90919293949596979899"

// binary to decimal conversion using the Ryū algorithm.
//
// See Ulf Adams, "Ryū: Fast Float-to-String Conversion" (doi:10.1145/3192366.3192369)
//
// Fixed precision formatting is a variant of the original paper's
// algorithm, where a single multiplication by 10^k is required,
// sharing the same rounding guarantees.

// ryuFtoaFixed32 formats mant*(2^exp) with prec decimal digits.
func ryuFtoaFixed32(d *decimalSlice, mant uint32, exp int, prec int) {
	if prec < 0 {
		panic("ryuFtoaFixed32 called with negative prec")
	}
	if prec > 9 {
		panic("ryuFtoaFixed32 called with prec > 9")
	}
	// Zero input.
	if mant == 0 {
		d.nd, d.dp = 0, 0
		return
	}
	// Renormalize to a 25-bit mantissa.
	e2 := exp
	if b := bitsLen32(mant); b < 25 {
		mant <<= uint(25 - b)
		e2 += b - 25
	}
	// Choose an exponent such that rounded mant*(2^e2)*(10^q) has
	// at least prec decimal digits, i.e
	//     mant*(2^e2)*(10^q) >= 10^(prec-1)
	// Because mant >= 2^24, it is enough to choose:
	//     2^(e2+24) >= 10^(-q+prec-1)
	// or q = -mulByLog2Log10(e2+24) + prec - 1
	q := -mulByLog2Log10(e2+24) + prec - 1

	// Now compute mant*(2^e2)*(10^q).
	// Is it an exact computation?
	// Only small positive powers of 10 are exact (5^28 has 66 bits).
	exact := q <= 27 && q >= 0

	di, dexp2, d0 := mult64bitPow10(mant, e2, q)
	if dexp2 >= 0 {
		panic("not enough significant bits after mult64bitPow10")
	}
	// As a special case, computation might still be exact, if exponent
	// was negative and if it amounts to computing an exact division.
	// In that case, we ignore all lower bits.
	// Note that division by 10^11 cannot be exact as 5^11 has 26 bits.
	if q < 0 && q >= -10 && divisibleByPower5(uint64(mant), -q) {
		exact = true
		d0 = true
	}
	// Remove extra lower bits and keep rounding info.
	extra := uint(-dexp2)
	extraMask := uint32(1<<extra - 1)

	di, dfrac := di>>extra, di&extraMask
	roundUp := false
	if exact {
		// If we computed an exact product, d + 1/2
		// should round to d+1 if 'd' is odd.
		roundUp = dfrac > 1<<(extra-1) ||
			(dfrac == 1<<(extra-1) && !d0) ||
			(dfrac == 1<<(extra-1) && d0 && di&1 == 1)
	} else {
		// otherwise, d+1/2 always rounds up because
		// we truncated below.
		roundUp = dfrac>>(extra-1) == 1
	}
	if dfrac != 0 {
		d0 = false
	}
	// Proceed to the requested number of digits
	formatDecimal(d, uint64(di), !d0, roundUp, prec)
	// Adjust exponent
	d.dp -= q
}

// Binary shift left (* 2) by k bits.  k <= maxShift to avoid overflow.
func leftShift(a *decimal, k uint) {
	delta := leftcheats[k].delta
	if prefixIsLessThan(a.d[0:a.nd], leftcheats[k].cutoff) {
		delta--
	}

	r := a.nd         // read index
	w := a.nd + delta // write index

	// Pick up a digit, put down a digit.
	var n uint
	for r--; r >= 0; r-- {
		n += (uint(a.d[r]) - '0') << k
		quo := n / 10
		rem := n - 10*quo
		w--
		if w < len(a.d) {
			a.d[w] = byte(rem + '0')
		} else if rem != 0 {
			a.trunc = true
		}
		n = quo
	}

	// Put down extra digits.
	for n > 0 {
		quo := n / 10
		rem := n - 10*quo
		w--
		if w < len(a.d) {
			a.d[w] = byte(rem + '0')
		} else if rem != 0 {
			bug("leftShift")
			a.trunc = true
		}
		n = quo
	}

	a.nd += delta
	if a.nd >= len(a.d) {
		a.nd = len(a.d)
	}
	a.dp += delta
	trim(a)
}

// roundShortest rounds d (= mant * 2^exp) to the shortest number of digits
// that will let the original floating point value be precisely reconstructed.
func roundShortest(d *decimal, mant uint64, exp int, flt *floatInfo) {
	// If mantissa is zero, the number is zero; stop now.
	if mant == 0 {
		d.nd = 0
		return
	}

	// Compute upper and lower such that any decimal number
	// between upper and lower (possibly inclusive)
	// will round to the original floating point number.

	// We may see at once that the number is already shortest.
	//
	// Suppose d is not denormal, so that 2^exp <= d < 10^dp.
	// The closest shorter number is at least 10^(dp-nd) away.
	// The lower/upper bounds computed below are at distance
	// at most 2^(exp-mantbits).
	//
	// So the number is already shortest if 10^(dp-nd) > 2^(exp-mantbits),
	// or equivalently log2(10)*(dp-nd) > exp-mantbits.
	// It is true if 332/100*(dp-nd) >= exp-mantbits (log2(10) > 3.32).
	minexp := flt.bias + 1 // minimum possible exponent
	if exp > minexp && 332*(d.dp-d.nd) >= 100*(exp-int(flt.mantbits)) {
		// The number is already shortest.
		return
	}

	// d = mant << (exp - mantbits)
	// Next highest floating point number is mant+1 << exp-mantbits.
	// Our upper bound is halfway between, mant*2+1 << exp-mantbits-1.
	upper := new(decimal)
	upper.Assign(mant*2 + 1)
	upper.Shift(exp - int(flt.mantbits) - 1)

	// d = mant << (exp - mantbits)
	// Next lowest floating point number is mant-1 << exp-mantbits,
	// unless mant-1 drops the significant bit and exp is not the minimum exp,
	// in which case the next lowest is mant*2-1 << exp-mantbits-1.
	// Either way, call it mantlo << explo-mantbits.
	// Our lower bound is halfway between, mantlo*2+1 << explo-mantbits-1.
	var mantlo uint64
	var explo int
	if mant > 1<<flt.mantbits || exp == minexp {
		mantlo = mant - 1
		explo = exp
	} else {
		mantlo = mant*2 - 1
		explo = exp - 1
	}
	lower := new(decimal)
	lower.Assign(mantlo*2 + 1)
	lower.Shift(explo - int(flt.mantbits) - 1)

	// The upper and lower bounds are possible outputs only if
	// the original mantissa is even, so that IEEE round-to-even
	// would round to the original mantissa and not the neighbors.
	inclusive := mant%2 == 0

	// As we walk the digits we want to know whether rounding up would fall
	// within the upper bound. This is tracked by upperdelta:
	//
	// If upperdelta == 0, the digits of d and upper are the same so far.
	//
	// If upperdelta == 1, we saw a difference of 1 between d and upper on a
	// previous digit and subsequently only 9s for d and 0s for upper.
	// (Thus rounding up may fall outside the bound, if it is exclusive.)
	//
	// If upperdelta == 2, then the difference is greater than 1
	// and we know that rounding up falls within the bound.
	var upperdelta uint8

	// Now we can figure out the minimum number of digits required.
	// Walk along until d has distinguished itself from upper and lower.
	for ui := 0; ; ui++ {
		// lower, d, and upper may have the decimal points at different
		// places. In this case upper is the longest, so we iterate from
		// ui==0 and start li and mi at (possibly) -1.
		mi := ui - upper.dp + d.dp
		if mi >= d.nd {
			bug("roundShortest")
			break
		}
		li := ui - upper.dp + lower.dp
		l := byte('0') // lower digit
		if li >= 0 && li < lower.nd {
			l = lower.d[li]
		}
		m := byte('0') // middle digit
		if mi >= 0 {
			m = d.d[mi]
		}
		u := byte('0') // upper digit
		if ui < upper.nd {
			u = upper.d[ui]
		}

		// Okay to round down (truncate) if lower has a different digit
		// or if lower is inclusive and is exactly the result of rounding
		// down (i.e., and we have reached the final digit of lower).
		okdown := l != m || inclusive && li+1 == lower.nd

		switch {
		case upperdelta == 0 && m+1 < u:
			// Example:
			// m = 12345xxx
			// u = 12347xxx
			upperdelta = 2
		case upperdelta == 0 && m != u:
			// Example:
			// m = 12345xxx
			// u = 12346xxx
			upperdelta = 1
		case upperdelta == 1 && (m != '9' || u != '0'):
			// Example:
			// m = 1234598x
			// u = 1234600x
			upperdelta = 2
		}
		// Okay to round up if upper has a different digit and either upper
		// is inclusive or upper is bigger than the result of rounding up.
		okup := upperdelta > 0 && (inclusive || upperdelta > 1 || ui+1 < upper.nd)

		// If it's okay to do either, then round to the nearest one.
		// If it's okay to do only one, do it.
		switch {
		case okdown && okup:
			d.Round(mi + 1)
			return
		case okdown:
			d.RoundDown(mi + 1)
			return
		case okup:
			d.RoundUp(mi + 1)
			return
		}
	}
}

var uint64pow10 = [...]uint64{
	1, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9,
	1e10, 1e11, 1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19,
}

// ryuFtoaFixed64 formats mant*(2^exp) with prec decimal digits.
func ryuFtoaFixed64(d *decimalSlice, mant uint64, exp int, prec int) {
	if prec > 18 {
		panic("ryuFtoaFixed64 called with prec > 18")
	}
	// Zero input.
	if mant == 0 {
		d.nd, d.dp = 0, 0
		return
	}
	// Renormalize to a 55-bit mantissa.
	e2 := exp
	if b := bitsLen64(mant); b < 55 {
		mant = mant << uint(55-b)
		e2 += b - 55
	}
	// Choose an exponent such that rounded mant*(2^e2)*(10^q) has
	// at least prec decimal digits, i.e
	//     mant*(2^e2)*(10^q) >= 10^(prec-1)
	// Because mant >= 2^54, it is enough to choose:
	//     2^(e2+54) >= 10^(-q+prec-1)
	// or q = -mulByLog2Log10(e2+54) + prec - 1
	//
	// The minimal required exponent is -mulByLog2Log10(1025)+18 = -291
	// The maximal required exponent is mulByLog2Log10(1074)+18 = 342
	q := -mulByLog2Log10(e2+54) + prec - 1

	// Now compute mant*(2^e2)*(10^q).
	// Is it an exact computation?
	// Only small positive powers of 10 are exact (5^55 has 128 bits).
	exact := q <= 55 && q >= 0

	di, dexp2, d0 := mult128bitPow10(mant, e2, q)
	if dexp2 >= 0 {
		panic("not enough significant bits after mult128bitPow10")
	}
	// As a special case, computation might still be exact, if exponent
	// was negative and if it amounts to computing an exact division.
	// In that case, we ignore all lower bits.
	// Note that division by 10^23 cannot be exact as 5^23 has 54 bits.
	if q < 0 && q >= -22 && divisibleByPower5(mant, -q) {
		exact = true
		d0 = true
	}
	// Remove extra lower bits and keep rounding info.
	extra := uint(-dexp2)
	extraMask := uint64(1<<extra - 1)

	di, dfrac := di>>extra, di&extraMask
	roundUp := false
	if exact {
		// If we computed an exact product, d + 1/2
		// should round to d+1 if 'd' is odd.
		roundUp = dfrac > 1<<(extra-1) ||
			(dfrac == 1<<(extra-1) && !d0) ||
			(dfrac == 1<<(extra-1) && d0 && di&1 == 1)
	} else {
		// otherwise, d+1/2 always rounds up because
		// we truncated below.
		roundUp = dfrac>>(extra-1) == 1
	}
	if dfrac != 0 {
		d0 = false
	}
	// Proceed to the requested number of digits
	formatDecimal(d, di, !d0, roundUp, prec)
	// Adjust exponent
	d.dp -= q
}

// detailedPowersOfTen{Min,Max}Exp10 is the power of 10 represented by the
// first and last rows of detailedPowersOfTen. Both bounds are inclusive.
const (
	detailedPowersOfTenMinExp10 = -348
	detailedPowersOfTenMaxExp10 = +347
)

// mult64bitPow10 takes a floating-point input with a 25-bit
// mantissa and multiplies it with 10^q. The resulting mantissa
// is m*P >> 57 where P is a 64-bit element of the detailedPowersOfTen tables.
// It is typically 31 or 32-bit wide.
// The returned boolean is true if all trimmed bits were zero.
//
// That is:
//
//	m*2^e2 * round(10^q) = resM * 2^resE + ε
//	exact = ε == 0
func mult64bitPow10(m uint32, e2, q int) (resM uint32, resE int, exact bool) {
	if q == 0 {
		// P == 1<<63
		return m << 6, e2 - 6, true
	}
	if q < detailedPowersOfTenMinExp10 || detailedPowersOfTenMaxExp10 < q {
		// This never happens due to the range of float32/float64 exponent
		panic("mult64bitPow10: power of 10 is out of range")
	}
	pow := detailedPowersOfTen[q-detailedPowersOfTenMinExp10][1]
	if q < 0 {
		// Inverse powers of ten must be rounded up.
		pow += 1
	}
	hi, lo := bitsMul64(uint64(m), pow)
	e2 += mulByLog10Log2(q) - 63 + 57
	return uint32(hi<<7 | lo>>57), e2, lo<<7 == 0
}

// mult128bitPow10 takes a floating-point input with a 55-bit
// mantissa and multiplies it with 10^q. The resulting mantissa
// is m*P >> 119 where P is a 128-bit element of the detailedPowersOfTen tables.
// It is typically 63 or 64-bit wide.
// The returned boolean is true is all trimmed bits were zero.
//
// That is:
//
//	m*2^e2 * round(10^q) = resM * 2^resE + ε
//	exact = ε == 0
func mult128bitPow10(m uint64, e2, q int) (resM uint64, resE int, exact bool) {
	if q == 0 {
		// P == 1<<127
		return m << 8, e2 - 8, true
	}
	if q < detailedPowersOfTenMinExp10 || detailedPowersOfTenMaxExp10 < q {
		// This never happens due to the range of float32/float64 exponent
		panic("mult128bitPow10: power of 10 is out of range")
	}
	pow := detailedPowersOfTen[q-detailedPowersOfTenMinExp10]
	if q < 0 {
		// Inverse powers of ten must be rounded up.
		pow[0] += 1
	}
	e2 += mulByLog10Log2(q) - 127 + 119

	// long multiplication
	l1, l0 := bitsMul64(m, pow[0])
	h1, h0 := bitsMul64(m, pow[1])
	mid, carry := bitsAdd64(l1, h0, 0)
	h1 += carry
	return h1<<9 | mid>>55, e2, mid<<9 == 0 && l0 == 0
}

func atof64(s string) (f float64, n int, err error) {
	if val, n, ok := special(s); ok {
		return val, n, nil
	}

	mantissa, exp, neg, trunc, hex, n, ok := readFloat(s)
	if !ok {
		return 0, n, syntaxError(fnParseFloat, s)
	}

	if hex {
		f, err := atofHex(s[:n], &float64info, mantissa, exp, neg, trunc)
		return f, n, err
	}

	if optimize {
		// Try pure floating-point arithmetic conversion, and if that fails,
		// the Eisel-Lemire algorithm.
		if !trunc {
			if f, ok := atof64exact(mantissa, exp, neg); ok {
				return f, n, nil
			}
		}
		f, ok := eiselLemire64(mantissa, exp, neg)
		if ok {
			if !trunc {
				return f, n, nil
			}
			// Even if the mantissa was truncated, we may
			// have found the correct result. Confirm by
			// converting the upper mantissa bound.
			fUp, ok := eiselLemire64(mantissa+1, exp, neg)
			if ok && f == fUp {
				return f, n, nil
			}
		}
	}

	// Slow fallback.
	var d decimal
	if !d.set(s[:n]) {
		bug("atof64")
		return 0, n, syntaxError(fnParseFloat, s)
	}
	b, ovf := d.floatBits(&float64info)
	f = mathFloat64frombits(b)
	if ovf {
		err = rangeError(fnParseFloat, s)
	}
	return f, n, err
}

func (b *decimal) set(s string) (ok bool) {
	i := 0
	b.neg = false
	b.trunc = false

	// optional sign
	if i >= len(s) {
		bug("decimal.set")
		return
	}
	switch {
	case s[i] == '+':
		i++
	case s[i] == '-':
		b.neg = true
		i++
	}

	// digits
	sawdot := false
	sawdigits := false
	for ; i < len(s); i++ {
		switch {
		case s[i] == '_':
			// readFloat already checked underscores
			continue
		case s[i] == '.':
			if sawdot {
				bug("decimal.set")
				return
			}
			sawdot = true
			b.dp = b.nd
			continue

		case '0' <= s[i] && s[i] <= '9':
			sawdigits = true
			if s[i] == '0' && b.nd == 0 { // ignore leading zeros
				b.dp--
				continue
			}
			if b.nd < len(b.d) {
				b.d[b.nd] = s[i]
				b.nd++
			} else if s[i] != '0' {
				b.trunc = true
			}
			continue
		}
		break
	}
	if !sawdigits {
		bug("decimal.set")
		return
	}
	if !sawdot {
		b.dp = b.nd
	}

	// optional exponent moves decimal point.
	// if we read a very large, very long number,
	// just be sure to move the decimal point by
	// a lot (say, 100000).  it doesn't matter if it's
	// not the exact number.
	if i < len(s) && lower(s[i]) == 'e' {
		i++
		if i >= len(s) {
			bug("decimal.set")
			return
		}
		esign := 1
		if s[i] == '+' {
			i++
		} else if s[i] == '-' {
			i++
			esign = -1
		}
		if i >= len(s) || s[i] < '0' || s[i] > '9' {
			bug("decimal.set")
			return
		}
		e := 0
		for ; i < len(s) && ('0' <= s[i] && s[i] <= '9' || s[i] == '_'); i++ {
			if s[i] == '_' {
				// readFloat already checked underscores
				continue
			}
			if e < 10000 {
				e = e*10 + int(s[i]) - '0'
			}
		}
		b.dp += e * esign
	}

	if i != len(s) {
		bug("decimal.set")
		return
	}

	ok = true
	return
}

func eiselLemire32(man uint64, exp10 int, neg bool) (f float32, ok bool) {
	// The terse comments in this function body refer to sections of the
	// https://nigeltao.github.io/blog/2020/eisel-lemire.html blog post.
	//
	// That blog post discusses the float64 flavor (11 exponent bits with a
	// -1023 bias, 52 mantissa bits) of the algorithm, but the same approach
	// applies to the float32 flavor (8 exponent bits with a -127 bias, 23
	// mantissa bits). The computation here happens with 64-bit values (e.g.
	// man, xHi, retMantissa) before finally converting to a 32-bit float.

	// Exp10 Range.
	if man == 0 {
		bug("eiselLemire32")
		if neg {
			f = mathFloat32frombits(0x80000000) // Negative zero.
		}
		return f, true
	}
	if exp10 < detailedPowersOfTenMinExp10 || detailedPowersOfTenMaxExp10 < exp10 {
		return 0, false
	}

	// Normalization.
	clz := bitsLeadingZeros64(man)
	man <<= uint(clz)
	const float32ExponentBias = 127
	retExp2 := uint64(217706*exp10>>16+64+float32ExponentBias) - uint64(clz)

	// Multiplication.
	xHi, xLo := bitsMul64(man, detailedPowersOfTen[exp10-detailedPowersOfTenMinExp10][1])

	// Wider Approximation.
	if xHi&0x3FFFFFFFFF == 0x3FFFFFFFFF && xLo+man < man {
		yHi, yLo := bitsMul64(man, detailedPowersOfTen[exp10-detailedPowersOfTenMinExp10][0])
		mergedHi, mergedLo := xHi, xLo+yHi
		if mergedLo < xLo {
			bug("eiselLemire32")
			mergedHi++
		}
		if mergedHi&0x3FFFFFFFFF == 0x3FFFFFFFFF && mergedLo+1 == 0 && yLo+man < man {
			return 0, false
		}
		xHi, xLo = mergedHi, mergedLo
	}

	// Shifting to 54 Bits (and for float32, it's shifting to 25 bits).
	msb := xHi >> 63
	retMantissa := xHi >> (msb + 38)
	retExp2 -= 1 ^ msb

	// Half-way Ambiguity.
	if xLo == 0 && xHi&0x3FFFFFFFFF == 0 && retMantissa&3 == 1 {
		return 0, false
	}

	// From 54 to 53 Bits (and for float32, it's from 25 to 24 bits).
	retMantissa += retMantissa & 1
	retMantissa >>= 1
	if retMantissa>>24 > 0 {
		retMantissa >>= 1
		retExp2 += 1
	}
	// retExp2 is a uint64. Zero or underflow means that we're in subnormal
	// float32 space. 0xFF or above means that we're in Inf/NaN float32 space.
	//
	// The if block is equivalent to (but has fewer branches than):
	//   if retExp2 <= 0 || retExp2 >= 0xFF { etc }
	if retExp2-1 >= 0xFF-1 {
		return 0, false
	}
	retBits := retExp2<<23 | retMantissa&0x007FFFFF
	if neg {
		retBits |= 0x80000000
	}
	return mathFloat32frombits(uint32(retBits)), true
}

// decimal power of ten to binary power of two.
var powtab = []int{1, 3, 6, 9, 13, 16, 19, 23, 26}

func eiselLemire64(man uint64, exp10 int, neg bool) (f float64, ok bool) {
	// The terse comments in this function body refer to sections of the
	// https://nigeltao.github.io/blog/2020/eisel-lemire.html blog post.

	// Exp10 Range.
	if man == 0 {
		bug("eiselLemire64")
		if neg {
			f = mathFloat64frombits(0x8000000000000000) // Negative zero.
		}
		return f, true
	}
	if exp10 < detailedPowersOfTenMinExp10 || detailedPowersOfTenMaxExp10 < exp10 {
		return 0, false
	}

	// Normalization.
	clz := bitsLeadingZeros64(man)
	man <<= uint(clz)
	const float64ExponentBias = 1023
	retExp2 := uint64(217706*exp10>>16+64+float64ExponentBias) - uint64(clz)

	// Multiplication.
	xHi, xLo := bitsMul64(man, detailedPowersOfTen[exp10-detailedPowersOfTenMinExp10][1])

	// Wider Approximation.
	if xHi&0x1FF == 0x1FF && xLo+man < man {
		yHi, yLo := bitsMul64(man, detailedPowersOfTen[exp10-detailedPowersOfTenMinExp10][0])
		mergedHi, mergedLo := xHi, xLo+yHi
		if mergedLo < xLo {
			mergedHi++
		}
		if mergedHi&0x1FF == 0x1FF && mergedLo+1 == 0 && yLo+man < man {
			return 0, false
		}
		xHi, xLo = mergedHi, mergedLo
	}

	// Shifting to 54 Bits.
	msb := xHi >> 63
	retMantissa := xHi >> (msb + 9)
	retExp2 -= 1 ^ msb

	// Half-way Ambiguity.
	if xLo == 0 && xHi&0x1FF == 0 && retMantissa&3 == 1 {
		return 0, false
	}

	// From 54 to 53 Bits.
	retMantissa += retMantissa & 1
	retMantissa >>= 1
	if retMantissa>>53 > 0 {
		retMantissa >>= 1
		retExp2 += 1
	}
	// retExp2 is a uint64. Zero or underflow means that we're in subnormal
	// float64 space. 0x7FF or above means that we're in Inf/NaN float64 space.
	//
	// The if block is equivalent to (but has fewer branches than):
	//   if retExp2 <= 0 || retExp2 >= 0x7FF { etc }
	if retExp2-1 >= 0x7FF-1 {
		return 0, false
	}
	retBits := retExp2<<52 | retMantissa&0x000FFFFFFFFFFFFF
	if neg {
		retBits |= 0x8000000000000000
	}
	return mathFloat64frombits(retBits), true
}

const fnParseFloat = "ParseFloat"

func atof32(s string) (f float32, n int, err error) {
	if val, n, ok := special(s); ok {
		return float32(val), n, nil
	}

	mantissa, exp, neg, trunc, hex, n, ok := readFloat(s)
	if !ok {
		return 0, n, syntaxError(fnParseFloat, s)
	}

	if hex {
		f, err := atofHex(s[:n], &float32info, mantissa, exp, neg, trunc)
		return float32(f), n, err
	}

	if optimize {
		// Try pure floating-point arithmetic conversion, and if that fails,
		// the Eisel-Lemire algorithm.
		if !trunc {
			if f, ok := atof32exact(mantissa, exp, neg); ok {
				return f, n, nil
			}
		}
		f, ok := eiselLemire32(mantissa, exp, neg)
		if ok {
			if !trunc {
				return f, n, nil
			}
			// Even if the mantissa was truncated, we may
			// have found the correct result. Confirm by
			// converting the upper mantissa bound.
			fUp, ok := eiselLemire32(mantissa+1, exp, neg)
			if ok && f == fUp {
				return f, n, nil
			}
		}
	}

	// Slow fallback.
	var d decimal
	if !d.set(s[:n]) {
		bug("atof32 %s", s)
		return 0, n, syntaxError(fnParseFloat, s)
	}
	b, ovf := d.floatBits(&float32info)
	f = mathFloat32frombits(uint32(b))
	if ovf {
		err = rangeError(fnParseFloat, s)
	}
	return f, n, err
}

const (
	utf8RuneError = '\uFFFD' // the "error" Rune or "Unicode replacement character"
	utf8RuneSelf  = 0x80     // characters below RuneSelf are represented as themselves in a single byte.
	utf8UTFMax    = 4        // maximum number of bytes of a UTF-8 encoded Unicode character.
	utf8MaxRune   = '\U0010FFFF'

	// These names of these constants are chosen to give nice alignment in the
	// table below. The first nibble is an index into acceptRanges or F for
	// special one-byte cases. The second nibble is the Rune length or the
	// Status for the special one-byte case.
	xx = 0xF1 // invalid: size 1
	as = 0xF0 // ASCII: size 1
	s1 = 0x02 // accept 0, size 2
	s2 = 0x13 // accept 1, size 3
	s3 = 0x03 // accept 0, size 3
	s4 = 0x23 // accept 2, size 3
	s5 = 0x34 // accept 3, size 4
	s6 = 0x04 // accept 0, size 4
	s7 = 0x44 // accept 4, size 4

	// The default lowest and highest continuation byte.
	locb = 0b10000000
	hicb = 0b10111111

	maskx = 0b00111111
	mask2 = 0b00011111
	mask3 = 0b00001111
	mask4 = 0b00000111
)

// DecodeRuneInString is like DecodeRune but its input is a string. If s is
// empty it returns (RuneError, 0). Otherwise, if the encoding is invalid, it
// returns (RuneError, 1). Both are impossible results for correct, non-empty
// UTF-8.
//
// An encoding is invalid if it is incorrect UTF-8, encodes a rune that is
// out of range, or is not the shortest possible UTF-8 encoding for the
// value. No other validation is performed.
func utf8DecodeRuneInString(s string) (r rune, size int) {
	n := len(s)
	if n < 1 {
		bug("utf8DecodeRuneInString")
		return utf8RuneError, 0
	}
	s0 := s[0]
	x := utf8first[s0]
	if x >= as {
		// The following code simulates an additional check for x == xx and
		// handling the ASCII and invalid cases accordingly. This mask-and-or
		// approach prevents an additional branch.
		mask := rune(x) << 31 >> 31 // Create 0x0000 or 0xFFFF.
		return rune(s[0])&^mask | utf8RuneError&mask, 1
	}
	sz := int(x & 7)
	accept := utf8AcceptRanges[x>>4]
	if n < sz {
		return utf8RuneError, 1
	}
	s1 := s[1]
	if s1 < accept.lo || accept.hi < s1 {
		return utf8RuneError, 1
	}
	if sz <= 2 { // <= instead of == to help the compiler eliminate some bounds checks
		return rune(s0&mask2)<<6 | rune(s1&maskx), 2
	}

	s2 := s[2]
	if s2 < locb || hicb < s2 {
		return utf8RuneError, 1
	}
	if sz <= 3 {
		return rune(s0&mask3)<<12 | rune(s1&maskx)<<6 | rune(s2&maskx), 3
	}
	s3 := s[3]
	if s3 < locb || hicb < s3 {
		return utf8RuneError, 1
	}
	return rune(s0&mask4)<<18 | rune(s1&maskx)<<12 | rune(s2&maskx)<<6 | rune(s3&maskx), 4
}

const (
	rune1Max = 1<<7 - 1
	rune2Max = 1<<11 - 1
	rune3Max = 1<<16 - 1
)

// Code points in the surrogate range are not valid for UTF-8.
const (
	surrogateMin = 0xD800
	surrogateMax = 0xDFFF
)

const (
	tx = 0b10000000
	t2 = 0b11000000
	t3 = 0b11100000
	t4 = 0b11110000
)

const (
	unicodeMaxLatin1 = '\u00FF' // maximum Latin-1 value.

	// linearMax is the maximum size table for linear search for non-Latin1 rune.
	// Derived by running 'go test -calibrate'.
	unicodeLinearMax = 18
)

var (
	// unicodeWhiteSpace is the set of Unicode characters with property WhiteSpace.
	unicodeWhiteSpace = &unicodeRangeTable{
		R16: []unicodeRange16{
			{0x0009, 0x000d, 1},
			{0x0020, 0x0085, 101},
			{0x00a0, 0x1680, 5600},
			{0x2000, 0x200a, 1},
			{0x2028, 0x2029, 1},
			{0x202f, 0x205f, 48},
			{0x3000, 0x3000, 1},
		},
		LatinOffset: 2,
	}
)

// unicodeRangeTable defines a set of Unicode code points by listing the ranges of
// code points within the set. The ranges are listed in two slices
// to save space: a slice of 16-bit ranges and a slice of 32-bit ranges.
// The two slices must be in sorted order and non-overlapping.
// Also, R32 should contain only values >= 0x10000 (1<<16).
type unicodeRangeTable struct {
	R16         []unicodeRange16
	R32         []unicodeRange32
	LatinOffset int // number of entries in R16 with Hi <= MaxLatin1
}

// Range16 represents of a range of 16-bit Unicode code points. The range runs from Lo to Hi
// inclusive and has the specified stride.
type unicodeRange16 struct {
	Lo     uint16
	Hi     uint16
	Stride uint16
}

// Range32 represents of a range of Unicode code points and is used when one or
// more of the values will not fit in 16 bits. The range runs from Lo to Hi
// inclusive and has the specified stride. Lo and Hi must always be >= 1<<16.
type unicodeRange32 struct {
	Lo     uint32
	Hi     uint32
	Stride uint32
}

func unicodeIsExcludingLatin(rangeTab *unicodeRangeTable, r rune) bool {
	r16 := rangeTab.R16
	// Compare as uint32 to correctly handle negative runes.
	if off := rangeTab.LatinOffset; len(r16) > off && uint32(r) <= uint32(r16[len(r16)-1].Hi) {
		return unicodeIs16(r16[off:], uint16(r))
	}
	r32 := rangeTab.R32
	if len(r32) > 0 && r >= rune(r32[0].Lo) {
		bug("unicodeIsExcludingLatin")
		return unicodeIs32(r32, uint32(r))
	}
	return false
}

// is16 reports whether r is in the sorted slice of 16-bit ranges.
func unicodeIs16(ranges []unicodeRange16, r uint16) bool {
	if len(ranges) <= unicodeLinearMax || r <= unicodeMaxLatin1 {
		for i := range ranges {
			range_ := &ranges[i]
			if r < range_.Lo {
				return false
			}
			if r <= range_.Hi {
				return range_.Stride == 1 || (r-range_.Lo)%range_.Stride == 0
			}
		}
		bug("unicodeIs16")
		return false
	}

	// binary search over ranges
	lo := 0
	hi := len(ranges)
	for lo < hi {
		m := lo + (hi-lo)/2
		range_ := &ranges[m]
		if range_.Lo <= r && r <= range_.Hi {
			return range_.Stride == 1 || (r-range_.Lo)%range_.Stride == 0
		}
		if r < range_.Lo {
			hi = m
		} else {
			lo = m + 1
		}
	}
	return false
}

// is32 reports whether r is in the sorted slice of 32-bit ranges.
func unicodeIs32(ranges []unicodeRange32, r uint32) bool {
	if len(ranges) <= unicodeLinearMax {
		bug("unicodeIs32")
		for i := range ranges {
			range_ := &ranges[i]
			if r < range_.Lo {
				return false
			}
			if r <= range_.Hi {
				return range_.Stride == 1 || (r-range_.Lo)%range_.Stride == 0
			}
		}
		return false
	}

	// binary search over ranges
	lo := 0
	hi := len(ranges)
	for lo < hi {
		m := lo + (hi-lo)/2
		range_ := ranges[m]
		if range_.Lo <= r && r <= range_.Hi {
			return range_.Stride == 1 || (r-range_.Lo)%range_.Stride == 0
		}
		if r < range_.Lo {
			hi = m
		} else {
			lo = m + 1
		}
	}
	return false
}

// PrintRanges defines the set of printable characters according to Go.
// ASCII space, U+0020, is handled separately.
var unicodePrintRanges = []*unicodeRangeTable{
	L,
	M,
	N,
	P,
	S,
}

// Bit masks for each code point under U+0100, for fast lookup.
const (
	pC     = 1 << iota // a control character.
	pP                 // a punctuation character.
	pN                 // a numeral.
	pS                 // a symbolic character.
	pZ                 // a spacing character.
	pLu                // an upper-case letter.
	pLl                // a lower-case letter.
	pp                 // a printable character according to Go's definition.
	pg     = pp | pZ   // a graphical character according to the Unicode definition.
	pLo    = pLl | pLu // a letter that is neither upper nor lower case.
	pLmask = pLo
)

var (
	Letter = _L // Letter/L is the set of Unicode letters, category L.
	L      = _L
	Mark   = _M // Mark/M is the set of Unicode mark characters, category M.
	M      = _M
	Number = _N // Number/N is the set of Unicode number characters, category N.
	N      = _N
	Punct  = _P // Punct/P is the set of Unicode punctuation characters, category P.
	P      = _P
	Symbol = _S // Symbol/S is the set of Unicode symbol characters, category S.
	S      = _S
)

type stringer interface {
	String() string
}

// strings

///////////////////

const (
	nfc Form = iota
)

// Bytes returns f(b). May return b if f(b) = b.
func (f Form) Bytes(b []byte) []byte {
	bug("Bytes")
	src := inputBytes(b)
	ft := formTable[f]
	n, ok := ft.quickSpan(src, 0, len(b), true)
	if ok {
		return b
	}
	out := make([]byte, n, len(b))
	copy(out, b[0:n])
	rb := reorderBuffer{f: *ft, src: src, nsrc: len(b), out: out, flushF: appendFlush}
	return doAppendInner(&rb, n)
}

// IsNormal returns true if b == f(b).
func (f Form) IsNormal(b []byte) bool {
	bug("IsNormal")
	src := inputBytes(b)
	ft := formTable[f]
	bp, ok := ft.quickSpan(src, 0, len(b), true)
	if ok {
		return true
	}
	rb := reorderBuffer{f: *ft, src: src, nsrc: len(b)}
	rb.setFlusher(nil, cmpNormalBytes)
	for bp < len(b) {
		rb.out = b[bp:]
		if bp = decomposeSegment(&rb, bp, true); bp < 0 {
			return false
		}
		bp, _ = rb.f.quickSpan(rb.src, bp, len(b), true)
	}
	return true
}

func cmpNormalBytes(rb *reorderBuffer) bool {
	bug("cmpNormalBytes")
	b := rb.out
	for i := 0; i < rb.nrune; i++ {
		info := rb.rune[i]
		if int(info.size) > len(b) {
			return false
		}
		p := info.pos
		pe := p + info.size
		for ; p < pe; p++ {
			if b[0] != rb.byte[p] {
				return false
			}
			b = b[1:]
		}
	}
	return true
}

type input struct {
	str   string
	bytes []byte
}

func inputBytes(str []byte) input {
	bug("inputBytes")
	return input{bytes: str}
}

func (in *input) setString(str string) {
	bug("setString")
	in.str = str
	in.bytes = nil
}

func (in *input) _byte(p int) byte {
	bug("_byte")
	if in.bytes == nil {
		return in.str[p]
	}
	return in.bytes[p]
}

func (in *input) skipASCII(p, max int) int {
	if in.bytes == nil {
		for ; p < max && in.str[p] < utf8RuneSelf; p++ {
		}
	} else {
		bug("skipASCII")
		for ; p < max && in.bytes[p] < utf8RuneSelf; p++ {
		}
	}
	return p
}

// func (in *input) skipContinuationBytes(p int) int {
// 	bug("skipContinuationBytes")
// 	if in.bytes == nil {
// 		for ; p < len(in.str) && !utf8.RuneStart(in.str[p]); p++ {
// 		}
// 	} else {
// 		for ; p < len(in.bytes) && !utf8.RuneStart(in.bytes[p]); p++ {
// 		}
// 	}
// 	return p
// }

func (in *input) appendSlice(buf []byte, b, e int) []byte {
	if in.bytes != nil {
		bug("appendSlice")
		return append(buf, in.bytes[b:e]...)
	}
	for i := b; i < e; i++ {
		buf = append(buf, in.str[i])
	}
	return buf
}

func (in *input) copySlice(buf []byte, b, e int) int {
	if in.bytes == nil {
		return copy(buf, in.str[b:e])
	}
	bug("copySlice")
	return copy(buf, in.bytes[b:e])
}

func (in *input) charinfoNFC(p int) (uint16, int) {
	if in.bytes == nil {
		return nfcData.lookupString(in.str[p:])
	}
	return nfcData.lookup(in.bytes[p:])
}

func (in *input) charinfoNFKC(p int) (uint16, int) {
	bug("charinfoNFKC")
	if in.bytes == nil {
		return nfkcData.lookupString(in.str[p:])
	}
	return nfkcData.lookup(in.bytes[p:])
}

func (in *input) hangul(p int) (r rune) {
	var size int
	if in.bytes == nil {
		if !isHangulString(in.str[p:]) {
			return 0
		}
		r, size = utf8DecodeRuneInString(in.str[p:])
	} else {
		bug("hangul")
		if !isHangul(in.bytes[p:]) {
			return 0
		}
		r, size = utf8DecodeRune(in.bytes[p:])
	}
	if size != hangulUTF8Size {
		return 0
	}
	return r
}

// quickSpan returns a boundary n such that src[0:n] == f(src[0:n]) and
// whether any non-normalized parts were found. If atEOF is false, n will
// not point past the last segment if this segment might be become
// non-normalized by appending other runes.
func (f *formInfo) quickSpan(src input, i, end int, atEOF bool) (n int, ok bool) {
	var lastCC uint8
	ss := streamSafe(0)
	lastSegStart := i
	for n = end; i < n; {
		if j := src.skipASCII(i, n); i != j {
			i = j
			lastSegStart = i - 1
			lastCC = 0
			ss = 0
			continue
		}
		info := f.info(src, i)
		if info.size == 0 {
			if atEOF {
				// include incomplete runes
				return n, true
			}
			bug("bug")
			return lastSegStart, true
		}
		// This block needs to be before the next, because it is possible to
		// have an overflow for runes that are starters (e.g. with U+FF9E).
		switch ss.next(info) {
		case ssStarter:
			lastSegStart = i
		case ssOverflow:
			bug("quickSpan")
			return lastSegStart, false
		case ssSuccess:
			if lastCC > info.ccc {
				return lastSegStart, false
			}
		}
		if f.composing {
			if !info.isYesC() {
				break
			}
		} else {
			bug("quickSpan")
			if !info.isYesD() {
				bug("quickSpan")
				break
			}
		}
		lastCC = info.ccc
		i += int(info.size)
	}
	if i == n {
		if !atEOF {
			bug("quickSpan")
			n = lastSegStart
		}
		return n, true
	}
	return lastSegStart, false
}

// QuickSpanString returns a boundary n such that s[0:n] == f(s[0:n]).
// It is not guaranteed to return the largest such n.
func (f Form) QuickSpanString(s string) int {
	bug("QuickSpanString")
	n, _ := formTable[f].quickSpan(inputString(s), 0, len(s), true)
	return n
}

// decomposeSegment scans the first segment in src into rb. It inserts 0x034f
// (Grapheme Joiner) when it encounters a sequence of more than 30 non-starters
// and returns the number of bytes consumed from src or iShortDst or iShortSrc.
func decomposeSegment(rb *reorderBuffer, sp int, atEOF bool) int {
	// Force one character to be consumed.
	info := rb.f.info(rb.src, sp)
	if info.size == 0 {
		bug("bug")
		return 0
	}
	if s := rb.ss.next(info); s == ssStarter {
		// TODO: this could be removed if we don't support merging.
		if rb.nrune > 0 {
			bug("decomposeSegment")
			goto end
		}
	} else if s == ssOverflow {
		bug("decomposeSegment")
		rb.insertCGJ()
		goto end
	}
	if err := rb.insertFlush(rb.src, sp, info); err != iSuccess {
		bug("decomposeSegment")
		return int(err)
	}
	for {
		sp += int(info.size)
		if sp >= rb.nsrc {
			if !atEOF && !info.BoundaryAfter() {
				bug("decomposeSegment")
				return int(iShortSrc)
			}
			break
		}
		info = rb.f.info(rb.src, sp)
		if info.size == 0 {
			if !atEOF {
				bug("decomposeSegment")
				return int(iShortSrc)
			}
			break
		}
		if s := rb.ss.next(info); s == ssStarter {
			break
		} else if s == ssOverflow {
			rb.insertCGJ()
			break
		}

		if err := rb.insertFlush(rb.src, sp, info); err != iSuccess {
			bug("decomposeSegment")
			return int(err)
		}
	}
end:
	if !rb.doFlush() {
		bug("decomposeSegment")
		return int(iShortDst)
	}
	return sp
}

// Rune info is stored in a separate trie per composing form. A composing form
// and its corresponding decomposing form share the same trie.  Each trie maps
// a rune to a uint16. The values take two forms.  For v >= 0x8000:
//   bits
//   15:    1 (inverse of NFD_QC bit of qcInfo)
//   13..7: qcInfo (see below). isYesD is always true (no decompostion).
//    6..0: ccc (compressed CCC value).
// For v < 0x8000, the respective rune has a decomposition and v is an index
// into a byte array of UTF-8 decomposition sequences and additional info and
// has the form:
//    <header> <decomp_byte>* [<tccc> [<lccc>]]
// The header contains the number of bytes in the decomposition (excluding this
// length byte). The two most significant bits of this length byte correspond
// to bit 5 and 4 of qcInfo (see below).  The byte sequence itself starts at v+1.
// The byte sequence is followed by a trailing and leading CCC if the values
// for these are not zero.  The value of v determines which ccc are appended
// to the sequences.  For v < firstCCC, there are none, for v >= firstCCC,
// the sequence is followed by a trailing ccc, and for v >= firstLeadingCC
// there is an additional leading ccc. The value of tccc itself is the
// trailing CCC shifted left 2 bits. The two least-significant bits of tccc
// are the number of trailing non-starters.

const (
	qcInfoMask      = 0x3F // to clear all but the relevant bits in a qcInfo
	headerLenMask   = 0x3F // extract the length value from the header byte
	headerFlagsMask = 0xC0 // extract the qcInfo bits from the header byte
)

// functions dispatchable per form
type lookupFunc func(b input, i int) Properties

// formInfo holds Form-specific functions and tables.
type formInfo struct {
	form                     Form
	composing, compatibility bool // form type
	info                     lookupFunc
	nextMain                 iterFunc
}

var formTable = []*formInfo{{
	form:          nfc,
	composing:     true,
	compatibility: false,
	info:          lookupInfoNFC,
	nextMain:      nextComposed,
}}

// BoundaryAfter returns true if runes cannot combine with or otherwise
// interact with this or previous runes.
func (p Properties) BoundaryAfter() bool {
	bug("BoundaryAfter")
	// TODO: loosen these conditions.
	return p.isInert()
}

const (
	maxNonStarters = 30
	// The maximum number of characters needed for a buffer is
	// maxNonStarters + 1 for the starter + 1 for the GCJ
	maxBufferSize    = maxNonStarters + 2
	maxNFCExpansion  = 3  // NFC(0x1D160)
	maxNFKCExpansion = 18 // NFKC(0xFDFA)

	maxByteBufferSize = utf8UTFMax * maxBufferSize // 128
)

// ssState is used for reporting the segment state after inserting a rune.
// It is returned by streamSafe.next.
type ssState int

const (
	// Indicates a rune was successfully added to the segment.
	ssSuccess ssState = iota
	// Indicates a rune starts a new segment and should not be added.
	ssStarter
	// Indicates a rune caused a segment overflow and a CGJ should be inserted.
	ssOverflow
)

// streamSafe implements the policy of when a CGJ should be inserted.
type streamSafe uint8

// first inserts the first rune of a segment. It is a faster version of next if
// it is known p represents the first rune in a segment.
func (ss *streamSafe) first(p Properties) {
	bug("first")
	*ss = streamSafe(p.nTrailingNonStarters())
}

// insert returns a ssState value to indicate whether a rune represented by p
// can be inserted.
func (ss *streamSafe) next(p Properties) ssState {
	if *ss > maxNonStarters {
		panic("streamSafe was not reset")
	}
	n := p.nLeadingNonStarters()
	if *ss += streamSafe(n); *ss > maxNonStarters {
		*ss = 0
		return ssOverflow
	}
	// The Stream-Safe Text Processing prescribes that the counting can stop
	// as soon as a starter is encountered. However, there are some starters,
	// like Jamo V and T, that can combine with other runes, leaving their
	// successive non-starters appended to the previous, possibly causing an
	// overflow. We will therefore consider any rune with a non-zero nLead to
	// be a non-starter. Note that it always hold that if nLead > 0 then
	// nLead == nTrail.
	if n == 0 {
		*ss = streamSafe(p.nTrailingNonStarters())
		return ssStarter
	}
	return ssSuccess
}

// backwards is used for checking for overflow and segment starts
// when traversing a string backwards. Users do not need to call first
// for the first rune. The state of the streamSafe retains the count of
// the non-starters loaded.
func (ss *streamSafe) backwards(p Properties) ssState {
	bug("backwards")
	if *ss > maxNonStarters {
		panic("streamSafe was not reset")
	}
	c := *ss + streamSafe(p.nTrailingNonStarters())
	if c > maxNonStarters {
		return ssOverflow
	}
	*ss = c
	if p.nLeadingNonStarters() == 0 {
		return ssStarter
	}
	return ssSuccess
}

func (ss streamSafe) isMax() bool {
	bug("isMax")
	return ss == maxNonStarters
}

// GraphemeJoiner is inserted after maxNonStarters non-starter runes.
const GraphemeJoiner = "\u034F"

// reorderBuffer is used to normalize a single segment.  Characters inserted with
// insert are decomposed and reordered based on CCC. The compose method can
// be used to recombine characters.  Note that the byte buffer does not hold
// the UTF-8 characters in order.  Only the rune array is maintained in sorted
// order. flush writes the resulting segment to a byte array.
type reorderBuffer struct {
	rune  [maxBufferSize]Properties // Per character info.
	byte  [maxByteBufferSize]byte   // UTF-8 buffer. Referenced by runeInfo.pos.
	nbyte uint8                     // Number or bytes.
	ss    streamSafe                // For limiting length of non-starter sequence.
	nrune int                       // Number of runeInfos.
	f     formInfo

	src      input
	nsrc     int
	tmpBytes input

	out    []byte
	flushF func(*reorderBuffer) bool
}

func (rb *reorderBuffer) init(f Form, src []byte) {
	bug("init")
	rb.f = *formTable[f]
	rb.src.setBytes(src)
	rb.nsrc = len(src)
	rb.ss = 0
}

func (rb *reorderBuffer) initString(f Form, src string) {
	bug("initString")
	rb.f = *formTable[f]
	rb.src.setString(src)
	rb.nsrc = len(src)
	rb.ss = 0
}

func (rb *reorderBuffer) setFlusher(out []byte, f func(*reorderBuffer) bool) {
	bug("setFlusher")
	rb.out = out
	rb.flushF = f
}

// MaxSegmentSize is the maximum size of a byte buffer needed to consider any
// sequence of starter and non-starter runes for the purpose of normalization.
const MaxSegmentSize = maxByteBufferSize

// An Iter iterates over a string or byte slice, while normalizing it
// to a given Form.
type Iter struct {
	rb     reorderBuffer
	buf    [maxByteBufferSize]byte
	info   Properties // first character saved from previous iteration
	next   iterFunc   // implementation of next depends on form
	asciiF iterFunc

	p        int    // current position in input source
	multiSeg []byte // remainder of multi-segment decomposition
}

type iterFunc func(*Iter) []byte

// Init initializes i to iterate over src after normalizing it to Form f.
func (i *Iter) Init(f Form, src []byte) {
	bug("Init")
	i.p = 0
	if len(src) == 0 {
		i.setDone()
		i.rb.nsrc = 0
		return
	}
	i.multiSeg = nil
	i.rb.init(f, src)
	i.next = i.rb.f.nextMain
	i.asciiF = nextASCIIBytes
	i.info = i.rb.f.info(i.rb.src, i.p)
	i.rb.ss.first(i.info)
}

// InitString initializes i to iterate over src after normalizing it to Form f.
func (i *Iter) InitString(f Form, src string) {
	bug("InitString")
	i.p = 0
	if len(src) == 0 {
		i.setDone()
		i.rb.nsrc = 0
		return
	}
	i.multiSeg = nil
	i.rb.initString(f, src)
	i.next = i.rb.f.nextMain
	i.asciiF = nextASCIIString
	i.info = i.rb.f.info(i.rb.src, i.p)
	i.rb.ss.first(i.info)
}

// Seek sets the segment to be returned by the next call to Next to start
// at position p.  It is the responsibility of the caller to set p to the
// start of a segment.
func (i *Iter) Seek(offset int64, whence int) (int64, error) {
	bug("Seek")
	var abs int64
	switch whence {
	case 0:
		abs = offset
	case 1:
		abs = int64(i.p) + offset
	case 2:
		abs = int64(i.rb.nsrc) + offset
	default:
		return 0, fmt.Errorf("norm: invalid whence")
	}
	if abs < 0 {
		return 0, fmt.Errorf("norm: negative position")
	}
	if int(abs) >= i.rb.nsrc {
		i.setDone()
		return int64(i.p), nil
	}
	i.p = int(abs)
	i.multiSeg = nil
	i.next = i.rb.f.nextMain
	i.info = i.rb.f.info(i.rb.src, i.p)
	i.rb.ss.first(i.info)
	return abs, nil
}

// returnSlice returns a slice of the underlying input type as a byte slice.
// If the underlying is of type []byte, it will simply return a slice.
// If the underlying is of type string, it will copy the slice to the buffer
// and return that.
func (i *Iter) returnSlice(a, b int) []byte {
	bug("returnSlice")
	if i.rb.src.bytes == nil {
		return i.buf[:copy(i.buf[:], i.rb.src.str[a:b])]
	}
	return i.rb.src.bytes[a:b]
}

// Pos returns the byte position at which the next call to Next will commence processing.
func (i *Iter) Pos() int {
	bug("Pos")
	return i.p
}

func (i *Iter) setDone() {
	bug("setDone")
	i.next = nextDone
	i.p = i.rb.nsrc
}

// Done returns true if there is no more input to process.
func (i *Iter) Done() bool {
	bug("Done")
	return i.p >= i.rb.nsrc
}

// Next returns f(i.input[i.Pos():n]), where n is a boundary of i.input.
// For any input a and b for which f(a) == f(b), subsequent calls
// to Next will return the same segments.
// Modifying runes are grouped together with the preceding starter, if such a starter exists.
// Although not guaranteed, n will typically be the smallest possible n.
func (i *Iter) Next() []byte {
	bug("Next")
	return i.next(i)
}

func nextASCIIBytes(i *Iter) []byte {
	bug("nextASCIIBytes")
	p := i.p + 1
	if p >= i.rb.nsrc {
		p0 := i.p
		i.setDone()
		return i.rb.src.bytes[p0:p]
	}
	if i.rb.src.bytes[p] < utf8RuneSelf {
		p0 := i.p
		i.p = p
		return i.rb.src.bytes[p0:p]
	}
	i.info = i.rb.f.info(i.rb.src, i.p)
	i.next = i.rb.f.nextMain
	return i.next(i)
}

func nextASCIIString(i *Iter) []byte {
	bug("nextASCIIString")
	p := i.p + 1
	if p >= i.rb.nsrc {
		i.buf[0] = i.rb.src.str[i.p]
		i.setDone()
		return i.buf[:1]
	}
	if i.rb.src.str[p] < utf8RuneSelf {
		i.buf[0] = i.rb.src.str[i.p]
		i.p = p
		return i.buf[:1]
	}
	i.info = i.rb.f.info(i.rb.src, i.p)
	i.next = i.rb.f.nextMain
	return i.next(i)
}

func nextHangul(i *Iter) []byte {
	bug("nextHangul")
	p := i.p
	next := p + hangulUTF8Size
	if next >= i.rb.nsrc {
		i.setDone()
	} else if i.rb.src.hangul(next) == 0 {
		i.rb.ss.next(i.info)
		i.info = i.rb.f.info(i.rb.src, i.p)
		i.next = i.rb.f.nextMain
		return i.next(i)
	}
	i.p = next
	return i.buf[:decomposeHangul(i.buf[:], i.rb.src.hangul(p))]
}

func nextDone(i *Iter) []byte {
	bug("nextDone")
	return nil
}

// nextMulti is used for iterating over multi-segment decompositions
// for decomposing normal forms.
func nextMulti(i *Iter) []byte {
	bug("nextMulti")
	j := 0
	d := i.multiSeg
	// skip first rune
	for j = 1; j < len(d) && !utf8RuneStart(d[j]); j++ {
	}
	for j < len(d) {
		info := i.rb.f.info(input{bytes: d}, j)
		if info.BoundaryBefore() {
			i.multiSeg = d[j:]
			return d[:j]
		}
		j += int(info.size)
	}
	// treat last segment as normal decomposition
	i.next = i.rb.f.nextMain
	return i.next(i)
}

// nextMultiNorm is used for iterating over multi-segment decompositions
// for composing normal forms.
func nextMultiNorm(i *Iter) []byte {
	bug("nextMultiNorm")
	j := 0
	d := i.multiSeg
	for j < len(d) {
		info := i.rb.f.info(input{bytes: d}, j)
		if info.BoundaryBefore() {
			i.rb.compose()
			seg := i.buf[:i.rb.flushCopy(i.buf[:])]
			i.rb.insertUnsafe(input{bytes: d}, j, info)
			i.multiSeg = d[j+int(info.size):]
			return seg
		}
		i.rb.insertUnsafe(input{bytes: d}, j, info)
		j += int(info.size)
	}
	i.multiSeg = nil
	i.next = nextComposed
	return doNormComposed(i)
}

// nextDecomposed is the implementation of Next for forms NFD and NFKD.
func nextDecomposed(i *Iter) (next []byte) {
	bug("nextDecomposed")
	outp := 0
	inCopyStart, outCopyStart := i.p, 0
	for {
		if sz := int(i.info.size); sz <= 1 {
			i.rb.ss = 0
			p := i.p
			i.p++ // ASCII or illegal byte.  Either way, advance by 1.
			if i.p >= i.rb.nsrc {
				i.setDone()
				return i.returnSlice(p, i.p)
			} else if i.rb.src._byte(i.p) < utf8RuneSelf {
				i.next = i.asciiF
				return i.returnSlice(p, i.p)
			}
			outp++
		} else if d := i.info.Decomposition(); d != nil {
			// Note: If leading CCC != 0, then len(d) == 2 and last is also non-zero.
			// Case 1: there is a leftover to copy.  In this case the decomposition
			// must begin with a modifier and should always be appended.
			// Case 2: no leftover. Simply return d if followed by a ccc == 0 value.
			p := outp + len(d)
			if outp > 0 {
				i.rb.src.copySlice(i.buf[outCopyStart:], inCopyStart, i.p)
				// TODO: this condition should not be possible, but we leave it
				// in for defensive purposes.
				if p > len(i.buf) {
					return i.buf[:outp]
				}
			} else if i.info.multiSegment() {
				// outp must be 0 as multi-segment decompositions always
				// start a new segment.
				if i.multiSeg == nil {
					i.multiSeg = d
					i.next = nextMulti
					return nextMulti(i)
				}
				// We are in the last segment.  Treat as normal decomposition.
				d = i.multiSeg
				i.multiSeg = nil
				p = len(d)
			}
			prevCC := i.info.tccc
			if i.p += sz; i.p >= i.rb.nsrc {
				i.setDone()
				i.info = Properties{} // Force BoundaryBefore to succeed.
			} else {
				i.info = i.rb.f.info(i.rb.src, i.p)
			}
			switch i.rb.ss.next(i.info) {
			case ssOverflow:
				i.next = nextCGJDecompose
				fallthrough
			case ssStarter:
				if outp > 0 {
					copy(i.buf[outp:], d)
					return i.buf[:p]
				}
				return d
			}
			copy(i.buf[outp:], d)
			outp = p
			inCopyStart, outCopyStart = i.p, outp
			if i.info.ccc < prevCC {
				goto doNorm
			}
			continue
		} else if r := i.rb.src.hangul(i.p); r != 0 {
			outp = decomposeHangul(i.buf[:], r)
			i.p += hangulUTF8Size
			inCopyStart, outCopyStart = i.p, outp
			if i.p >= i.rb.nsrc {
				i.setDone()
				break
			} else if i.rb.src.hangul(i.p) != 0 {
				i.next = nextHangul
				return i.buf[:outp]
			}
		} else {
			p := outp + sz
			if p > len(i.buf) {
				break
			}
			outp = p
			i.p += sz
		}
		if i.p >= i.rb.nsrc {
			i.setDone()
			break
		}
		prevCC := i.info.tccc
		i.info = i.rb.f.info(i.rb.src, i.p)
		if v := i.rb.ss.next(i.info); v == ssStarter {
			break
		} else if v == ssOverflow {
			i.next = nextCGJDecompose
			break
		}
		if i.info.ccc < prevCC {
			goto doNorm
		}
	}
	if outCopyStart == 0 {
		return i.returnSlice(inCopyStart, i.p)
	} else if inCopyStart < i.p {
		i.rb.src.copySlice(i.buf[outCopyStart:], inCopyStart, i.p)
	}
	return i.buf[:outp]
doNorm:
	// Insert what we have decomposed so far in the reorderBuffer.
	// As we will only reorder, there will always be enough room.
	i.rb.src.copySlice(i.buf[outCopyStart:], inCopyStart, i.p)
	i.rb.insertDecomposed(i.buf[0:outp])
	return doNormDecomposed(i)
}

func doNormDecomposed(i *Iter) []byte {
	bug("doNormDecomposed")
	for {
		i.rb.insertUnsafe(i.rb.src, i.p, i.info)
		if i.p += int(i.info.size); i.p >= i.rb.nsrc {
			i.setDone()
			break
		}
		i.info = i.rb.f.info(i.rb.src, i.p)
		if i.info.ccc == 0 {
			break
		}
		if s := i.rb.ss.next(i.info); s == ssOverflow {
			i.next = nextCGJDecompose
			break
		}
	}
	// new segment or too many combining characters: exit normalization
	return i.buf[:i.rb.flushCopy(i.buf[:])]
}

func nextCGJDecompose(i *Iter) []byte {
	bug("nextCGJDecompose")
	i.rb.ss = 0
	i.rb.insertCGJ()
	i.next = nextDecomposed
	i.rb.ss.first(i.info)
	buf := doNormDecomposed(i)
	return buf
}

// nextComposed is the implementation of Next for forms NFC and NFKC.
func nextComposed(i *Iter) []byte {
	bug("nextComposed")
	outp, startp := 0, i.p
	var prevCC uint8
	for {
		if !i.info.isYesC() {
			goto doNorm
		}
		prevCC = i.info.tccc
		sz := int(i.info.size)
		if sz == 0 {
			sz = 1 // illegal rune: copy byte-by-byte
		}
		p := outp + sz
		if p > len(i.buf) {
			break
		}
		outp = p
		i.p += sz
		if i.p >= i.rb.nsrc {
			i.setDone()
			break
		} else if i.rb.src._byte(i.p) < utf8RuneSelf {
			i.rb.ss = 0
			i.next = i.asciiF
			break
		}
		i.info = i.rb.f.info(i.rb.src, i.p)
		if v := i.rb.ss.next(i.info); v == ssStarter {
			break
		} else if v == ssOverflow {
			i.next = nextCGJCompose
			break
		}
		if i.info.ccc < prevCC {
			goto doNorm
		}
	}
	return i.returnSlice(startp, i.p)
doNorm:
	// reset to start position
	i.p = startp
	i.info = i.rb.f.info(i.rb.src, i.p)
	i.rb.ss.first(i.info)
	if i.info.multiSegment() {
		d := i.info.Decomposition()
		info := i.rb.f.info(input{bytes: d}, 0)
		i.rb.insertUnsafe(input{bytes: d}, 0, info)
		i.multiSeg = d[int(info.size):]
		i.next = nextMultiNorm
		return nextMultiNorm(i)
	}
	i.rb.ss.first(i.info)
	i.rb.insertUnsafe(i.rb.src, i.p, i.info)
	return doNormComposed(i)
}

func doNormComposed(i *Iter) []byte {
	bug("doNormComposed")
	// First rune should already be inserted.
	for {
		if i.p += int(i.info.size); i.p >= i.rb.nsrc {
			i.setDone()
			break
		}
		i.info = i.rb.f.info(i.rb.src, i.p)
		if s := i.rb.ss.next(i.info); s == ssStarter {
			break
		} else if s == ssOverflow {
			i.next = nextCGJCompose
			break
		}
		i.rb.insertUnsafe(i.rb.src, i.p, i.info)
	}
	i.rb.compose()
	seg := i.buf[:i.rb.flushCopy(i.buf[:])]
	return seg
}

func nextCGJCompose(i *Iter) []byte {
	bug("nextCGJCompose")
	i.rb.ss = 0 // instead of first
	i.rb.insertCGJ()
	i.next = nextComposed
	// Note that we treat any rune with nLeadingNonStarters > 0 as a non-starter,
	// even if they are not. This is particularly dubious for U+FF9E and UFF9A.
	// If we ever change that, insert a check here.
	i.rb.ss.first(i.info)
	i.rb.insertUnsafe(i.rb.src, i.p, i.info)
	return doNormComposed(i)
}

// flush appends the normalized segment to out and resets rb.
func (rb *reorderBuffer) flush(out []byte) []byte {
	bug("flush")
	for i := 0; i < rb.nrune; i++ {
		start := rb.rune[i].pos
		end := start + rb.rune[i].size
		out = append(out, rb.byte[start:end]...)
	}
	rb.reset()
	return out
}

// flushCopy copies the normalized segment to buf and resets rb.
// It returns the number of bytes written to buf.
func (rb *reorderBuffer) flushCopy(buf []byte) int {
	bug("flushCopy")
	p := 0
	for i := 0; i < rb.nrune; i++ {
		runep := rb.rune[i]
		p += copy(buf[p:], rb.byte[runep.pos:runep.pos+runep.size])
	}
	rb.reset()
	return p
}

// insertErr is an error code returned by insert. Using this type instead
// of error improves performance up to 20% for many of the benchmarks.
type insertErr int

const (
	iSuccess insertErr = -iota
	iShortDst
	iShortSrc
)

// insertUnsafe inserts the given rune in the buffer ordered by CCC.
// It is assumed there is sufficient space to hold the runes. It is the
// responsibility of the caller to ensure this. This can be done by checking
// the state returned by the streamSafe type.
func (rb *reorderBuffer) insertUnsafe(src input, i int, info Properties) {
	bug("insertUnsafe")
	if rune := src.hangul(i); rune != 0 {
		rb.decomposeHangul(rune)
	}
	if info.hasDecomposition() {
		// TODO: inline.
		rb.insertDecomposed(info.Decomposition())
	} else {
		rb.insertSingle(src, i, info)
	}
}

// insertDecomposed inserts an entry in to the reorderBuffer for each rune
// in dcomp. dcomp must be a sequence of decomposed UTF-8-encoded runes.
// It flushes the buffer on each new segment start.
func (rb *reorderBuffer) insertDecomposed(dcomp []byte) insertErr {
	rb.tmpBytes.setBytes(dcomp)
	// As the streamSafe accounting already handles the counting for modifiers,
	// we don't have to call next. However, we do need to keep the accounting
	// intact when flushing the buffer.
	for i := 0; i < len(dcomp); {
		info := rb.f.info(rb.tmpBytes, i)
		if info.BoundaryBefore() && rb.nrune > 0 && !rb.doFlush() {
			bug("insertDecomposed")
			return iShortDst
		}
		i += copy(rb.byte[rb.nbyte:], dcomp[i:i+int(info.size)])
		rb.insertOrdered(info)
	}
	return iSuccess
}

// EncodeRune writes into p (which must be large enough) the UTF-8 encoding of the rune.
// If the rune is out of range, it writes the encoding of RuneError.
// It returns the number of bytes written.
func utf8EncodeRune(p []byte, r rune) int {
	// Negative values are erroneous. Making it unsigned addresses the problem.
	switch i := uint32(r); {
	case i <= rune1Max:
		bug("bug")
		p[0] = byte(r)
		return 1
	case i <= rune2Max:
		_ = p[1] // eliminate bounds checks
		p[0] = t2 | byte(r>>6)
		p[1] = tx | byte(r)&maskx
		return 2
	case i > utf8MaxRune, surrogateMin <= i && i <= surrogateMax:
		bug("bug")
		r = utf8RuneError
		fallthrough
	case i <= rune3Max:
		_ = p[2] // eliminate bounds checks
		p[0] = t3 | byte(r>>12)
		p[1] = tx | byte(r>>6)&maskx
		p[2] = tx | byte(r)&maskx
		return 3
	default:
		bug("bug")
		_ = p[3] // eliminate bounds checks
		p[0] = t4 | byte(r>>18)
		p[1] = tx | byte(r>>12)&maskx
		p[2] = tx | byte(r>>6)&maskx
		p[3] = tx | byte(r)&maskx
		return 4
	}
}

// DecodeRune unpacks the first UTF-8 encoding in p and returns the rune and
// its width in bytes. If p is empty it returns (RuneError, 0). Otherwise, if
// the encoding is invalid, it returns (RuneError, 1). Both are impossible
// results for correct, non-empty UTF-8.
//
// An encoding is invalid if it is incorrect UTF-8, encodes a rune that is
// out of range, or is not the shortest possible UTF-8 encoding for the
// value. No other validation is performed.
func utf8DecodeRune(p []byte) (r rune, size int) {
	n := len(p)
	if n < 1 {
		bug("bug")
		return utf8RuneError, 0
	}
	p0 := p[0]
	x := utf8first[p0]
	if x >= as {
		// The following code simulates an additional check for x == xx and
		// handling the ASCII and invalid cases accordingly. This mask-and-or
		// approach prevents an additional branch.
		mask := rune(x) << 31 >> 31 // Create 0x0000 or 0xFFFF.
		return rune(p[0])&^mask | utf8RuneError&mask, 1
	}
	sz := int(x & 7)
	accept := utf8AcceptRanges[x>>4]
	if n < sz {
		return utf8RuneError, 1
	}
	b1 := p[1]
	if b1 < accept.lo || accept.hi < b1 {
		return utf8RuneError, 1
	}
	if sz <= 2 { // <= instead of == to help the compiler eliminate some bounds checks
		return rune(p0&mask2)<<6 | rune(b1&maskx), 2
	}
	b2 := p[2]
	if b2 < locb || hicb < b2 {
		bug("bug")
		return utf8RuneError, 1
	}
	if sz <= 3 {
		return rune(p0&mask3)<<12 | rune(b1&maskx)<<6 | rune(b2&maskx), 3
	}

	b3 := p[3]
	if b3 < locb || hicb < b3 {
		bug("bug")
		return utf8RuneError, 1
	}

	return rune(p0&mask4)<<18 | rune(b1&maskx)<<12 | rune(b2&maskx)<<6 | rune(b3&maskx), 4
}

// For Hangul we combine algorithmically, instead of using tables.
const (
	hangulBase  = 0xAC00 // UTF-8(hangulBase) -> EA B0 80
	hangulBase0 = 0xEA
	hangulBase1 = 0xB0
	hangulBase2 = 0x80

	hangulEnd  = hangulBase + jamoLVTCount // UTF-8(0xD7A4) -> ED 9E A4
	hangulEnd0 = 0xED
	hangulEnd1 = 0x9E
	hangulEnd2 = 0xA4

	jamoLBase  = 0x1100 // UTF-8(jamoLBase) -> E1 84 00
	jamoLBase0 = 0xE1
	jamoLBase1 = 0x84
	jamoLEnd   = 0x1113
	jamoVBase  = 0x1161
	jamoVEnd   = 0x1176
	jamoTBase  = 0x11A7
	jamoTEnd   = 0x11C3

	jamoTCount   = 28
	jamoVCount   = 21
	jamoVTCount  = 21 * 28
	jamoLVTCount = 19 * 21 * 28
)

const hangulUTF8Size = 3

func isHangul(b []byte) bool {
	bug("isHangul")
	if len(b) < hangulUTF8Size {
		return false
	}
	b0 := b[0]
	if b0 < hangulBase0 {
		return false
	}
	b1 := b[1]
	switch {
	case b0 == hangulBase0:
		return b1 >= hangulBase1
	case b0 < hangulEnd0:
		return true
	case b0 > hangulEnd0:
		return false
	case b1 < hangulEnd1:
		return true
	}
	return b1 == hangulEnd1 && b[2] < hangulEnd2
}

func isHangulWithoutJamoT(b []byte) bool {
	bug("isHangulWithoutJamoT")
	c, _ := utf8DecodeRune(b)
	c -= hangulBase
	return c < jamoLVTCount && c%jamoTCount == 0
}

// decomposeHangul writes the decomposed Hangul to buf and returns the number
// of bytes written.  len(buf) should be at least 9.
func decomposeHangul(buf []byte, r rune) int {
	bug("decomposeHangul")
	const JamoUTF8Len = 3
	r -= hangulBase
	x := r % jamoTCount
	r /= jamoTCount
	utf8EncodeRune(buf, jamoLBase+r/jamoVCount)
	utf8EncodeRune(buf[JamoUTF8Len:], jamoVBase+r%jamoVCount)
	if x != 0 {
		utf8EncodeRune(buf[2*JamoUTF8Len:], jamoTBase+x)
		return 3 * JamoUTF8Len
	}
	return 2 * JamoUTF8Len
}

// compose recombines the runes in the buffer.
// It should only be used to recompose a single segment, as it will not
// handle alternations between Hangul and non-Hangul characters correctly.
func (rb *reorderBuffer) compose() {
	// Lazily load the map used by the combine func below, but do
	// it outside of the loop.
	recompMapOnce.Do(buildRecompMap)

	// UAX #15, section X5 , including Corrigendum #5
	// "In any character sequence beginning with starter S, a character C is
	//  blocked from S if and only if there is some character B between S
	//  and C, and either B is a starter or it has the same or higher
	//  combining class as C."
	bn := rb.nrune
	if bn == 0 {
		bug("compose")
		return
	}
	k := 1
	b := rb.rune[:]
	for s, i := 0, 1; i < bn; i++ {
		if isJamoVT(rb.bytesAt(i)) {
			// Redo from start in Hangul mode. Necessary to support
			// U+320E..U+321E in NFKC mode.
			rb.combineHangul(s, i, k)
			return
		}

		ii := b[i]
		// We can only use combineForward as a filter if we later
		// get the info for the combined character. This is more
		// expensive than using the filter. Using combinesBackward()
		// is safe.
		if ii.combinesBackward() {
			cccB := b[k-1].ccc
			cccC := ii.ccc
			blocked := false // b[i] blocked by starter or greater or equal CCC?
			if cccB == 0 {
				s = k - 1
			} else {
				blocked = s != k-1 && cccB >= cccC
			}
			if !blocked {
				combined := combine(rb.runeAt(s), rb.runeAt(i))
				if combined != 0 {
					rb.assignRune(s, combined)
					continue
				}
			}
		}
		b[k] = b[i]
		k++
	}
	rb.nrune = k
}

// We pack quick check data in 4 bits:
//
//	5:    Combines forward  (0 == false, 1 == true)
//	4..3: NFC_QC Yes(00), No (10), or Maybe (11)
//	2:    NFD_QC Yes (0) or No (1). No also means there is a decomposition.
//	1..0: Number of trailing non-starters.
//
// When all 4 bits are zero, the character is inert, meaning it is never
// influenced by normalization.
type qcInfo uint8

func (p Properties) isYesD() bool {
	bug("isYesD")
	return p.flags&0x4 == 0
}

func (p Properties) combinesForward() bool {
	bug("isYesD")
	return p.flags&0x20 != 0
}

func (p Properties) isInert() bool {
	bug("isInert")
	return p.flags&qcInfoMask == 0 && p.ccc == 0
}

func (p Properties) multiSegment() bool {
	bug("multiSegment")
	return p.index >= firstMulti && p.index < endMulti
}

// Decomposition returns the decomposition for the underlying rune
// or nil if there is none.
func (p Properties) Decomposition() []byte {
	// TODO: create the decomposition for Hangul?
	if p.index == 0 {
		bug("Decomposition")
		return nil
	}
	i := p.index
	n := decomps[i] & headerLenMask
	i++
	return decomps[i : i+uint16(n)]
}

// Size returns the length of UTF-8 encoding of the rune.
func (p Properties) Size() int {
	bug("Size")
	return int(p.size)
}

// CCC returns the canonical combining class of the underlying rune.
func (p Properties) CCC() uint8 {
	bug("CCC")
	if p.index >= firstCCCZeroExcept {
		return 0
	}
	return ccc[p.ccc]
}

// LeadCCC returns the CCC of the first rune in the decomposition.
// If there is no decomposition, LeadCCC equals CCC.
func (p Properties) LeadCCC() uint8 {
	bug("LeadCCC")
	return ccc[p.ccc]
}

// TrailCCC returns the CCC of the last rune in the decomposition.
// If there is no decomposition, TrailCCC equals CCC.
func (p Properties) TrailCCC() uint8 {
	bug("TrailCCC")
	return ccc[p.tccc]
}

// Recomposition
// We use 32-bit keys instead of 64-bit for the two codepoint keys.
// This clips off the bits of three entries, but we know this will not
// result in a collision. In the unlikely event that changes to
// UnicodeData.txt introduce collisions, the compiler will catch it.
// Note that the recomposition map for NFC and NFKC are identical.

// combine returns the combined rune or 0 if it doesn't exist.
//
// The caller is responsible for calling
// recompMapOnce.Do(buildRecompMap) sometime before this is called.
func combine(a, b rune) rune {
	key := uint32(uint16(a))<<16 + uint32(uint16(b))
	if recompMap == nil {
		bug("caller error") // see func comment
	}
	return recompMap[key]
}

func lookupInfoNFKC(b input, i int) Properties {
	bug("lookupInfoNFKC")
	v, sz := b.charinfoNFKC(i)
	return compInfo(v, sz)
}

// Properties returns properties for the first rune in s.
func (f Form) Properties(s []byte) Properties {
	bug("Properties")
	if f == nfc {
		return compInfo(nfcData.lookup(s))
	}
	return compInfo(nfkcData.lookup(s))
}

// PropertiesString returns properties for the first rune in s.
func (f Form) PropertiesString(s string) Properties {
	bug("PropertiesString")
	if f == nfc {
		return compInfo(nfcData.lookupString(s))
	}
	return compInfo(nfkcData.lookupString(s))
}

const (
	// Version is the Unicode edition from which the tables are derived.
	Version = "13.0.0"

	// MaxTransformChunkSize indicates the maximum number of bytes that Transform
	// may need to write atomically for any Form. Making a destination buffer at
	// least this size ensures that Transform can always make progress and that
	// the user does not need to grow the buffer on an ErrShortDst.
	MaxTransformChunkSize = 35 + maxNonStarters*4
)

const (
	firstMulti            = 0x1870
	firstCCC              = 0x2CAB
	endMulti              = 0x2F77
	firstLeadingCCC       = 0x49C5
	firstCCCZeroExcept    = 0x4A8F
	firstStarterWithNLead = 0x4AB6
	lastDecomp            = 0x4AB8
	maxDecomp             = 0x8000
)

// lookup returns the trie value for the first UTF-8 encoding in s and
// the width in bytes of this encoding. The size will be 0 if s does not
// hold enough bytes to complete the encoding. len(s) must be greater than 0.
func (t *nfcTrie) lookup(s []byte) (v uint16, sz int) {
	c0 := s[0]
	switch {
	case c0 < 0x80: // is ASCII
		return nfcValues[c0], 1
	case c0 < 0xC2:
		bug("lookup")
		return 0, 1 // Illegal UTF-8: not a starter, not ASCII.
	case c0 < 0xE0: // 2-byte UTF-8
		if len(s) < 2 {
			bug("lookup")
			return 0, 0
		}
		i := nfcIndex[c0]
		c1 := s[1]
		if c1 < 0x80 || 0xC0 <= c1 {
			bug("lookup")
			return 0, 1 // Illegal UTF-8: not a continuation byte.
		}
		return t.lookupValue(uint32(i), c1), 2
	case c0 < 0xF0: // 3-byte UTF-8
		if len(s) < 3 {
			bug("lookup")
			return 0, 0
		}
		i := nfcIndex[c0]
		c1 := s[1]
		if c1 < 0x80 || 0xC0 <= c1 {
			bug("lookup")
			return 0, 1 // Illegal UTF-8: not a continuation byte.
		}
		o := uint32(i)<<6 + uint32(c1)
		i = nfcIndex[o]
		c2 := s[2]
		if c2 < 0x80 || 0xC0 <= c2 {
			bug("lookup")
			return 0, 2 // Illegal UTF-8: not a continuation byte.
		}
		return t.lookupValue(uint32(i), c2), 3
	case c0 < 0xF8: // 4-byte UTF-8
		if len(s) < 4 {
			bug("lookup")
			return 0, 0
		}
		i := nfcIndex[c0]
		c1 := s[1]
		if c1 < 0x80 || 0xC0 <= c1 {
			bug("lookup")
			return 0, 1 // Illegal UTF-8: not a continuation byte.
		}
		o := uint32(i)<<6 + uint32(c1)
		i = nfcIndex[o]
		c2 := s[2]
		if c2 < 0x80 || 0xC0 <= c2 {
			bug("lookup")
			return 0, 2 // Illegal UTF-8: not a continuation byte.
		}
		o = uint32(i)<<6 + uint32(c2)
		i = nfcIndex[o]
		c3 := s[3]
		if c3 < 0x80 || 0xC0 <= c3 {
			bug("lookup")
			return 0, 3 // Illegal UTF-8: not a continuation byte.
		}
		return t.lookupValue(uint32(i), c3), 4
	}
	bug("lookup")
	// Illegal rune
	return 0, 1
}

// lookupUnsafe returns the trie value for the first UTF-8 encoding in s.
// s must start with a full and valid UTF-8 encoded rune.
func (t *nfcTrie) lookupUnsafe(s []byte) uint16 {
	bug("lookupUnsafe")
	c0 := s[0]
	if c0 < 0x80 { // is ASCII
		return nfcValues[c0]
	}
	i := nfcIndex[c0]
	if c0 < 0xE0 { // 2-byte UTF-8
		return t.lookupValue(uint32(i), s[1])
	}
	i = nfcIndex[uint32(i)<<6+uint32(s[1])]
	if c0 < 0xF0 { // 3-byte UTF-8
		return t.lookupValue(uint32(i), s[2])
	}
	i = nfcIndex[uint32(i)<<6+uint32(s[2])]
	if c0 < 0xF8 { // 4-byte UTF-8
		return t.lookupValue(uint32(i), s[3])
	}
	return 0
}

// lookupStringUnsafe returns the trie value for the first UTF-8 encoding in s.
// s must start with a full and valid UTF-8 encoded rune.
func (t *nfcTrie) lookupStringUnsafe(s string) uint16 {
	bug("lookupStringUnsafe")
	c0 := s[0]
	if c0 < 0x80 { // is ASCII
		return nfcValues[c0]
	}
	i := nfcIndex[c0]
	if c0 < 0xE0 { // 2-byte UTF-8
		return t.lookupValue(uint32(i), s[1])
	}
	i = nfcIndex[uint32(i)<<6+uint32(s[1])]
	if c0 < 0xF0 { // 3-byte UTF-8
		return t.lookupValue(uint32(i), s[2])
	}
	i = nfcIndex[uint32(i)<<6+uint32(s[2])]
	if c0 < 0xF8 { // 4-byte UTF-8
		return t.lookupValue(uint32(i), s[3])
	}
	return 0
}

type valueRange struct {
	value  uint16 // header: value:stride
	lo, hi byte   // header: lo:n
}

type sparseBlocks struct {
	values []valueRange
	offset []uint16
}

var nfcSparse = sparseBlocks{
	values: nfcSparseValues[:],
	offset: nfcSparseOffset[:],
}

var nfkcSparse = sparseBlocks{
	values: nfkcSparseValues[:],
	offset: nfkcSparseOffset[:],
}

var (
	nfcData  = newNfcTrie(0)
	nfkcData = newNfkcTrie(0)
)

// lookup returns the trie value for the first UTF-8 encoding in s and
// the width in bytes of this encoding. The size will be 0 if s does not
// hold enough bytes to complete the encoding. len(s) must be greater than 0.
func (t *nfkcTrie) lookup(s []byte) (v uint16, sz int) {
	bug("lookup")
	c0 := s[0]
	switch {
	case c0 < 0x80: // is ASCII
		return nfkcValues[c0], 1
	case c0 < 0xC2:
		return 0, 1 // Illegal UTF-8: not a starter, not ASCII.
	case c0 < 0xE0: // 2-byte UTF-8
		if len(s) < 2 {
			return 0, 0
		}
		i := nfkcIndex[c0]
		c1 := s[1]
		if c1 < 0x80 || 0xC0 <= c1 {
			return 0, 1 // Illegal UTF-8: not a continuation byte.
		}
		return t.lookupValue(uint32(i), c1), 2
	case c0 < 0xF0: // 3-byte UTF-8
		if len(s) < 3 {
			return 0, 0
		}
		i := nfkcIndex[c0]
		c1 := s[1]
		if c1 < 0x80 || 0xC0 <= c1 {
			return 0, 1 // Illegal UTF-8: not a continuation byte.
		}
		o := uint32(i)<<6 + uint32(c1)
		i = nfkcIndex[o]
		c2 := s[2]
		if c2 < 0x80 || 0xC0 <= c2 {
			return 0, 2 // Illegal UTF-8: not a continuation byte.
		}
		return t.lookupValue(uint32(i), c2), 3
	case c0 < 0xF8: // 4-byte UTF-8
		if len(s) < 4 {
			return 0, 0
		}
		i := nfkcIndex[c0]
		c1 := s[1]
		if c1 < 0x80 || 0xC0 <= c1 {
			return 0, 1 // Illegal UTF-8: not a continuation byte.
		}
		o := uint32(i)<<6 + uint32(c1)
		i = nfkcIndex[o]
		c2 := s[2]
		if c2 < 0x80 || 0xC0 <= c2 {
			return 0, 2 // Illegal UTF-8: not a continuation byte.
		}
		o = uint32(i)<<6 + uint32(c2)
		i = nfkcIndex[o]
		c3 := s[3]
		if c3 < 0x80 || 0xC0 <= c3 {
			return 0, 3 // Illegal UTF-8: not a continuation byte.
		}
		return t.lookupValue(uint32(i), c3), 4
	}
	// Illegal rune
	return 0, 1
}

// lookupUnsafe returns the trie value for the first UTF-8 encoding in s.
// s must start with a full and valid UTF-8 encoded rune.
func (t *nfkcTrie) lookupUnsafe(s []byte) uint16 {
	bug("lookupUnsafe")
	c0 := s[0]
	if c0 < 0x80 { // is ASCII
		return nfkcValues[c0]
	}
	i := nfkcIndex[c0]
	if c0 < 0xE0 { // 2-byte UTF-8
		return t.lookupValue(uint32(i), s[1])
	}
	i = nfkcIndex[uint32(i)<<6+uint32(s[1])]
	if c0 < 0xF0 { // 3-byte UTF-8
		return t.lookupValue(uint32(i), s[2])
	}
	i = nfkcIndex[uint32(i)<<6+uint32(s[2])]
	if c0 < 0xF8 { // 4-byte UTF-8
		return t.lookupValue(uint32(i), s[3])
	}
	return 0
}

// lookupString returns the trie value for the first UTF-8 encoding in s and
// the width in bytes of this encoding. The size will be 0 if s does not
// hold enough bytes to complete the encoding. len(s) must be greater than 0.
func (t *nfkcTrie) lookupString(s string) (v uint16, sz int) {
	bug("lookupString")
	c0 := s[0]
	switch {
	case c0 < 0x80: // is ASCII
		return nfkcValues[c0], 1
	case c0 < 0xC2:
		return 0, 1 // Illegal UTF-8: not a starter, not ASCII.
	case c0 < 0xE0: // 2-byte UTF-8
		if len(s) < 2 {
			return 0, 0
		}
		i := nfkcIndex[c0]
		c1 := s[1]
		if c1 < 0x80 || 0xC0 <= c1 {
			return 0, 1 // Illegal UTF-8: not a continuation byte.
		}
		return t.lookupValue(uint32(i), c1), 2
	case c0 < 0xF0: // 3-byte UTF-8
		if len(s) < 3 {
			return 0, 0
		}
		i := nfkcIndex[c0]
		c1 := s[1]
		if c1 < 0x80 || 0xC0 <= c1 {
			return 0, 1 // Illegal UTF-8: not a continuation byte.
		}
		o := uint32(i)<<6 + uint32(c1)
		i = nfkcIndex[o]
		c2 := s[2]
		if c2 < 0x80 || 0xC0 <= c2 {
			return 0, 2 // Illegal UTF-8: not a continuation byte.
		}
		return t.lookupValue(uint32(i), c2), 3
	case c0 < 0xF8: // 4-byte UTF-8
		if len(s) < 4 {
			return 0, 0
		}
		i := nfkcIndex[c0]
		c1 := s[1]
		if c1 < 0x80 || 0xC0 <= c1 {
			return 0, 1 // Illegal UTF-8: not a continuation byte.
		}
		o := uint32(i)<<6 + uint32(c1)
		i = nfkcIndex[o]
		c2 := s[2]
		if c2 < 0x80 || 0xC0 <= c2 {
			return 0, 2 // Illegal UTF-8: not a continuation byte.
		}
		o = uint32(i)<<6 + uint32(c2)
		i = nfkcIndex[o]
		c3 := s[3]
		if c3 < 0x80 || 0xC0 <= c3 {
			return 0, 3 // Illegal UTF-8: not a continuation byte.
		}
		return t.lookupValue(uint32(i), c3), 4
	}
	// Illegal rune
	return 0, 1
}

// nfkcTrie. Total size: 18768 bytes (18.33 KiB). Checksum: c51186dd2412943d.
type nfkcTrie struct{}

func newNfkcTrie(i int) *nfkcTrie {
	return &nfkcTrie{}
}

// lookupValue determines the type of block n and looks up the value for b.
func (t *nfkcTrie) lookupValue(n uint32, b byte) uint16 {
	bug("lookupValue")
	switch {
	case n < 92:
		return uint16(nfkcValues[n<<6+uint32(b)])
	default:
		n -= 92
		return uint16(nfkcSparse.lookup(n, b))
	}
}

// recompMap: 7528 bytes (entries only)
var recompMap map[uint32]rune
var recompMapOnce sync.Once
