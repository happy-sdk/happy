// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package humanize

import (
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func TestByteParsing(t *testing.T) {
	tests := []struct {
		in  string
		exp uint64
	}{
		{"42", 42},
		{"42MB", 42000000},
		{"42MiB", 44040192},
		{"42mb", 42000000},
		{"42mib", 44040192},
		{"42MIB", 44040192},
		{"42 MB", 42000000},
		{"42 MiB", 44040192},
		{"42 mb", 42000000},
		{"42 mib", 44040192},
		{"42 MIB", 44040192},
		{"42.5MB", 42500000},
		{"42.5MiB", 44564480},
		{"42.5 MB", 42500000},
		{"42.5 MiB", 44564480},
		// No need to say B
		{"42M", 42000000},
		{"42Mi", 44040192},
		{"42m", 42000000},
		{"42mi", 44040192},
		{"42MI", 44040192},
		{"42 M", 42000000},
		{"42 Mi", 44040192},
		{"42 m", 42000000},
		{"42 mi", 44040192},
		{"42 MI", 44040192},
		{"42.5M", 42500000},
		{"42.5Mi", 44564480},
		{"42.5 M", 42500000},
		{"42.5 Mi", 44564480},
		// Bug #42
		{"1,005.03 MB", 1005030000},
		// Large testing, breaks when too much larger than
		// this.
		{"12.5 EB", uint64(12.5 * float64(EByte))},
		{"12.5 E", uint64(12.5 * float64(EByte))},
		{"12.5 EiB", uint64(12.5 * float64(EiByte))},
	}

	for _, p := range tests {
		got, err := ParseBytes(p.in)
		if err != nil {
			t.Errorf("Couldn't parse %v: %v", p.in, err)
		}
		if got != p.exp {
			t.Errorf("Expected %v for %v, got %v",
				p.exp, p.in, got)
		}
	}
}

func TestByteErrors(t *testing.T) {
	got, err := ParseBytes("84 JB")
	if err == nil {
		t.Errorf("Expected error, got %v", got)
	}
	_, err = ParseBytes("")
	if err == nil {
		t.Errorf("Expected error parsing nothing")
	}
	got, err = ParseBytes("16 EiB")
	if err == nil {
		t.Errorf("Expected error, got %v", got)
	}
}

type Tests struct {
	Name     string
	In       string
	Expected string
}

func TestIBytes(t *testing.T) {
	data := testutils.Data[uint64, string]{
		{"Exactly 1 KiB", KiByte, "1 KiB"},
		{"Exactly 1 MiB", MiByte, "1 MiB"},
		{"Exactly 1 GiB", GiByte, "1 GiB"},
		{"Exactly 1 TiB", TiByte, "1 TiB"},
		{"Exactly 1 PiB", PiByte, "1 PiB"},
		{"Exactly 1 EiB", EiByte, "1 EiB"},
		{"Half a KiB", KiByte / 2, "512 B"},
		{"Just below 1 KiB", KiByte - 1, "1023 B"},
		{"1.5 KiB", KiByte + KiByte/2, "1.5 KiB"},
		{"Just below 1 MiB", MiByte - 1, "1 MiB"},
		{"10 MiB", 10 * MiByte, "10 MiB"},
		{"100.5 MiB", 100*MiByte + MiByte/2, "100.5 MiB"},
		{"1 GiB plus 1 Byte", GiByte + 1, "1 GiB"},
		{"500 TiB", 500 * TiByte, "500 TiB"},
	}
	for _, test := range data {
		testutils.Equal(t, test.Want, IBytes(test.In), test.Name)
	}
}
func TestBytes(t *testing.T) {
	data := testutils.Data[uint64, string]{
		{"Exactly 1 kB", KByte, "1 kB"},
		{"Exactly 1 MB", MByte, "1 MB"},
		{"Exactly 1 GB", GByte, "1 GB"},
		{"Exactly 1 TB", TByte, "1 TB"},
		{"Exactly 1 PB", PByte, "1 PB"},
		{"Exactly 1 EB", EByte, "1 EB"},
		{"Half a kB", KByte / 2, "500 B"},
		{"Just below 1 kB", KByte - 1, "999 B"},
		{"1.5 kB", KByte + KByte/2, "1.5 kB"},
		{"Just below 1 MB", MByte - 1, "1 MB"},
		{"10 MB", 10 * MByte, "10 MB"},
		{"100.5 MB", 100*MByte + MByte/2, "100.5 MB"},
		{"1 GB plus 1 Byte", GByte + 1, "1 GB"},
		// Edge case: very large number
		{"500 TB", 500 * TByte, "500 TB"},
	}
	for _, test := range data {
		testutils.Equal(t, test.Want, Bytes(test.In), test.Name)
	}
}
