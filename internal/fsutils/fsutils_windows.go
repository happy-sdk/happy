// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

//go:build windows

package fsutils

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

func AvailableSpace(path string) (uint64, error) {
	lpFreeBytesAvailable := int64(0)
	lpTotalNumberOfBytes := int64(0)
	lpTotalNumberOfFreeBytes := int64(0)

	drive := windows.StringToUTF16Ptr(path)

	ret, _, err := windows.GetDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(drive)),
		uintptr(unsafe.Pointer(&lpFreeBytesAvailable)),
		uintptr(unsafe.Pointer(&lpTotalNumberOfBytes)),
		uintptr(unsafe.Pointer(&lpTotalNumberOfFreeBytes)),
	)
	if ret == 0 {
		return 0, err
	}

	return uint64(lpFreeBytesAvailable), nil
}
