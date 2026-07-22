// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

// Package v2 defines schema version 2: a translation file that describes
// its own bundle identity and the locale(s) it carries, instead of relying
// on a filename to supply the locale (see the sibling v1 package for the
// original, unversioned shape this supersedes).
//
// A v2 document's top-level shape is fixed - KeyBundle and KeyLocales are
// required, KeySource is optional - and no other key is allowed (besides
// the parent schema package's KeyVersion, already removed by the time
// Parse sees the body). Each KeyLocales entry is itself an object with a
// required KeyLocaleKeys (the locale's own translation tree) and an
// optional KeyLocaleNotes (translator/tooling context for that locale
// specifically, not the whole bundle - different languages can need
// different context):
//
//	{
//	  "version": 2,
//	  "bundle": "com.github.example.app",
//	  "source": "en",
//	  "locales": {
//	    "en": {
//	      "notes": "optional, free text - context for this locale's translations",
//	      "keys": { "greeting": {"$message": {"msg": "hello"}} }
//	    },
//	    "de": {
//	      "keys": { "greeting": {"$message": {"msg": "hallo"}} }
//	    }
//	  }
//	}
//
// "locales" may hold as few or as many entries as its author wants - a
// maintainer who prefers one file per language (this monorepo's own
// convention) writes several v2 documents side by side, each with a single
// "locales" entry (filenames don't matter to this schema at all - see the
// parent schema package's dispatch), while one who wants a single
// exportable bundle file can put every supported locale in one document.
// Both are the same schema; a bundle split across several files should
// repeat the same "bundle"/"source" in each one.
//
// "source" names the locale this bundle's translations are authored/
// maintained in - the one every other locale is translated from. It
// defaults to DefaultSource ("en") when omitted, so an English-authored
// bundle (the common case) never needs to declare it at all.
//
// Every key within a locale's own KeyLocaleKeys tree - the path leading
// down to a message - must be lowercase a-z, 0-9, or "_", with one
// exception: v1.MessageKey ("$message" - unchanged from schema version 1)
// marks a message value and is never itself subject to this rule, nor is
// anything nested inside it (that's v1.ParseMessage's own fixed schema,
// reused here as-is - the message value shape hasn't changed between
// versions at all).
package v2
