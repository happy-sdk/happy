// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package l10n

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/happy-sdk/happy/pkg/i18n"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/session"
	"golang.org/x/text/language"
)

// genArg is one typed parameter of a generated accessor: the argument's name
// and the Go type its i18n verb resolves to.
type genArg struct {
	Name   string
	GoType string
}

// genKey is one message a typed accessor is generated for: its full i18n key
// and its ordered typed arguments.
type genKey struct {
	Key  string
	Args []genArg
}

// goTypeForArgType maps an i18n.ArgType (derived from the message's own
// {name:verb} template - never hand-authored) to the Go parameter type a
// generated accessor should use. Anything without a precise Go type maps to
// `any`, so a template using an exotic verb still generates a compiling
// accessor rather than failing generation.
//
// ArgTypeCurrency maps to currency.Amount, not a scalar type: the runtime
// {name:currency} verb (program.go's formatInto) only ever accepts a real
// currency.Amount value - anything else (a bare string or float64) renders
// program.go's visible "%!currency(type=value)" mismatch marker, since a
// single placeholder can't carry both an amount and a currency code. Mapping
// it to "string" here would generate a typed accessor whose type promises a
// currency but can never actually produce one - see importsForArgTypes,
// which adds the matching x/text/currency import whenever this type is used.
func goTypeForArgType(t i18n.ArgType) string {
	switch t {
	case i18n.ArgTypeString, i18n.ArgTypeQuoted, i18n.ArgTypeType, i18n.ArgTypeHex:
		return "string"
	case i18n.ArgTypeDecimal:
		return "int"
	case i18n.ArgTypeFloat, i18n.ArgTypeNumber:
		return "float64"
	case i18n.ArgTypeTruth:
		return "bool"
	case i18n.ArgTypeCurrency:
		return "currency.Amount"
	default:
		return "any"
	}
}

// goKeywords are Go's reserved words - none of them are valid identifiers,
// so a translation arg literally named e.g. "type" (a real case: pkg/i18n's
// own error.unsupported_type key has an arg named "type") can't be used
// verbatim as a generated accessor's Go parameter name. goParamName escapes
// it; the arg's actual name (unescaped) still goes to i18n.T as the key.
var goKeywords = map[string]bool{
	"break": true, "default": true, "func": true, "interface": true, "select": true,
	"case": true, "defer": true, "go": true, "map": true, "struct": true,
	"chan": true, "else": true, "goto": true, "package": true, "switch": true,
	"const": true, "fallthrough": true, "if": true, "range": true, "type": true,
	"continue": true, "for": true, "import": true, "return": true, "var": true,
}

// goParamName returns name as a safe Go parameter identifier, escaping it
// (by appending "_") if it collides with a Go keyword, or if it's the blank
// identifier "_" - a legal {name:verb} placeholder name (see named.go's
// namedPlaceholderRe, which allows a leading underscore) but not a keyword,
// and not usable as a value ("_" can't appear on the right-hand side of an
// expression, so `i18n.T(key, "_", _)` fails to compile). The translation
// key/arg name itself is never altered - only the local Go variable that
// holds its value in a generated accessor.
func goParamName(name string) string {
	if name == "_" || goKeywords[name] {
		return name + "_"
	}
	return name
}

// exportedFuncName turns a full translation key into an exported Go identifier
// by CamelCasing the portion of the key below its bundle. E.g. bundle
// "com.github.example.app" + key "com.github.example.app.help.description" ->
// "HelpDescription".
func exportedFuncName(bundle, key string) string {
	rel := strings.TrimPrefix(key, bundle+".")
	rel = strings.TrimPrefix(rel, bundle) // key == bundle edge case
	parts := strings.FieldsFunc(rel, func(r rune) bool {
		return r == '.' || r == '_' || r == '-'
	})
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]))
		b.WriteString(p[1:])
	}
	if b.Len() == 0 {
		return "Message"
	}
	name := b.String()
	// A Go identifier can't start with a digit.
	if name[0] >= '0' && name[0] <= '9' {
		name = "M" + name
	}
	return name
}

