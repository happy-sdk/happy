// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package humanize

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

const (
	// IEC Sizes in kibis of bits

	// Deprecated: Use github.com/happy-sdk/happy/pkg/bytesize instead.
	Byte = 1 << (iota * 10)

	// Deprecated: Use github.com/happy-sdk/happy/pkg/bytesize instead.
	KiByte

	// Deprecated: Use github.com/happy-sdk/happy/pkg/bytesize instead.
	MiByte

	// Deprecated: Use github.com/happy-sdk/happy/pkg/bytesize instead.
	GiByte

	// Deprecated: Use github.com/happy-sdk/happy/pkg/bytesize instead.
	TiByte

	// Deprecated: Use github.com/happy-sdk/happy/pkg/bytesize instead.
	PiByte

	// Deprecated: Use github.com/happy-sdk/happy/pkg/bytesize instead.
	EiByte
)

const (
	// SI Sizes

	// Deprecated: Use github.com/happy-sdk/happy/pkg/bytesize instead.
	IByte = 1

	// Deprecated: Use github.com/happy-sdk/happy/pkg/bytesize instead.
	KByte = IByte * 1000

	// Deprecated: Use github.com/happy-sdk/happy/pkg/bytesize instead.
	MByte = KByte * 1000

	// Deprecated: Use github.com/happy-sdk/happy/pkg/bytesize instead.
	GByte = MByte * 1000

	// Deprecated: Use github.com/happy-sdk/happy/pkg/bytesize instead.
	TByte = GByte * 1000

	// Deprecated: Use github.com/happy-sdk/happy/pkg/bytesize instead.
	PByte = TByte * 1000

	// Deprecated: Use github.com/happy-sdk/happy/pkg/bytesize instead.
	EByte = PByte * 1000

	// Deprecated: Use github.com/happy-sdk/happy/pkg/bytesize instead.
	ZByte = EByte * 1000
)

var sizeSymbs = map[string]uint64{
	"": Byte, "b": Byte,
	"kib": KiByte,
	"kb":  KByte,
	"mib": MiByte,
	"mb":  MByte,
	"gib": GiByte,
	"gb":  GByte,
	"tib": TiByte,
	"tb":  TByte,
	"pib": PiByte,
	"pb":  PByte,
	"eib": EiByte,
	"eb":  EByte,
	"ki":  KiByte,
	"k":   KByte,
	"mi":  MiByte,
	"m":   MByte,
	"gi":  GiByte,
	"g":   GByte,
	"ti":  TiByte,
	"t":   TByte,
	"pi":  PiByte,
	"p":   PByte,
	"ei":  EiByte,
	"e":   EByte,
}

// Bytes converts a byte count into a human-readable string using decimal (SI) units (kB, MB, GB, etc.).
// It uses a base of 1000 (1 kB = 1000 bytes).
//
// Deprecated: Use github.com/happy-sdk/happy/pkg/bytesize.ByteSI(s).String() instead.
func Bytes(s uint64) string {
	sizes := []string{"B", "kB", "MB", "GB", "TB", "PB", "EB"}
	return bytesToStr(s, 1000, sizes)
}

// IBytes converts a byte count into a human-readable string using binary (IEC) units (KiB, MiB, GiB, etc.).
// It uses a base of 1024 (1 KiB = 1024 bytes).
//
// Deprecated: Use github.com/happy-sdk/happy/pkg/bytesize.IECSize(s).String() instead.
func IBytes(s uint64) string {
	sizes := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	return bytesToStr(s, 1024, sizes)
}

// ParseBytes parses a string representation of bytes (like "1.5 GB" or "1024 B") and returns the number of bytes it represents.
// It supports both decimal (SI) and binary (IEC) units and handles values with commas as well.
//
// Deprecated: Use github.com/happy-sdk/happy/pkg/bytesize.Parse(s) instead.
func ParseBytes(s string) (uint64, error) {
	lastDigit := strings.IndexFunc(s, func(r rune) bool {
		return !unicode.IsDigit(r) && r != '.' && r != ','
	})

	if lastDigit == -1 {
		lastDigit = len(s)
	}

	num := s[:lastDigit]
	num = strings.ReplaceAll(num, ",", "")

	f, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return 0, err
	}

	extra := strings.ToLower(strings.TrimSpace(s[lastDigit:]))
	if m, ok := sizeSymbs[extra]; ok {
		f *= float64(m)
		if f >= math.MaxUint64 {
			return 0, fmt.Errorf("too large: %v", s)
		}
		return uint64(f), nil
	}

	return 0, fmt.Errorf("unhandled size name: %v", extra)
}

// tostr converts a byte count into a human-readable string with a specific base (1024 or 1000).
// It supports various units such as B, kB, MB, etc., and dynamically chooses the appropriate unit based on the size.
func bytesToStr(s uint64, base float64, sizes []string) string {
	if s < KByte {
		return strconv.FormatUint(s, 10) + " B"
	}

	e := math.Log(float64(s)) / math.Log(base)
	eFloor := math.Floor(e)
	power := math.Pow(base, eFloor)
	val := float64(s) / power

	// Check if the value is exactly a whole number or very close to the next unit
	isWholeNumber := math.Abs(val-math.Round(val)) < 0.1
	nextUnitThreshold := math.Pow(base, eFloor+1) - power*0.05

	// Adjust eFloor and val if the value is very close to the next unit
	if float64(s) >= nextUnitThreshold {
		eFloor++
		val = float64(s) / math.Pow(base, eFloor)
		isWholeNumber = true // Force whole number formatting
	}

	format := "%.0f %s"
	if !isWholeNumber {
		format = "%.1f %s"
	}

	return fmt.Sprintf(format, val, sizes[int(eFloor)])
}
