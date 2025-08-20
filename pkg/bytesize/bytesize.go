// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package bytesize

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// Size is an interface for byte sizes, providing human-readable string and raw bytes.
type Size interface {
	String() string
	Bytes() uint64
}

// IECSize represents IEC (base-2) byte sizes (1 KiB = 2^10 bytes).
type IECSize uint64

// IEC (base-2) byte size constants.
const (
	Byte IECSize = 1
	KiB  IECSize = 1 << (iota * 10) // 2^10 = 1024 (Kibibyte)
	MiB                             // 2^20 = 1048576 (Mebibyte)
	GiB                             // 2^30 = 1073741824 (Gibibyte)
	TiB                             // 2^40 = 1099511627776 (Tebibyte)
	PiB                             // 2^50 = 1125899906842624 (Pebibyte)
	EiB                             // 2^60 = 1152921504606846976 (Exbibyte)

	// *big.Int
	// ZiB                             // 2^70 = 1180591620717411303424 (Zebibyte)
	// YiB                             // 2^80 = 1208925819614629174706176 (Yobibyte)
)

// String converts an IECSize to a human-readable string (base 1024).
func (s IECSize) String() string {
	return bytesToStr(uint64(s), 1024, iecSizes)
}

// Bytes returns the raw byte count.
func (s IECSize) Bytes() uint64 {
	return uint64(s)
}

func (s IECSize) MarshalSetting() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s *IECSize) UnmarshalSetting(data []byte) error {
	val, err := Parse(string(data))
	if err != nil {
		return err
	}
	*s = IECSize(val.Bytes())
	return nil
}

// SISize represents SI (base-10) byte sizes (1 KB = 10^3 bytes).
type SISize uint64

// SI (base-10) byte size constants.
const (
	ByteSI SISize = 1
	KB     SISize = 1000 * ByteSI // 10^3 = 1000 (Kilobyte)
	MB     SISize = 1000 * KB     // 10^6 = 1000000 (Megabyte)
	GB     SISize = 1000 * MB     // 10^9 = 1000000000 (Gigabyte)
	TB     SISize = 1000 * GB     // 10^12 = 1000000000000 (Terabyte)
	PB     SISize = 1000 * TB     // 10^15 = 1000000000000000 (Petabyte)
	EB     SISize = 1000 * PB     // 10^18 = 1000000000000000000 (Exabyte)

	// *big.Int
	// ZB     SISize = 1000 * EB     // 10^21 = 1000000000000000000000 (Zettabyte)
	// YB     SISize = 1000 * ZB     // 10^24 = 1000000000000000000000000 (Yottabyte)
)

// String converts an SISize to a human-readable string (base 1000).
func (s SISize) String() string {
	return bytesToStr(uint64(s), 1000, siSizes)
}

// Bytes returns the raw byte count.
func (s SISize) Bytes() uint64 {
	return uint64(s)
}

func (s SISize) MarshalSetting() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s *SISize) UnmarshalSetting(data []byte) error {
	val, err := Parse(string(data))
	if err != nil {
		return err
	}
	*s = SISize(val.Bytes())
	return nil
}

var (
	siSizes  = []string{"B", "kB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}
	iecSizes = []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"}
	siSymbs  = map[string]uint64{
		"b":  uint64(ByteSI),
		"kb": uint64(KB),
		"mb": uint64(MB),
		"gb": uint64(GB),
		"tb": uint64(TB),
		"pb": uint64(PB),
		"eb": uint64(EB),
		// "zb": uint64(ZB),
		// "yb": uint64(YB),
		"k": uint64(KB),
		"m": uint64(MB),
		"g": uint64(GB),
		"t": uint64(TB),
		"p": uint64(PB),
		"e": uint64(EB),
		// "z":  uint64(ZB),
		// "y":  uint64(YB),
	}
	iecSymbs = map[string]uint64{
		"":    uint64(Byte),
		"kib": uint64(KiB),
		"mib": uint64(MiB),
		"gib": uint64(GiB),
		"tib": uint64(TiB),
		"pib": uint64(PiB),
		"eib": uint64(EiB),
		// "zib": uint64(ZiB),
		// "yib": uint64(YiB),
		"ki": uint64(KiB),
		"mi": uint64(MiB),
		"gi": uint64(GiB),
		"ti": uint64(TiB),
		"pi": uint64(PiB),
		"ei": uint64(EiB),
		// "zi":  uint64(ZiB),
		// "yi":  uint64(YiB),
	}
)

// Parse parses a string like "1.5 GB" or "1024 MiB" into a Size (SISize or IECSize).
func Parse(s string) (Size, error) {
	lastDigit := strings.IndexFunc(s, func(r rune) bool {
		return !unicode.IsDigit(r) && r != '.' && r != ','
	})
	if lastDigit == -1 {
		lastDigit = len(s)
	}

	num := strings.ReplaceAll(s[:lastDigit], ",", "")
	f, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid number: %w", err)
	}

	extra := strings.ToLower(strings.TrimSpace(s[lastDigit:]))

	var m uint64
	var isSI bool
	if val, ok := siSymbs[extra]; ok {
		m = val
		isSI = true
	} else if val, ok := iecSymbs[extra]; ok {
		m = val
		isSI = false
	} else {
		return nil, fmt.Errorf("unknown unit: %v", extra)
	}

	f *= float64(m)
	if f >= math.MaxUint64 {
		return nil, fmt.Errorf("too large: %v", s)
	}
	bytesVal := uint64(f)

	if isSI {
		return SISize(bytesVal), nil
	}
	return IECSize(bytesVal), nil
}

// bytesToStr converts a byte count to a human-readable string with the given base and units.
func bytesToStr(s uint64, base float64, sizes []string) string {
	if float64(s) < base {
		return strconv.FormatUint(s, 10) + " B"
	}

	e := math.Log(float64(s)) / math.Log(base)
	eFloor := math.Floor(e)
	if int(eFloor) >= len(sizes) {
		eFloor = float64(len(sizes) - 1) // Cap at largest unit
	}
	power := math.Pow(base, eFloor)
	val := float64(s) / power

	// Check if value is a whole number or close to the next unit.
	isWholeNumber := math.Abs(val-math.Round(val)) < 0.1
	nextUnitThreshold := math.Pow(base, eFloor+1) - power*0.05

	if float64(s) >= nextUnitThreshold && int(eFloor) < len(sizes)-1 {
		eFloor++
		val = float64(s) / math.Pow(base, eFloor)
		isWholeNumber = true
	}

	if isWholeNumber {
		return fmt.Sprintf("%.0f %s", val, sizes[int(eFloor)])
	}
	return fmt.Sprintf("%.1f %s", val, sizes[int(eFloor)])
}
