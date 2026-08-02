// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

// Package i18n provides translation, pluralization, and locale-aware
// message formatting: a compiled rendering engine for named
// {name}/{name:verb} templates and for richer messages whose wording
// varies by number or category, an immutable snapshot published for a
// lock-free render path, and a small set of conventions for shipping a
// package's own translations as an embedded bundle.
//
// # Getting started
//
// A package ships its translations as JSON files embedded with Go's embed
// package, registered from an init() function via Embed or MustEmbed (see
// "Choosing an Embed variant" below):
//
//	//go:embed locales/*
//	var locales embed.FS
//
//	func init() { i18n.MustEmbed(locales) }
//
// The embedding application calls Initialize once, with its fallback
// language, before rendering anything - typically as one of its first
// boot steps:
//
//	for _, issue := range i18n.Initialize(language.English) {
//		if issue.Fatal {
//			log.Fatal(issue.Err)
//		}
//		log.Println("i18n:", issue.Err)
//	}
//
// From there, T and TL render a registered key, interpolating named
// arguments:
//
//	i18n.T("com.example.app.greeting", "name", "World")
//	i18n.TL(language.German, "com.example.app.greeting", "name", "World")
//
// # Choosing an Embed variant
//
// Embed, EmbedIssues, and MustEmbed all register the same kind of bundle;
// they differ only in how a broken bundle is reported, which is a policy
// choice tied to what the registering package itself depends on:
//
//	MustEmbed  panics on a broken bundle. Use it for a package with no
//	           third-party dependencies - happy itself, everything under
//	           sdk/, everything under pkg/. Such a module is never released
//	           with a broken bundle, so a failure at runtime means something
//	           is badly wrong and the application should fail to start, the
//	           same as any other boot/init error.
//	Embed      records a broken bundle for pickup by the next Initialize or
//	           Reload call and otherwise reports nothing itself. Use it for
//	           a package that may carry third-party dependencies (everything
//	           under lib/ or addons/) and is only ever used inside a
//	           happy-sdk application, which is guaranteed to call one of
//	           those.
//	EmbedIssues does the same recording, but also returns the failure
//	           directly. Use it instead of Embed for a package that might
//	           also be used standalone, in a plain Go application that
//	           never calls Initialize or Reload.
//
// # Keys, templates, and messages
//
// A translation is registered under a string key - by convention a
// reverse-DNS path rooted in the registering package's own import path,
// e.g. "com.github.happy-sdk.happy.pkg.i18n.error.disabled" (see
// GetPackagePrefix) - whose value is either a plain template string or a
// *Message for wording that varies by number or category.
//
// A template interpolates named, not positional, arguments - {name} or
// {name:verb} (e.g. {name:s}, {count:d}, {amount:currency}) - resolved
// against Args passed as bare "key", value pairs (mirroring log/slog's
// convention) or as typed helpers (String, Int, Float64, ...):
//
//	i18n.T("com.example.app.greeting", "name", "World")
//	i18n.T("com.example.app.greeting", i18n.String("name", "World"))
//
// A *Message adds one or more named fragments - built with WithFragment
// (CLDR cardinal plural), WithOrdinalFragment (CLDR ordinal), or
// WithSelectFragment (exact match, e.g. by gender) - each contributing one
// wording per resolved category into the surrounding template. See
// Message for the full "$message" JSON shape a translation file uses for
// the same thing.
//
// # Registering translations directly
//
// RegisterTranslation and RegisterTranslations register one key, or a
// whole (possibly nested) tree of keys, applying immediately.
// QueueTranslation and QueueTranslations do the same but defer applying
// until the next Reload - useful when registering many translations up
// front without rebuilding the render snapshot after each one.
//
// # Reporting, not logging
//
// This package reports, it doesn't log: every problem it encounters -
// loading a bundle, applying a queued translation - surfaces as an
// []InitIssue (see Initialize, Reload, Embed, EmbedIssues) rather than
// being written to slog directly, since only the caller actually knows
// what a given failure should mean (a fatal boot error, a warning, or
// something else entirely) and how to surface it through its own logging.
// The one exception is SetLanguage's own slog.Debug on a successful
// language change, which is informational, not an error/warning being
// reported through InitIssue at all.
package i18n

