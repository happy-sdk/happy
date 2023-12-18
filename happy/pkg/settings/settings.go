// Copyright 2023 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

// Package settings is like a Swiss Army knife for handling app configurations in Go.
// It's designed to make your life easier when dealing with all sorts of settings -
// think of it as your go-to toolkit for configuration management.
//
// Here's the lowdown on what this package brings to the table:
//   - Marshaller and Unmarshaller: These interfaces are your friends for transforming
//     settings to and from byte slices. They're super handy for storage or network transmission.
//   - String, Bool, Int, Uint etc.: Meet the fundamental building blocks for your settings.
//   - Blueprint: This is the brains of the operation. It lets you define and manage
//     settings schemas, validate inputs, and even extend configurations with more settings.
//   - Schema: This is the blueprint's state. It's a collection of settings that can be compiled
//     into a schema to store schema version. It also provides a way to create profiles.
//   - Profile: Think of it as your settings' memory. It keeps track of user preferences
//     and application settings, making sure everything's in place and up-to-date.
//   - Preferences: This is the profile's state. It's a collection of settings that can be
//     compiled into a profile to store user preferences.
//   - Reflective Magic: We use reflection (responsibly!) to dynamically handle fields in
//     your structs. This means less boilerplate and more action.
package settings

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unicode"

	"golang.org/x/text/language"
)

var (
	ErrSettings = errors.New("settings")
	ErrSetting  = errors.New("setting")
	ErrProfile  = fmt.Errorf("%w: profile", ErrSettings)
	ErrSpec     = errors.New("spec error")
)

type Marshaller interface {
	MarshalSetting() ([]byte, error)
}

type Unmarshaller interface {
	UnmarshalSetting([]byte) error
}

type SettingField interface {
	fmt.Stringer
	Marshaller
	SettingKind() Kind
}

type Settings interface {
	Blueprint() (*Blueprint, error)
}

func toDotSeparated(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			result = append(result, '.')
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}

func ParseToSpec(s Settings) ([]SettingSpec, error) {
	return nil, nil
}

func NewBlueprint(s Settings) (*Blueprint, error) {

	// Use reflection to inspect the interface
	val := reflect.ValueOf(s)
	typ := val.Type()

	// Check if the interface is a pointer and dereference it if so
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil, errors.New("provided interface is nil")
		}
		val = val.Elem()
	} else {
		copy := reflect.New(typ)
		copy.Elem().Set(val)
		val = copy.Elem()
	}

	// After dereferencing, check if it's a struct
	if val.Kind() != reflect.Struct {
		return nil, errors.New("provided interface is not a struct or a pointer to a struct")
	}

	b := &Blueprint{
		i18n: make(map[language.Tag]map[string]string),
		mode: getExecutionMode(),
	}

	b.pkg = b.setPKG()

	// Iterate over the struct fields
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		// Skip the embedded field
		if field.Anonymous || !field.IsExported() {
			continue
		}
		value := val.Field(i)
		spec, err := b.settingSpecFromField(field, value)
		if err != nil {
			return nil, err
		}
		if err := b.AddSpec(spec); err != nil {
			return nil, err
		}
	}

	return b, nil
}

func (b *Blueprint) settingSpecFromField(field reflect.StructField, value reflect.Value) (SettingSpec, error) {

	spec := SettingSpec{}

	spec.IsSet = isFieldSet(value)
	spec.Key = field.Tag.Get("key")
	// Use struct field name converted to dot.separated.format if 'key' tag is not present
	if spec.Key == "" {
		spec.Key = toDotSeparated(field.Name)
	}

	if fieldImplementsSettings(field) {
		spec.Mutability = SettingImmutable
		spec.IsSet = true
		spec.Kind = KindSettings
		var err error
		spec.Settings, err = callBlueprintIfImplementsSettings(value)
		if err != nil {
			return spec, err
		}
	} else if fieldImplementsSetting(field) {
		spec.Required = field.Tag.Get("required") == "" || field.Tag.Get("required") == "true"

		mutation := field.Tag.Get("mutation")
		switch mutation {
		case "once":
			spec.Mutability = SettingOnce
		case "mutable":
			spec.Mutability = SettingMutable
		default:
			spec.Mutability = SettingImmutable
		}

		kindGetterMethod := value.MethodByName("SettingKind")
		if kindGetterMethod.IsValid() {
			results := kindGetterMethod.Call(nil)
			if len(results) != 1 {
				return spec, fmt.Errorf("%w: %q field %q must implement either Setting or Settings interface", ErrBlueprint, b.pkg, spec.Key)
			}
			spec.Kind = results[0].Interface().(Kind)
		} else {
			spec.Kind = KindCustom
		}

		spec.Default = field.Tag.Get("default")

		if spec.IsSet {
			spec.Value = getStringValue(value)
		} else {
			spec.Value = spec.Default
		}

		if value.CanInterface() {
			if unmarshaller, ok := value.Addr().Interface().(Unmarshaller); ok {
				spec.Unmarchaler = unmarshaller
			}
		}
		if marshaller, ok := value.Interface().(Marshaller); ok {
			spec.Marchaler = marshaller
		}

		if spec.Unmarchaler == nil {
			return spec, fmt.Errorf("%w: %q field %q must implement either SettingField interface missing UnmarshalSetting", ErrBlueprint, b.pkg, spec.Key)
		}
		if spec.Marchaler == nil {
			return spec, fmt.Errorf("%w: %q field %q must implement either SettingField interface missing MarshalSetting", ErrBlueprint, b.pkg, spec.Key)
		}
	} else {
		return spec, fmt.Errorf("%w: %q field %q must implement either Settings or SettingField interface", ErrBlueprint, b.pkg, spec.Key)
	}

	return spec, nil
}

