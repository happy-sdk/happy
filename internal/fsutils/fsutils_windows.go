// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

//go:build windows

package fsutils

import (
	"golang.org/x/sys/windows"
)

func AvailableSpace(path string) (uint64, error) {
	lpFreeBytesAvailable := uint64(0)
	lpTotalNumberOfBytes := uint64(0)
	lpTotalNumberOfFreeBytes := uint64(0)

	drive := windows.StringToUTF16Ptr(path)

	err := windows.GetDiskFreeSpaceEx(
		drive,
		&lpFreeBytesAvailable,
		&lpTotalNumberOfBytes,
		&lpTotalNumberOfFreeBytes,
	)
	return lpFreeBytesAvailable, err
}
