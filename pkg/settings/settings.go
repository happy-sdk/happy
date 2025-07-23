// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

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
)

var (
	Error          = errors.New("settings")
	ErrSetting     = errors.New("setting")
	ErrProfile     = fmt.Errorf("%w: profile", Error)
	ErrPreferences = fmt.Errorf("%w: preferences", Error)
	ErrSpec        = errors.New("spec error")
)

// Marshaller interface for marshaling settings
type Marshaller interface {
	MarshalSetting() ([]byte, error)
}

// Unmarshaller interface for unmarshaling settings
type Unmarshaller interface {
	UnmarshalSetting([]byte) error
}

// Value interface that combines multiple interfaces
type Value interface {
	fmt.Stringer
	Marshaller
	Unmarshaller
}

// SettingKind() Kind
type Settings interface {
	Blueprint() (*Blueprint, error)
}

func New[S Settings](s S) (*Blueprint, error) {
	// Use reflection to inspect the interface
	val := reflect.ValueOf(s)
	typ := val.Type()
	var isPointer bool

	// Check if the provided interface is a pointer
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil, errors.New("provided interface is nil")
		}
		isPointer = true
		val = val.Elem()
		typ = typ.Elem() // Update the type to the dereferenced type
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
		mode: getExecutionMode(),
	}

	b.pkgSettingsStructName = setPKG(b.mode)

	// Iterate over the struct fields
	for i := range val.NumField() {
		field := typ.Field(i)
		// Skip the embedded field
		if field.Anonymous || !field.IsExported() {
			continue
		}
		value := val.Field(i)
		spec, err := settingSpecFromField(field, value)
		if err != nil {
			if isPointer {
				return nil, fmt.Errorf("%T: %w", s, err)
			}
			return nil, err
		}

		// handle short syntax group keys
		// Update the field value if needed
		if isPointer {

			// Get the field by name from the original value to set it back
			originalField := reflect.ValueOf(s).Elem().FieldByName(field.Name)
			if originalField.CanSet() {
				// Assuming settingSpecFromField returns the updated value we want to set
				// Use type assertion to ensure compatibility with the original field type
				if setter, ok := originalField.Addr().Interface().(Value); ok {
					if err := setter.UnmarshalSetting([]byte(spec.Value)); err != nil {
						return nil, fmt.Errorf("failed to set field %s: %w", field.Name, err)
					}
				} else if nested, ok := originalField.Addr().Interface().(Settings); ok {
					// Handle nested settings
					if spec.Settings, err = nested.Blueprint(); err != nil {
						return nil, fmt.Errorf("failed to set nested settings for field %s: %w", field.Name, err)
					}
				} else {
					return nil, fmt.Errorf("field %s does not implement settings.Value interface", field.Name)
				}
			}
		}

		if err := b.AddSpec(spec); err != nil {
			return nil, err
		}
	}

	return b, nil
}

func toUndersCoreSeparated(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			result = append(result, '_')
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}

// Function to check if a field has been set (is not the zero value for its type)
func isFieldSet(field reflect.Value) bool {
	if !field.IsValid() {
		return false
	}

	// For other types, use the zero value comparison
	zero := reflect.Zero(field.Type()).Interface()
	return !reflect.DeepEqual(field.Interface(), zero)
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
	return fieldType.Implements(settingsType) || reflect.PointerTo(fieldType).Implements(settingsType)
}

// fieldImplementsSetting checks if a field implements the Value interface
func fieldImplementsSetting(field reflect.StructField) bool {
	fieldType := field.Type

	// Check if the field is a pointer and get the element type if so
	if fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}

	// Get the reflect.Type representation of the Value interface
	settingType := reflect.TypeOf((*Value)(nil)).Elem()

	// Check if the field's type or pointer to field's type implements the Setting interface
	implements := fieldType.Implements(settingType) || reflect.PointerTo(fieldType).Implements(settingType)
	return implements
}

func getStringValue(v reflect.Value) string {
	if !v.IsValid() {
		return "<invalid>"
	}

	// Check if the value directly implements fmt.Stringer
	stringerType := reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
	if v.Type().Implements(stringerType) {
		return v.Interface().(fmt.Stringer).String()
	}

	// Check if the pointer to the value implements fmt.Stringer, if it's addressable
	if v.CanAddr() && v.Addr().Type().Implements(stringerType) {
		return v.Addr().Interface().(fmt.Stringer).String()
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
	for _, arg := range os.Args[1:] {
		// os.Args[0] is the executable name, so we start from os.Args[1]
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
