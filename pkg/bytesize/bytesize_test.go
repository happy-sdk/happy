// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package bytesize

import (
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func TestIECString(t *testing.T) {
	data := testutils.NamedData[IECSize, string]{
		{"Exactly 1 KiB", KiB, "1 KiB"},
		{"Exactly 1 MiB", MiB, "1 MiB"},
		{"Exactly 1 GiB", GiB, "1 GiB"},
		{"Exactly 1 TiB", TiB, "1 TiB"},
		{"Exactly 1 PiB", PiB, "1 PiB"},
		{"Exactly 1 EiB", EiB, "1 EiB"},
		{"Half a KiB", KiB / 2, "512 B"},
		{"Just below 1 KiB", KiB - 1, "1023 B"},
		{"1.5 KiB", KiB + KiB/2, "1.5 KiB"},
		{"Just below 1 MiB", MiB - 1, "1 MiB"},
		{"10 MiB", 10 * MiB, "10 MiB"},
		{"100.5 MiB", 100*MiB + MiB/2, "100.5 MiB"},
		{"1 GiB plus 1 Byte", GiB + 1, "1 GiB"},
		{"500 TiB", 500 * TiB, "500 TiB"},
		{"Exactly 1 PiB", PiB, "1 PiB"},
		{"Exactly 1 EiB", EiB, "1 EiB"},
	}
	for _, test := range data {
		testutils.Equal(t, test.Want, test.In.String(), test.Name)
	}
}

func TestSIString(t *testing.T) {
	data := testutils.NamedData[SISize, string]{
		{"Exactly 1 kB", KB, "1 kB"},
		{"Exactly 1 MB", MB, "1 MB"},
		{"Exactly 1 GB", GB, "1 GB"},
		{"Exactly 1 TB", TB, "1 TB"},
		{"Exactly 1 PB", PB, "1 PB"},
		{"Exactly 1 EB", EB, "1 EB"},
		{"Half a kB", KB / 2, "500 B"},
		{"Just below 1 kB", KB - 1, "999 B"},
		{"1.5 kB", KB + KB/2, "1.5 kB"},
		{"Just below 1 MB", MB - 1, "1 MB"},
		{"10 MB", 10 * MB, "10 MB"},
		{"100.5 MB", 100*MB + MB/2, "100.5 MB"},
		{"1 GB plus 1 Byte", GB + 1, "1 GB"},
		{"500 TB", 500 * TB, "500 TB"},
		{"Exactly 1 PB", PB, "1 PB"},
		{"Exactly 1 EB", EB, "1 EB"},
	}
	for _, test := range data {
		testutils.Equal(t, test.Want, test.In.String(), test.Name)
	}
}

func TestParse(t *testing.T) {
	data := testutils.Data[string, uint64]{
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

		{"1,005.03 MB", 1005030000},

		{"12.5 EB", uint64(12.5 * float64(EB))},
		{"12.5 E", uint64(12.5 * float64(EB))},
		{"12.5 EiB", uint64(12.5 * float64(EiB))},
	}
	for _, p := range data {
		got, err := Parse(p.In)
		testutils.NoError(t, err)
		testutils.Equal(t, p.Want, got.Bytes())
		testutils.Equal(t, p.Want, got.Bytes())
	}
}

func TestByteErrors(t *testing.T) {
	got, err := Parse("84 JB")
	if err == nil {
		t.Errorf("Expected error, got %v", got)
	}
	_, err = Parse("")
	if err == nil {
		t.Errorf("Expected error parsing nothing")
	}
	got, err = Parse("16 EiB")
	if err == nil {
		t.Errorf("Expected error, got %v", got)
	}
}
