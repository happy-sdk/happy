package settings

import (
	"fmt"
	"testing"
)

func TestProfile(t *testing.T) {
	// Create a sample schema for testing
	definitions := []Definition{
		Immutable("immutable-setting", "immutable-value", KindString),
		Once("mutable-once-setting", "mutable-once-value", KindString),
		Mutable("mutable-setting", "mutable-value", KindString),
	}

	blueprint := New()
	for _, d := range definitions {
		if err := blueprint.Define(d); err != nil {
			t.Fatalf("Failed to define setting: %v", err)
		}
	}
	schema, err := blueprint.Schema("1.0.0")
	if err != nil {
		t.Fatalf("Failed to compile schema: %v", err)
	}

	profile, err := schema.Profile(Default)
	if err != nil {
		t.Fatalf("Failed to create profile: %v", err)
	}

	t.Run("Version", func(t *testing.T) {
		version := profile.Version()
		if version != "1.0.0" {
			t.Errorf("Expected version '1.0.0', but got '%s'", version)
		}
	})

	t.Run("Get", func(t *testing.T) {
		immutable, err := profile.Get("immutable-setting")
		if err != nil {
			t.Fatalf("Failed to get immutable setting: %v", err)
		}
		if immutable.String() != "immutable-value" {
			t.Errorf("Expected 'immutable-value', but got '%v'", immutable.v.Any())
		}

		mutableonce, err := profile.Get("mutable-once-setting")
		if err != nil {
			t.Fatalf("Failed to get mutable-once setting: %v", err)
		}
		if mutableonce.String() != "mutable-once-value" {
			t.Errorf("Expected 'mutable-once-value', but got '%v'", mutableonce.v.Any())
		}

		mutable, err := profile.Get("mutable-setting")
		if err != nil {
			t.Fatalf("Failed to get mutable setting: %v", err)
		}
		if mutable.v.Any() != "mutable-value" {
			t.Errorf("Expected 'mutable-value', but got '%v'", mutable.v.Any())
		}

		notFound, err := profile.Get("non-existent-setting")
		if err == nil {
			t.Errorf("Expected an error for non-existent setting, but got none")
		}
		if notFound.String() != "" {
			t.Errorf("Expected a empty value for non-existent setting, but got '%v'", notFound.v.Any())
		}
	})

	t.Run("Reset", func(t *testing.T) {
		mutable, err := profile.Get("mutable-setting")
		if err != nil {
			t.Fatalf("Failed to get mutable setting for reset: %v", err)
		}

		err = profile.Reset("mutable-setting")
		if err != nil {
			t.Fatalf("Failed to reset mutable setting: %v", err)
		}

		// Verify that the setting has been reset
		mutable, err = profile.Get("mutable-setting")
		if err != nil {
			t.Fatalf("Failed to get mutable setting after reset: %v", err)
		}
		if mutable.v.Any() != "mutable-value" {
			t.Errorf("Expected 'mutable-value' after reset, but got '%v'", mutable.v.Any())
		}
	})
}