// Function to check if a field has been set (is not the zero value for its type)
func isFieldSet(field reflect.Value) bool {
	zero := reflect.Zero(field.Type()).Interface()
	return !reflect.DeepEqual(field.Interface(), zero)
}

// callBlueprintIfImplementsSettings calls the Blueprint method if the field implements Settings.
func callBlueprintIfImplementsSettings(fieldValue reflect.Value) (*Blueprint, error) {
	// Ensure the value is valid and not a nil pointer
	if !fieldValue.IsValid() || (fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil()) {
		return nil, errors.New("invalid or nil value provided")
	}

	// Check if the field implements the Settings interface
	settingsType := reflect.TypeOf((*Settings)(nil)).Elem()
	if fieldValue.Type().Implements(settingsType) {
		method := fieldValue.MethodByName("Blueprint")
		if method.IsValid() {
			results := method.Call(nil)

			if len(results) != 2 {
				return nil, errors.New("unexpected number of return values from Blueprint method")
			}

			// Extract and return the results
			blueprint, _ := results[0].Interface().(*Blueprint)
			errVal := results[1].Interface()
			var err error
			if errVal != nil {
				err, _ = errVal.(error)
			}

			return blueprint, err
		}
	}

	return nil, errors.New("type does not implement Settings or Blueprint method not found")
}

func fieldImplementsSettings(field reflect.StructField) bool {
	fieldType := field.Type

	// Check if the field is a pointer and get the element type if so
	if fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}

	// Get the reflect.Type representation of the Settings interface
	settingsType := reflect.TypeOf((*Settings)(nil)).Elem()

	// Check if the field's type implements the Settings interface
	return fieldType.Implements(settingsType)
}

func fieldImplementsSetting(field reflect.StructField) bool {
	fieldType := field.Type

	// Check if the field is a pointer and get the element type if so
	if fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}

	// Get the reflect.Type representation of the Setting interface
	settingType := reflect.TypeOf((*SettingField)(nil)).Elem()

	// Check if the field's type implements the Setting interface
	return fieldType.Implements(settingType)
}

func getStringValue(v reflect.Value) string {
	if !v.IsValid() {
		return "<invalid>"
	}

	// Check if the value implements fmt.Stringer
	if v.Type().Implements(reflect.TypeOf((*fmt.Stringer)(nil)).Elem()) {
		return v.Interface().(fmt.Stringer).String()
	}

	// Fallback for non-Stringer string types
	if v.Kind() == reflect.String {
		return v.String()
	}

	// Handle other types as needed
	return fmt.Sprintf("<%s value>", v.Type())
}

// executionMode determines the context in which the application is running.
type ExecutionMode int

const (
	ModeUnknown    ExecutionMode = 1 << iota // Unknown mode
	ModeProduction                           // Running as a compiled binary
	ModeDevel                                // Running with `go run`
	ModeTesting                              // Running in a test context
)

func (e ExecutionMode) String() string {
	switch e {
	case ModeProduction:
		return "production"
	case ModeDevel:
		return "devel"
	case ModeTesting:
		return "testing"
	default:
		return "unkonown"
	}
}

func (e ExecutionMode) MarshalText() ([]byte, error) {
	return []byte(e.String()), nil
}

func (e *ExecutionMode) UnmarshalText(data []byte) error {
	var m ExecutionMode
	mode := string(data)

	switch mode {
	case "production":
		m = ModeProduction
	case "devel":
		m = ModeDevel
	case "testing":
		m = ModeTesting
	default:
		m = ModeUnknown
	}
	// Implement custom logic for unmarshaling a setting.
	// Example: convert byte slice to string
	*e = m
	return nil
}

func getExecutionMode() ExecutionMode {
	exePath := os.Args[0]
	exeName := filepath.Base(exePath)

	// Check for the presence of a test flag, which is added by `go test`.
	for _, arg := range os.Args[1:] { // os.Args[0] is the executable name, so we start from os.Args[1]
		if strings.HasPrefix(arg, "-test.") {
			return ModeTesting
		}
	}

	// Heuristics for `go run`: check if the executable is in a temporary directory and has a non-standard name.
	isTempBinary := (strings.HasPrefix(exePath, os.TempDir()) && len(exeName) > 10 && strings.Count(exeName, "-") >= 2) || strings.Contains(exePath, "/go-build")

	// Additional heuristics for Windows, where temporary binaries have a ".exe" suffix.
	if strings.HasSuffix(exeName, ".exe") && strings.Contains(exePath, `\AppData\Local\Temp\`) {
		isTempBinary = true
	}

	if isTempBinary {
		return ModeDevel
	}

	return ModeProduction
}
