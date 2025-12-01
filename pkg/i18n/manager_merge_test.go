// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"golang.org/x/text/language"
)

func TestQueueTranslation_MergeNestedObjects(t *testing.T) {
	Initialize(language.English)

	// Queue SDK default translations with nested structure
	sdkTranslations := map[string]any{
		"app": map[string]any{
			"description": "SDK default description",
		},
	}
	err := QueueTranslations(language.English, sdkTranslations)
	testutils.NoError(t, err)

	// Queue application translations that should merge with SDK defaults
	appTranslations := map[string]any{
		"app": map[string]any{
			"name": "My Application",
		},
	}
	err = QueueTranslations(language.English, appTranslations)
	testutils.NoError(t, err)

	// Reload to process queue
	Reload()

	// Verify both SDK default and application values are present (merged)
	result1 := T("app.description")
	testutils.Equal(t, "SDK default description", result1, "expected SDK default to be preserved")

	result2 := T("app.name")
	testutils.Equal(t, "My Application", result2, "expected application value to be added")
}

func TestQueueTranslation_OverwriteString(t *testing.T) {
	Initialize(language.English)

	// Queue initial string translation
	err := QueueTranslation(language.English, "simple.key", "Original Value")
	testutils.NoError(t, err)

	// Queue replacement string (should overwrite, not merge)
	err = QueueTranslation(language.English, "simple.key", "New Value")
	testutils.NoError(t, err)

	// Reload to process queue
	Reload()

	// Verify the overwritten value is used
	result := T("simple.key")
	testutils.Equal(t, "New Value", result, "expected string to be overwritten, not merged")
}

func TestQueueTranslation_MergeDeepNested(t *testing.T) {
	Initialize(language.English)

	// Queue SDK translations with deep nesting
	sdkTranslations := map[string]any{
		"app": map[string]any{
			"info": map[string]any{
				"version": "1.0.0",
				"author":  "SDK Team",
			},
		},
	}
	err := QueueTranslations(language.English, sdkTranslations)
	testutils.NoError(t, err)

	// Queue application translations that should merge at nested level
	appTranslations := map[string]any{
		"app": map[string]any{
			"info": map[string]any{
				"name": "My App",
			},
		},
	}
	err = QueueTranslations(language.English, appTranslations)
	testutils.NoError(t, err)

	// Reload to process queue
	Reload()

	// Verify all values are present (merged at all levels)
	result1 := T("app.info.version")
	testutils.Equal(t, "1.0.0", result1, "expected SDK value to be preserved")

	result2 := T("app.info.author")
	testutils.Equal(t, "SDK Team", result2, "expected SDK value to be preserved")

	result3 := T("app.info.name")
	testutils.Equal(t, "My App", result3, "expected application value to be added")
}

func TestQueueTranslation_StringOverwritesMap(t *testing.T) {
	Initialize(language.English)

	// Queue a map value
	err := QueueTranslation(language.English, "mixed.key", map[string]any{
		"nested": "value",
	})
	testutils.NoError(t, err)

	// Queue a string value (should overwrite the map)
	err = QueueTranslation(language.English, "mixed.key", "Simple String")
	testutils.NoError(t, err)

	// Reload to process queue
	Reload()

	// Verify the string value is used (map was overwritten)
	result := T("mixed.key")
	testutils.Equal(t, "Simple String", result, "expected string to overwrite map")
}

func TestQueueTranslation_MapOverwritesString(t *testing.T) {
	Initialize(language.English)

	// Queue a string value
	err := QueueTranslation(language.English, "mixed.key", "Simple String")
	testutils.NoError(t, err)

	// Queue a map value (should overwrite the string)
	err = QueueTranslation(language.English, "mixed.key", map[string]any{
		"nested": "value",
	})
	testutils.NoError(t, err)

	// Reload to process queue
	Reload()

	// Verify the map value is used (string was overwritten)
	result := T("mixed.key.nested")
	testutils.Equal(t, "value", result, "expected map to overwrite string")
}