func TestMutability(t *testing.T) {
	// Create a sample schema for testing
	definitions := []Definition{
		Immutable("immutable-setting", "immutable-value", KindString),
		Once("mutable-once-setting", "mutable-once-value", KindString),
		Mutable("mutable-setting", "mutable-value", KindString),
	}

	blueprint := New()
	for _, d := range definitions {
		if err := blueprint.Define(d); err != nil {
			t.Fatalf("Failed to define setting: %v", err)
		}
	}
	schema, err := blueprint.Schema("1.0.0")
	if err != nil {
		t.Fatalf("Failed to compile schema: %v", err)
	}

	profile, err := schema.Profile(Default)
	if err != nil {
		t.Fatalf("Failed to create profile: %v", err)
	}

	t.Run("Immutable", func(t *testing.T) {
		immutable, err := profile.Get("immutable-setting")
		if err != nil {
			t.Fatalf("Failed to get immutable setting: %v", err)
		}

		err = profile.Set("immutable-setting", "new-value")
		if err == nil {
			t.Errorf("Expected an error when setting immutable setting, but got none")
		}

		// Verify that the setting has not been changed
		immutable, err = profile.Get("immutable-setting")
		if err != nil {
			t.Fatalf("Failed to get immutable setting after set attempt: %v", err)
		}
		if immutable.v.Any() != "immutable-value" {
			t.Errorf("Expected 'immutable-value' after set attempt, but got '%v'", immutable.v.Any())
		}
	})

	t.Run("Once", func(t *testing.T) {
		mutableonce, err := profile.Get("mutable-once-setting")
		if err != nil {
			t.Fatalf("Failed to get mutable-once setting: %v", err)
		}

		err = profile.Set("mutable-once-setting", "new-value")
		if err != nil {
			t.Fatalf("Failed to set mutable-once setting: %v", err)
		}

		// Verify that the setting has been set
		mutableonce, err = profile.Get("mutable-once-setting")
		if err != nil {
			t.Fatalf("Failed to get mutable-once setting after set: %v", err)
		}
		if mutableonce.v.Any() != "new-value" {
			t.Errorf("Expected 'new-value' after set, but got '%v'", mutableonce.v.Any())
		}

		// Try setting the setting again
		err = profile.Set("mutable-once-setting", "new-value-again")
		if err == nil {
			t.Errorf("Expected an error when setting mutable-once setting a second time, but got none")
		}
	})

	t.Run("Mutable", func(t *testing.T) {
		mutable, err := profile.Get("mutable-setting")
		if err != nil {
			t.Fatalf("Failed to get mutable setting: %v", err)
		}

		// Modify the mutable setting multiple times using the Set method
		for i := 1; i <= 3; i++ {
			err := profile.Set("mutable-setting", fmt.Sprintf("new-value-%d", i))
			if err != nil {
				t.Fatalf("Failed to set mutable setting (iteration %d): %v", i, err)
			}

			// Verify that the setting has been set
			mutable, err = profile.Get("mutable-setting")
			if err != nil {
				t.Fatalf("Failed to get mutable setting (iteration %d) after set: %v", i, err)
			}
			expectedValue := fmt.Sprintf("new-value-%d", i)
			if mutable.v.Any() != expectedValue {
				t.Errorf("Expected '%s' (iteration %d) after set, but got '%v'", expectedValue, i, mutable.v.Any())
			}
		}
	})
}

func TestGroupedSettings(t *testing.T) {
	// Create a new Blueprint and define settings
	blueprint := New()
	definitions := []Definition{
		Immutable("group1/setting1", "value1", KindString),
		Mutable("group1/setting2", "value2", KindString),
		Immutable("group2/setting1", "value3", KindString),
		Mutable("group2/setting2", "value4", KindString),
	}

	for _, d := range definitions {
		if err := blueprint.Define(d); err != nil {
			t.Fatalf("Error defining setting: %v", err)
		}
	}

	// Compile a schema for the application version
	schema, err := blueprint.Schema("1.0.0")
	if err != nil {
		t.Fatalf("Error creating schema: %v", err)
	}

	// Create a profile from the schema
	profile, err := schema.Profile(Default)
	if err != nil {
		t.Fatalf("Error creating profile: %v", err)
	}

	// Test getting settings within existing groups
	{
		// Test getting an immutable setting
		_, err := profile.Get("group1/setting1")
		if err != nil {
			t.Errorf("Failed to get setting: %v", err)
		} else {
			// Add assertions here to verify the value
		}

		// Test getting a mutable setting
		_, err = profile.Get("group1/setting2")
		if err != nil {
			t.Errorf("Failed to get setting: %v", err)
		} else {
			// Add assertions here to verify the value
		}
	}

	// Test getting settings within a non-existent group
	{
		_, err := profile.Get("nonexistentgroup/setting1")
		if err == nil {
			t.Error("Expected an error for a non-existent group, but got none.")
		}

		// Add more tests and assertions as needed
	}
}

func TestGroupNotFound(t *testing.T) {
	// Create a new Blueprint and define settings
	blueprint := New()
	definitions := []Definition{
		Immutable("group1/setting1", "value1", KindString),
		Mutable("group1/setting2", "value2", KindString),
	}

	for _, d := range definitions {
		if err := blueprint.Define(d); err != nil {
			t.Fatalf("Error defining setting: %v", err)
		}
	}

	// Compile a schema for the application version
	schema, err := blueprint.Schema("1.0.0")
	if err != nil {
		t.Fatalf("Error creating schema: %v", err)
	}

	// Create a profile from the schema
	profile, err := schema.Profile(Default)
	if err != nil {
		t.Fatalf("Error creating profile: %v", err)
	}

	// Test getting a setting within a non-existent group
	_, err = profile.Get("nonexistentgroup/setting1")
	if err == nil {
		t.Error("Expected an error for a non-existent group, but got none.")
	}

	// Add more tests and assertions as needed
}