import (
	"context"
	"embed"
	"encoding/json/v2"
	"errors"
	"fmt"
	"sync"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

//go:embed locales/*
var locales embed.FS

type contextKey string

const (
	langContextKey contextKey = "language"
)

// packagePrefix is this package's own reverse-DNS translation key root -
// the same one its bundled locales/*.json files use. Its own sentinel
// errors below are keyed under it directly rather than through
// composePackageKey, since that derives the prefix from the caller's stack
// frame and these are constructed at package-init time, before any
// meaningful caller exists.
const packagePrefix = "com.github.happy-sdk.happy.pkg.i18n"

// enFallbacks holds this package's own bundled English error messages,
// read directly from locales/en.json at package-init time. It exists so
// newPackageError never needs a fallback string passed in: en.json is the
// single source of truth for these messages' English text, not duplicated
// as a second hardcoded copy in Go source.
var enFallbacks = loadEnFallbacks()

func loadEnFallbacks() map[string]string {
	data, err := locales.ReadFile("locales/en.json")
	if err != nil {
		return nil
	}
	var doc map[string]struct {
		Error map[string]string `json:"error"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil
	}
	return doc[packagePrefix].Error
}

// newPackageError builds one of this package's own sentinel errors:
// localized through i18n's own translation API and always satisfying
// errors.Is(err, Error), so callers can detect "this failed inside i18n"
// without caring which specific sentinel it was. Its fallback text comes
// from enFallbacks (locales/en.json), not a parameter - see enFallbacks.
//
// It also does not QueueTranslation(language.Und, ...) its fallback the
// way NewError/NewErrorWithLocale/NewErrorDepth do: this package's own
// locales/*.json already carries a proper translation for every key here in
// every supported language, loaded by RegisterTranslationsFS inside
// initialize(). Queuing the English fallback under language.Und would
// have reload() flush it onto whatever fallbackLang actually is - which is
// only English if the embedding application chose English as its fallback
// - overwriting that language's correct, just-loaded translation with
// English text. The fallback field alone still covers every case that
// needs it: Error() reads it directly pre-Initialize, and post-Initialize
// as the last resort if a lookup ever misses.
func newPackageError(key string) *LocalizedError {
	return &LocalizedError{
		key:      packagePrefix + ".error." + key,
		fallback: enFallbacks[key],
		tag:      language.Und,
		wraps:    Error,
	}
}

// Error is the base sentinel every error this package returns satisfies -
// check with errors.Is(err, i18n.Error) to detect "this failed inside
// i18n" without caring which specific condition caused it.
var Error = errors.New("i18n")

// ErrDisabled is for callers (e.g. happy.I18nSettings' language validator)
// to test with errors.Is when a build or profile intentionally never
// initializes the i18n manager - that absence is not itself a failure.
var ErrDisabled = newPackageError("disabled")

// ErrLanguageNotSupported is returned when a requested language.Tag hasn't
// been registered as supported by any RegisterTranslation* call. Carries
// no args on its own - callers get a filled-in copy via
// ErrLanguageNotSupported.WithArgs("lang", lang.String()).
var ErrLanguageNotSupported = newPackageError("language_not_supported")

// ErrUnsupportedType is returned when a translation value passed to
// RegisterTranslation/RegisterTranslations is not a string,
// catalog.Message, []catalog.Message, or a nested map of those.
var ErrUnsupportedType = newPackageError("unsupported_type")

// ErrReadDir is returned when a translations *FS's root or a language
// subdirectory within it can't be read.
var ErrReadDir = newPackageError("read_dir")

// ErrUnexpectedDirectory is returned when a translations *FS finds a
// directory where it expected a translation file.
var ErrUnexpectedDirectory = newPackageError("unexpected_directory")

// ErrReadFile is returned when a translation file within a translations
// *FS can't be read.
var ErrReadFile = newPackageError("read_file")

// ErrParseFile is returned when a translation file's content isn't valid
// JSON.
var ErrParseFile = newPackageError("parse_file")

// ErrParseLanguageTag is returned when a translations *FS can't parse a
// BCP 47 language tag from a file or directory name.
var ErrParseLanguageTag = newPackageError("parse_language_tag")

// ErrInvalidMessage is returned when a JSON object translation value looks
// like a Message descriptor (it has a "msg" field) but is otherwise
// malformed - e.g. a plural fragment missing its "count" field or its
// required "other" form. Carries no args on its own - callers get a
// filled-in copy via ErrInvalidMessage.WithArgs("name", ..., "reason", ...).
var ErrInvalidMessage = newPackageError("invalid_message")

// ErrInvalidSchemaVersion is returned when a translation file's reserved
// "$version" key (see package schema) can't be resolved: it's above the
// schema version this build of pkg/i18n knows how to read, isn't a
// recognizable integer, or a registered migration up to the current
// version failed. Carries no args on its own - callers get a filled-in
// copy via ErrInvalidSchemaVersion.WithArgs("lang", ..., "cause", ...).
var ErrInvalidSchemaVersion = newPackageError("invalid_schema_version")

// ErrUnknownLocaleKey is returned (as a non-Fatal InitIssue, never
// panicked or returned directly from registration - see reload) when a
// bundle's non-source locale registers a key that doesn't exist anywhere
// in that bundle's declared source locale (see schema/v2.KeySource).
// Schema validation alone can't catch this: it only checks that a key's
// *shape* is well-formed (see schema/v2's own key-naming rule), not
// whether it corresponds to anything real - a translator's typo in a key
// name (as opposed to a value) silently creates a new, orphaned key that
// nothing ever looks up, while the correctly-named key it was meant to be
// stays entirely missing from that locale. Carries no args on its own -
// callers get a filled-in copy via ErrUnknownLocaleKey.WithArgs("bundle",
// ..., "lang", ..., "key", ..., "source", ..., "has_suggestion",
// "yes"/"no", "suggestion", ...).
var ErrUnknownLocaleKey = newPackageError("unknown_locale_key")

// TranslationEntry represents a single translation entry with its key,
// fallback value, and translations for all supported languages.
type TranslationEntry struct {
	// Key is the translation key
	Key string
	// RootKey is the root translation key (package identifier) this entry belongs to.
	// For example, "com.github.happy-sdk.happy.sdk.cli" or "com.github.happy-sdk.happy.pkg.vars.varflag"
	RootKey string
	// Fallback is the translation value in the fallback language
	Fallback string
	// Translations is a map of language tags to their translation values.
	// If a language doesn't have a translation, it won't be present in the map.
	Translations map[language.Tag]string
}

// TranslationReport provides a report for a specific language's translation status.
type TranslationReport struct {
	// Language is the language tag this report is for
	Language language.Tag
	// Total is the total number of translation keys
	Total int
	// Translated is the number of keys that have translations for this language
	Translated int
	// Missing is the number of keys that are missing translations for this language
	Missing int
	// Percentage is the percentage of keys that are translated (0.0 to 100.0)
	Percentage float64
	// MissingEntries contains all translation entries that are missing translations for this language
	MissingEntries []TranslationEntry
	// RootKeys is the list of all root translation keys (package identifiers) found in translations
	RootKeys []string
	// PerRootKey is a map of root key to its translation statistics for this language
	PerRootKey map[string]RootKeyStats
}

// RootKeyStats provides translation statistics for a specific root key.
type RootKeyStats struct {
	// RootKey is the root translation key
	RootKey string
	// Total is the total number of translation keys for this root
	Total int
	// Translated is the number of keys that have translations for this language
	Translated int
	// Missing is the number of keys that are missing translations for this language
	Missing int
	// Percentage is the percentage of keys that are translated (0.0 to 100.0)
	Percentage float64
}

// InitIssue reports a problem encountered while loading a bundle or
// applying translations queued via QueueTranslation/QueueTranslations -
// via Initialize, Reload, Embed (from an init() that ran before either was
// ever called), or EmbedIssues. Fatal is true only for this package's own
// bundle failing to load (see Initialize) - serious enough that a caller
// building a stricter boot sequence (see happy-sdk's own initializer,
// which maps this to its own boot-failure handling) should treat it as an
// application boot failure, not merely log a warning and continue - the
// same severity MustEmbed's panic would otherwise signal, just captured
// here instead of thrown. Every other issue (a queue-flush problem, or an
// Embed/EmbedIssues failure) is never Fatal: each reflects one bundle or
// key among potentially many, and this package's own pre-existing "keep
// going" behavior for it is unchanged. Never independently logged here
// either way - this package reports, it doesn't log (see the package doc
// comment's note on slog usage) - a caller decides what, if anything, to
// do with each issue.
type InitIssue struct {
	Err   error
	Fatal bool
}

// pendingIssues holds every issue Embed/EmbedIssues has recorded that
// nothing has drained yet. Package init() functions - which is where Embed
// is meant to be called - all run before main(), and therefore before an
// application ever gets a chance to call Initialize/Reload; this is what
// lets a failure recorded during that init() phase still reach whichever
// of those is called later, automatically, with no cooperation required
// from the package that called Embed.
var (
	pendingIssuesMu sync.Mutex
	pendingIssues   []InitIssue
)

func recordPendingIssue(issue InitIssue) {
	pendingIssuesMu.Lock()
	defer pendingIssuesMu.Unlock()
	pendingIssues = append(pendingIssues, issue)
}

// drainPendingIssues removes and returns every issue recorded so far. Only
// Initialize and the exported Reload call this - never the internal
// reload() shared with defer reload() sites inside registerTranslation(s),
// whose return value is routinely discarded: draining is destructive
// (empties the pool), so doing it from a callsite that then throws the
// result away would silently and permanently lose whatever was pending.
func drainPendingIssues() []InitIssue {
	pendingIssuesMu.Lock()
	defer pendingIssuesMu.Unlock()
	issues := pendingIssues
	pendingIssues = nil
	return issues
}

// Initialize sets the fallback language and loads this package's own
// bundle plus any translations queued (via QueueTranslation/QueueTranslations)
// before this call - the bundle/queue-flush part is a no-op on every call
// after the first. Every call, including later ones,
// also drains and returns any issue Embed/EmbedIssues has recorded since
// the last drain (see pendingIssues) - so a bundle embedded via Embed from
// some other package's init(), before this was ever called, still
// surfaces here automatically. Issues are never panicked or silently
// dropped - see InitIssue - so a caller with its own boot-failure handling
// (see happy-sdk's own initializer) can react to a Fatal one exactly like
// any other boot/init error, instead of a raw panic surfacing from deep
// inside this package.
func Initialize(fallback language.Tag) []InitIssue {
	issues := initialize(fallback)
	return append(issues, drainPendingIssues()...)
}

// RegisterTranslationsFS registers every translation file fs's root (or
// language subdirectory) holds. It is the lower-level primitive Embed,
// EmbedIssues, and MustEmbed are all built on; call it directly only if a
// bundle needs an FS built with WithPrefix (a layout other than the
// "locales" subdirectory convention those three assume).
func RegisterTranslationsFS(fs *FS) error {
	return registerTranslationsFS(fs)
}

// Embed registers a self-describing (schema v2) bundle embedded under a
// "locales" subdirectory alongside its package's own Go source - the one
// convention every bundle in this monorepo uses. A package named "l10n"
// (e.g. sdk/app/l10n, addons/l10n) holds Go loader/tooling code only; its
// actual translation content lives in a nested "locales" subdirectory, not
// a second "l10n" (a "l10n/l10n" path was the awkward alternative this
// convention replaced):
//
//	//go:embed locales/*
//	var bundle embed.FS
//	func init() { i18n.Embed(bundle) }
//
// There is deliberately no way to point Embed at a different layout: one
// convention, not a configurable one, is the whole point - a caller with a
// genuinely different embed layout can still reach for the lower-level
// RegisterTranslationsFS(NewFS(fsys).WithPrefix(...)) primitives Embed
// itself is built on (NewFS's own default prefix is already "locales",
// which is why Embed needs nothing more than NewFS(fsys) below).
//
// Embed deliberately returns nothing and never logs anything itself - not
// even via slog: this package reports, it doesn't log (see the package
// doc comment). A failed registration is recorded (see pendingIssues) so
// it's automatically included the next time Initialize or Reload is
// called - the intended contract for a package meant only ever to be used
// inside a happy-sdk application, which is guaranteed to call one of
// those. A package that might instead (or also) be used standalone in a
// plain Go application that never calls either should use EmbedIssues,
// which reports the same failure directly instead of only through that
// later pickup.
//
// Embed vs MustEmbed is a policy choice, not just a convenience one: use
// MustEmbed for a package with no third-party dependencies (happy itself,
// everything under sdk/, everything under pkg/) - such a module is never
// released unless every bundle it ships is fully valid, so a broken one at
// runtime means something is badly wrong and the application should fail
// to start, the same as any other boot/init error. Use plain Embed (or
// EmbedIssues) for a package that may carry third-party dependencies
// (everything under lib/ or addons/) - report and let the embedding
// application decide whether to continue or exit cleanly with its own
// error, rather than a dependency's addon unilaterally panicking a host
// application it doesn't control.
func Embed(fsys embed.FS) {
	if err := RegisterTranslationsFS(NewFS(fsys)); err != nil {
		recordPendingIssue(InitIssue{Err: err})
	}
}

// EmbedIssues is Embed for a package that might be used standalone, in a
// plain Go application that never calls Initialize or Reload - it reports
// the same failure Embed would, directly, as the returned slice (empty on
// success), so such a caller can decide for itself how (or whether) to log
// it. It still also records the issue for automatic pickup (see Embed),
// so nothing changes for a caller that happens to run inside a happy-sdk
// application instead - both audiences see the same failure, through
// whichever mechanism actually applies to them.
func EmbedIssues(fsys embed.FS) []InitIssue {
	if err := RegisterTranslationsFS(NewFS(fsys)); err != nil {
		issue := InitIssue{Err: err}
		recordPendingIssue(issue)
		return []InitIssue{issue}
	}
	return nil
}

// MustEmbed is Embed for callers that want a broken bundle to fail loudly
// (panic) at process start instead of being reported - see Embed's doc
// comment for when to use which:
//
//	func init() { i18n.MustEmbed(bundle) }
func MustEmbed(fsys embed.FS) {
	if err := RegisterTranslationsFS(NewFS(fsys)); err != nil {
		panic(err)
	}
}

// RegisterTranslations registers a whole tree of translations for lang in
// one call, applying immediately (see QueueTranslations to defer applying
// until a later Reload). translations may be schema version 2's
// self-describing document shape (a "$version" key, in which case lang is
// ignored in favor of whatever locale(s) the document itself declares) or
// a plain, possibly-nested map of keys to string/*Message values.
func RegisterTranslations(lang language.Tag, translations map[string]any) error {
	return registerTranslations(lang, translations)
}

// RegisterTranslation registers a single key's value for lang, applying
// immediately (see QueueTranslation to defer applying until a later
// Reload). value must be a string, a *Message, or an
// x/text/message/catalog.Message/[]catalog.Message; anything else returns
// ErrUnsupportedType.
func RegisterTranslation(lang language.Tag, key string, value any) error {
	return registerTranslation(lang, "", key, value)
}

// SupportsLanguage reports whether lang has at least one registered
// translation - i.e. whether some earlier RegisterTranslation(s)/
// QueueTranslation(s)/Embed call has registered something for it.
func SupportsLanguage(lang language.Tag) bool {
	return supportsLanguage(lang)
}

// QueueTranslation registers key's value for lang exactly like
// RegisterTranslation, except it doesn't apply immediately: the
// translation sits queued until the next Reload (or the next
// RegisterTranslation(s) call, which reloads as a side effect). Use it to
// register many translations up front - e.g. while parsing several files
// - without rebuilding the render snapshot after every single one.
func QueueTranslation(lang language.Tag, key string, value any) error {
	return queueTranslation(lang, key, value)
}

// QueueTranslations is QueueTranslation for a whole tree of translations
// at once, exactly like RegisterTranslations except deferred until the
// next Reload.
func QueueTranslations(lang language.Tag, translations map[string]any) error {
	return queueTranslations(lang, translations)
}

// SetLanguage sets the current language T (and PT, PTD, TD) render
// against. It returns ErrLanguageNotSupported if lang has no registered
// translation (see SupportsLanguage), leaving the current language
// unchanged.
func SetLanguage(lang language.Tag) error {
	return setLanguage(lang)
}

// GetLanguage returns the current language (see SetLanguage), the one T
// (and PT, PTD, TD) render against. It is language.English until
// Initialize or SetLanguage first runs.
func GetLanguage() language.Tag {
	return getLanguage()
}

// GetFallbackLanguage returns the fallback language passed to Initialize -
// the last language T/TL fall back to when a key has no wording in any
// more specific locale (see the package doc comment's rendering cascade).
func GetFallbackLanguage() language.Tag {
	return getFallbackLanguage()
}

// GetBundleSourceLanguage reports bundle's declared source locale (see
// pkg/i18n/schema/v2's KeySource) - the language its translations are
// authored/maintained in, and every other locale is translated from. ok is
// false for a bundle that either predates schema versioning (a legacy,
// unversioned v1 bundle - see pkg/i18n/schema/v1) or hasn't been
// registered at all; callers should not assume schema/v2.DefaultSource in
// that case.
func GetBundleSourceLanguage(bundle string) (lang language.Tag, ok bool) {
	return mngr.getBundleSource(bundle)
}

// GetLanguages returns every language with at least one registered
// translation (see SupportsLanguage).
func GetLanguages() []language.Tag {
	return getLanguages()
}

// GetMessageArgTypes reports the derived ArgType of every argument the
// translation registered for (lang, key) interpolates - keyed by argument name.
// It is the same derivation Message.ArgTypes() performs (a fragment's own
// discriminating arg is only reported if it is also interpolated literally),
// exposed for a registered key whether its value is a rich Message or a plain
// {name:verb} string. ok is false when nothing is registered for (lang, key).
//
// This is additive tooling API (e.g. addons/l10n's typed-accessor generator);
// it has no effect on and is never consulted by the T/TL render path.
func GetMessageArgTypes(lang language.Tag, key string) (types map[string]ArgType, ok bool) {
	return mngr.messageArgTypes(lang, key)
}

// GetMessage returns the rich *Message registered for (lang, key), for
// translation tooling (e.g. addons/l10n) that wants schema-level detail -
// fragments and their forms, the translator-facing description, and each
// documented arg's own description - beyond what GetMessageArgTypes
// derives. ok is false when nothing is registered for (lang, key), or when
// what's registered there is a plain string or legacy x/text
// catalog.Message rather than a genuine *Message: callers that only need
// arg types regardless of which of those a key actually is should use
// GetMessageArgTypes instead, which handles all three uniformly. Never
// consulted by the T/TL render path.
func GetMessage(lang language.Tag, key string) (msg *Message, ok bool) {
	return mngr.message(lang, key)
}

// GetKeyBundle reports the schema version 2 bundle key belongs to - the
// same authoritative ownership T/TL's own fallback cascade uses internally
// (see the package doc comment's rendering cascade), not a heuristic
// derived from the key's own shape. Pair it with GetBundleSourceLanguage
// or GetBundleNote, which both take a bundle rather than a key. ok is
// false for a legacy (schema version 1) or unregistered key, which has no
// owning bundle at all.
func GetKeyBundle(key string) (bundle string, ok bool) {
	return mngr.keyBundle(key)
}

// GetBundleNote returns bundle's translator note for lang (see
// pkg/i18n/schema/v2's KeyLocaleNotes) - free-text guidance left in that
// locale's own translation file, e.g. context a literal string alone
// wouldn't convey. ok is false if bundle has no note for lang at all -
// most bundle/locale pairs won't, since notes are optional.
func GetBundleNote(bundle string, lang language.Tag) (note string, ok bool) {
	return mngr.bundleNote(bundle, lang)
}

// ParseLanguage parses langStr as a BCP 47 language tag and matches it
// against the supported languages (see GetLanguages), returning the
// closest supported match. It never errors: an empty or unparseable
// langStr, or one that matches no supported language, returns the fallback
// language (see GetFallbackLanguage) instead.
func ParseLanguage(langStr string) language.Tag {
	return parseLanguage(langStr)
}

// WithLanguage returns a copy of ctx carrying lang, for a request-scoped
// language distinct from the process-wide current one (see SetLanguage) -
// e.g. one determined per-request by I18nMiddleware. Retrieve it with
// LanguageFromContext.
func WithLanguage(ctx context.Context, lang language.Tag) context.Context {
	return context.WithValue(ctx, langContextKey, lang)
}

// LanguageFromContext returns the language ctx carries (see WithLanguage,
// I18nMiddleware), and whether it carried one at all. A handler wanting to
// render in a request's language typically does:
//
//	lang, ok := i18n.LanguageFromContext(ctx)
//	if !ok {
//		lang = i18n.GetFallbackLanguage()
//	}
//	i18n.TL(lang, "com.example.app.greeting", "name", name)
func LanguageFromContext(ctx context.Context) (lang language.Tag, ok bool) {
	lang, ok = ctx.Value(langContextKey).(language.Tag)
	return lang, ok
}

// GetPrinter returns a message printer for the given language
// or error if language is not supported with printer with default language.
//
// The returned printer renders through x/text's own catalog, a separate
// store from the compiled snapshot T/TL use - so unlike T/TL, it has no way
// to know about a key pruned by pruneUnknownLocaleKeysLocked (x/text's
// catalog.Builder exposes no delete API). In practice this only matters if
// something calls the printer directly with the literal bad key string,
// which nothing in this codebase does.
func GetPrinter() (p *message.Printer) {
	return getPrinter()
}

// GetPrinterFor returns a message printer for the given language
// or error if language is not supported with printer with default language.
// See GetPrinter for a note on its relationship to pruned locale keys.
func GetPrinterFor(lang language.Tag) (p *message.Printer, err error) {
	return getPrinterFor(lang)
}

// GetFallbackPrinter returns a message printer for the global fallback
// language (see Initialize). See GetPrinter for a note on its relationship
// to pruned locale keys.
func GetFallbackPrinter() (p *message.Printer) {
	return getFallbackPrinter()
}

// Reload applies every translation queued via
// QueueTranslation/QueueTranslations and republishes the immutable render
// snapshot T/TL read from. It's only necessary to call directly when using
// those two - RegisterTranslation and RegisterTranslations already reload
// as a side effect. Like Initialize, it also drains and returns any issue
// Embed/EmbedIssues has recorded since the last drain - see InitIssue for
// how to handle the returned issues, if any.
func Reload() []InitIssue {
	issues := reload()
	return append(issues, drainPendingIssues()...)
}

// T renders key for the current language (see GetLanguage/SetLanguage),
// interpolating args as named {name}/{name:verb} placeholders (see Arg,
// String, Int, ...). It cascades through the current language's BCP 47
// parent chain, the key's owning bundle's declared source locale, and
// finally the global fallback language (see Initialize); a key that
// matches nothing anywhere is returned unchanged, exactly as given.
func T(key string, args ...any) string {
	return t(key, args...)
}

// TL is T for an explicit language tag rather than the current language -
// useful for rendering in a language other than the process-wide current
// one, e.g. per-request in a server handling several locales at once (see
// I18nMiddleware, WithLanguage, LanguageFromContext).
func TL(lang language.Tag, key string, args ...any) string {
	return tForLanguage(lang, key, args...)
}

// TD is T with a caller-supplied fallback: it renders key for the current
// language exactly as T would, except when key has no registered
// translation anywhere (T's own last resort would otherwise return key
// itself), in which case it returns fallback instead. Useful for a value
// that's meaningful as plain text even before any translation for it has
// been registered - e.g. a CLI flag's usage string, which should read
// sensibly whether or not i18n has been initialized at all.
func TD(key string, fallback string, args ...any) string {
	result := t(key, args...)
	if result == key {
		return fallback
	}
	return result
}

// PT is T for a key given as a prefix and a key local to it, joined with
// ".". It exists for a package that already holds its own reverse-DNS
// prefix (see GetPackagePrefix) and wants to render many of its own keys
// without re-composing the full key at every call site:
//
//	i18np := i18n.GetPackagePrefix()
//	i18n.PT(i18np, "greeting", "name", "World")
func PT(prefix, localKey string, args ...any) string {
	return t(fmt.Sprintf("%s.%s", prefix, localKey), args...)
}

// PTD combines PT and TD: it renders prefix+"."+localKey for the current
// language, falling back to the caller-supplied fallback if that key has
// no registered translation anywhere.
func PTD(prefix, localKey, fallback string, args ...any) string {
	key := fmt.Sprintf("%s.%s", prefix, localKey)
	result := t(key, args...)
	if result == key {
		return fallback
	}
	return result
}

// GetPackagePrefix returns the caller's own reverse-DNS translation key
// root, derived from its import path exactly as NewError does for its own
// keys (e.g. "github.com/happy-sdk/happy/pkg/example" becomes
// "com.github.happy-sdk.happy.pkg.example") - so a package can compose its
// own keys (see PT, PTD) without hardcoding its import path a second time
// in Go source.
func GetPackagePrefix() string {
	return composePackageKey("", 2)
}

// GetAllTranslations returns all registered translation entries.
// Each entry contains the translation key, the fallback value, and
// translations for all supported languages.
//
// This can be used to:
//   - List all available translation keys
//   - Compute translation status (percentage of how much each language is translated)
//   - Export translations for external tools
func GetAllTranslations() []TranslationEntry {
	return getAllTranslations()
}

// GetTranslationReport returns a translation report for the specified language.
// The report includes the translation percentage and a list of entries
// that are missing translations for that language.
func GetTranslationReport(lang language.Tag) TranslationReport {
	return getTranslationReport(lang)
}
