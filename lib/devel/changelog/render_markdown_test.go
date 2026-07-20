// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package changelog

import (
	"strings"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func TestRenderMarkdownDefault(t *testing.T) {
	out, err := fixtureDocument().Render(FormatMarkdown)
	testutils.NoError(t, err)
	content := string(out)

	for _, want := range []string{
		"## Changelog",
		"### github.com/happy-sdk/happy@v1.2.0",
		"**Breaking Changes**",
		"* bbb2222 removed deprecated Foo()",
		"**Changes**",
		"* aaa1111 add a new feature",
		"### github.com/happy-sdk/happy/pkg/sub@v0.2.0",
		"* ccc3333 fix a bug in sub",
	} {
		testutils.Assert(t, strings.Contains(content, want), "expected markdown to contain %q, got:\n%s", want, content)
	}
}

func TestRenderMarkdownNoBreakingSectionWhenEmpty(t *testing.T) {
	r := NewRelease()
	r.Add("h", "h", "a", "just a change", EntryType{Typ: "fix", Kind: EntryKindPatch})
	d := NewDocument("Changelog").AddSection("pkg@v0.1.0", r)

	out, err := d.Render(FormatMarkdown)
	testutils.NoError(t, err)
	testutils.Assert(t, !strings.Contains(string(out), "Breaking Changes"), "expected no breaking changes section when there are no breaking changes")
}

func TestRenderMarkdownCustomTemplate(t *testing.T) {
	out, err := fixtureDocument().Render(FormatMarkdown, WithTemplate("TITLE={{.Title}} SECTIONS={{len .Sections}}"))
	testutils.NoError(t, err)
	testutils.Equal(t, "TITLE=Changelog SECTIONS=2", string(out), "unexpected custom template output")
}

func TestRenderMarkdownCustomTemplateInvalid(t *testing.T) {
	_, err := fixtureDocument().Render(FormatMarkdown, WithTemplate("{{.NoSuchField"))
	testutils.Error(t, err, "expected an error for an invalid template")
}

func TestRenderMarkdownCustomTemplateExecuteError(t *testing.T) {
	_, err := fixtureDocument().Render(FormatMarkdown, WithTemplate("{{index .Sections 99}}"))
	testutils.Error(t, err, "expected an error when the template fails during execution")
}

func TestRenderMarkdownSectionWithNilRelease(t *testing.T) {
	d := &Document{Title: "Changelog", Sections: []Section{{Title: "pkg@v0.1.0", Release: nil}}}
	out, err := d.Render(FormatMarkdown)
	testutils.NoError(t, err)
	testutils.Assert(t, strings.Contains(string(out), "### pkg@v0.1.0"), "expected the section heading even with a nil release")
}
