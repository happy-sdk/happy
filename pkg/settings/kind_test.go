// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package settings

import (
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func TestKind_String(t *testing.T) {
	tests := []struct {
		name string
		kind Kind
		want string
	}{
		{"Custom", KindCustom, "custom"},
		{"Settings", KindSettings, "settings"},
		{"Bool", KindBool, "bool"},
		{"Int", KindInt, "int"},
		{"Uint", KindUint, "uint"},
		{"String", KindString, "string"},
		{"Duration", KindDuration, "duration"},
		{"StringSlice", KindStringSlice, "slice"},
		{"Invalid", KindInvalid, "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.kind.String()
			testutils.Equal(t, tt.want, got)
		})
	}
}

