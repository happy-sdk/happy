// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package vars

// export access to vars internals for tests

func SetOptimize(b bool) bool {
	old := optimize
	optimize = b
	return old
}

func IsHost32bit() bool {
	return host32bit
}

func SetHost32bit() {
	host32bit = true
}

func RestoreHost32bit() {
	host32bit = ^uint(0)>>32 == 0
}

func ParseUint(str string, base, bitSize int) (r uint64, s string, err error) {
	return parseUint(str, base, bitSize)
}

func ParseInt(str string, base, bitSize int) (r int64, s string, err error) {
	return parseInt(str, base, bitSize)
}