// generateTypedAccessors emits compile-time-checked Go source for a bundle: one
// exported function per message key, each parameter typed from the message's
// own derived ArgTypes, each body delegating to the dynamic i18n.T API. The
// output is gofmt-formatted. keys are sorted by key for deterministic output.
func generateTypedAccessors(pkgName, bundle string, keys []genKey) ([]byte, error) {
	if pkgName == "" {
		return nil, fmt.Errorf("package name must not be empty")
	}
	sorted := make([]genKey, len(keys))
	copy(sorted, keys)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Key < sorted[j].Key })

	// Two distinct keys can collapse to the same exported name - e.g.
	// "foo.bar_baz" and "foo.bar.baz" both CamelCase to "FooBarBaz", since
	// exportedFuncName splits on ".", "_", and "-" alike. Undetected, that's
	// a "FooBarBaz redeclared" compile error in the generated file with no
	// indication of which two keys are at fault. Check up front instead.
	seenFn := make(map[string]string, len(sorted))
	for _, k := range sorted {
		fn := exportedFuncName(bundle, k.Key)
		if other, ok := seenFn[fn]; ok && other != k.Key {
			return nil, fmt.Errorf("keys %q and %q both generate the function name %q - rename one to avoid a naming collision", other, k.Key, fn)
		}
		seenFn[fn] = k.Key
	}

	// golang.org/x/text/currency is only needed when at least one generated
	// parameter is a currency.Amount (see goTypeForArgType).
	needsCurrency := false
	for _, k := range sorted {
		for _, a := range k.Args {
			if a.GoType == "currency.Amount" {
				needsCurrency = true
			}
		}
	}

	var buf bytes.Buffer
	buf.WriteString("// Code generated by \"l10n generate\"; DO NOT EDIT.\n\n")
	fmt.Fprintf(&buf, "package %s\n\n", pkgName)
	if needsCurrency {
		buf.WriteString("import (\n\t\"github.com/happy-sdk/happy/pkg/i18n\"\n\t\"golang.org/x/text/currency\"\n)\n\n")
	} else {
		buf.WriteString("import \"github.com/happy-sdk/happy/pkg/i18n\"\n\n")
	}

	for _, k := range sorted {
		fn := exportedFuncName(bundle, k.Key)
		// Parameters.
		params := make([]string, 0, len(k.Args))
		for _, a := range k.Args {
			params = append(params, fmt.Sprintf("%s %s", goParamName(a.Name), a.GoType))
		}
		fmt.Fprintf(&buf, "// %s returns the translation for %q in the current language.\n", fn, k.Key)
		fmt.Fprintf(&buf, "func %s(%s) string {\n", fn, strings.Join(params, ", "))
		if len(k.Args) == 0 {
			fmt.Fprintf(&buf, "\treturn i18n.T(%q)\n", k.Key)
		} else {
			callArgs := make([]string, 0, len(k.Args))
			for _, a := range k.Args {
				callArgs = append(callArgs, fmt.Sprintf("%q, %s", a.Name, goParamName(a.Name)))
			}
			fmt.Fprintf(&buf, "\treturn i18n.T(%q, %s)\n", k.Key, strings.Join(callArgs, ", "))
		}
		buf.WriteString("}\n\n")
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("generated source did not gofmt: %w\n%s", err, buf.String())
	}
	return formatted, nil
}

// collectBundleKeys gathers every message key belonging to bundle from the
// registered translations, resolving each one's typed argument list from the
// bundle's source locale (so the generated parameters match the language the
// bundle is authored in). Args are ordered by name for deterministic output.
func collectBundleKeys(bundle string, sourceLang language.Tag) []genKey {
	var keys []genKey
	for _, entry := range i18n.GetAllTranslations() {
		if entry.Key != bundle && !strings.HasPrefix(entry.Key, bundle+".") {
			continue
		}
		types, ok := i18n.GetMessageArgTypes(sourceLang, entry.Key)
		if !ok {
			// Not registered for the source locale; skip - a bundle key
			// should always exist in its own source language.
			continue
		}
		names := make([]string, 0, len(types))
		for name := range types {
			names = append(names, name)
		}
		sort.Strings(names)
		args := make([]genArg, 0, len(names))
		for _, name := range names {
			args = append(args, genArg{Name: name, GoType: goTypeForArgType(types[name])})
		}
		keys = append(keys, genKey{Key: entry.Key, Args: args})
	}
	return keys
}

func l10nGenerate() *command.Command {
	cmd := command.New("generate",
		command.Config{
			Description: settings.String(l10np + ".generate.description"),
			Immediate:   true,
		})

	cmd.WithFlags(
		varflag.StringFunc("bundle", "", l10np+".generate.flag_bundle", "b"),
		varflag.StringFunc("out", "", l10np+".generate.flag_out", "o"),
		varflag.StringFunc("package", "messages", l10np+".generate.flag_package", "p"),
	)

	cmd.Do(func(sess *session.Context, args action.Args) error {
		bundle := args.Flag("bundle").String()
		out := args.Flag("out").String()
		pkgName := args.Flag("package").String()

		if bundle == "" {
			// Default to the application's own module bundle.
			prefix, err := getAppModulePrefix(sess)
			if err != nil {
				return fmt.Errorf("no --bundle given and could not derive the app's module bundle: %w", err)
			}
			bundle = prefix
		}

		// The bundle's source locale drives which language's arg types are used.
		sourceLang, ok := i18n.GetBundleSourceLanguage(bundle)
		if !ok {
			// Fall back to the global fallback language for a v1/unversioned
			// bundle that never declared a source.
			sourceLang = i18n.GetFallbackLanguage()
		}

		keys := collectBundleKeys(bundle, sourceLang)
		if len(keys) == 0 {
			return fmt.Errorf("no translation keys found for bundle %q", bundle)
		}

		src, err := generateTypedAccessors(pkgName, bundle, keys)
		if err != nil {
			return err
		}

		if out == "" {
			fmt.Println(string(src))
			return nil
		}
		if dir := filepath.Dir(out); dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
		}
		if err := os.WriteFile(out, src, 0o644); err != nil {
			return fmt.Errorf("failed to write generated file: %w", err)
		}
		fmt.Printf("Generated %d typed accessors for bundle %q -> %s\n", len(keys), bundle, out)
		return nil
	})

	return cmd
}
