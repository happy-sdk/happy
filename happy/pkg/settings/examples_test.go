package settings_test

import (
	"fmt"

	"github.com/happy-sdk/settings"
)

func ExampleNew() {
	// Create a new blueprint.
	blueprint := settings.New()

	definitions := []settings.Definition{
		settings.Immutable("immutable-setting", "immutable-value", settings.KindString),
		settings.Once("mutable-once-setting", "mutable-once-value", settings.KindString),
		settings.Mutable("mutable-setting", "mutable-value", settings.KindString),
	}

	for _, d := range definitions {
		if err := blueprint.Define(d); err != nil {
			panic(err)
		}
	}

	// Compile a schema for current app version.
	schema, err := blueprint.Schema("1.0.0")
	if err != nil {
		panic(err)
	}

	// Create a default settings profile.
	profile, err := schema.Profile(settings.Default)
	if err != nil {
		panic(err)
	}

	immutable, err := profile.Get("immutable-setting")
	if err != nil {
		panic(err)
	}

	mutableonce, err := profile.Get("mutable-once-setting")
	if err != nil {
		panic(err)
	}

	mutable, err := profile.Get("mutable-setting")
	if err != nil {
		panic(err)
	}

	// print the setting values.
	fmt.Println(profile.Version())
	fmt.Println(immutable.Attr())
	fmt.Println(mutableonce.Attr())
	fmt.Println(mutable.Attr())

	// OUTPUT:
	// 1.0.0
	// immutable-setting=immutable-value
	// mutable-once-setting=mutable-once-value
	// mutable-setting=mutable-value
}
