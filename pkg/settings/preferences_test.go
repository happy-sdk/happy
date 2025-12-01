// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package settings

import (
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"github.com/happy-sdk/happy/pkg/version"
)

func TestNewPreferences(t *testing.T) {
	v := version.Version("v1.0.0")
	prefs := NewPreferences(v)
	testutils.NotNil(t, prefs)
	testutils.Equal(t, v, prefs.SchemaVersion())
}

func TestPreferences_Set(t *testing.T) {
	prefs := NewPreferences(version.Version("v1.0.0"))
	prefs.Set("test.key", "test value")
	
	// Verify by encoding and decoding
	data, err := prefs.GobEncode()
	testutils.NoError(t, err)
	
	var prefs2 Preferences
	err = prefs2.GobDecode(data)
	testutils.NoError(t, err)
	testutils.Equal(t, prefs.version, prefs2.version)
}

func TestPreferences_GobEncode(t *testing.T) {
	prefs := NewPreferences(version.Version("v1.0.0"))
	prefs.Set("key1", "value1")
	prefs.Set("key2", "value2")
	
	data, err := prefs.GobEncode()
	testutils.NoError(t, err)
	testutils.Assert(t, len(data) > 0, "expected non-empty encoded data")
}

func TestPreferences_GobDecode(t *testing.T) {
	prefs := NewPreferences(version.Version("v1.0.0"))
	prefs.Set("key1", "value1")
	
	data, err := prefs.GobEncode()
	testutils.NoError(t, err)
	
	var prefs2 Preferences
	err = prefs2.GobDecode(data)
	testutils.NoError(t, err)
	testutils.Equal(t, prefs.version, prefs2.version)
}

func TestPreferences_GobDecode_Empty(t *testing.T) {
	var prefs Preferences
	err := prefs.GobDecode([]byte{})
	testutils.NoError(t, err)
	testutils.Equal(t, version.Version("v1.0.0"), prefs.version)
}

func TestPreferences_UnmarshalJSON(t *testing.T) {
	jsonData := []byte(`{"version":"v1.0.0","key1":"value1","key2":"value2"}`)
	var prefs Preferences
	err := prefs.UnmarshalJSON(jsonData)
	testutils.NoError(t, err)
	testutils.Equal(t, version.Version("v1.0.0"), prefs.version)
}

func TestPreferences_UnmarshalJSON_Nested(t *testing.T) {
	jsonData := []byte(`{"version":"v1.0.0","group":{"key1":"value1","key2":"value2"}}`)
	var prefs Preferences
	err := prefs.UnmarshalJSON(jsonData)
	testutils.NoError(t, err)
	testutils.Equal(t, version.Version("v1.0.0"), prefs.version)
}

func TestPreferences_UnmarshalJSON_NoVersion(t *testing.T) {
	jsonData := []byte(`{"key1":"value1"}`)
	var prefs Preferences
	err := prefs.UnmarshalJSON(jsonData)
	testutils.Error(t, err)
}

func TestPreferences_UnmarshalJSON_InvalidVersion(t *testing.T) {
	jsonData := []byte(`{"version":"invalid"}`)
	var prefs Preferences
	err := prefs.UnmarshalJSON(jsonData)
	testutils.Error(t, err)
}

func TestParseNested(t *testing.T) {
	obj := map[string]any{
		"key1": "value1",
		"key2": map[string]any{
			"nested": "value",
		},
	}
	result := parseNested("prefix", obj)
	testutils.Assert(t, len(result) > 0, "expected non-empty result")
	testutils.ContainsKey(t, result, "prefix.key1")
}

