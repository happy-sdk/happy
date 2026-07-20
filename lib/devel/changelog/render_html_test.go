// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package changelog

import (
	"strings"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func TestRenderHTMLDefault(t *testing.T) {
	out, err := fixtureDocument().Render(FormatHTML)
	testutils.NoError(t, err)
	content := string(out)

	testutils.HasPrefix(t, content, "<!doctype html>", "expected a full static HTML document")
	for _, want := range []string{
		"<title>Changelog</title>",
		"<h1>Changelog</h1>",
		"<h2>github.com/happy-sdk/happy@v1.2.0</h2>",
		"<h3>Breaking Changes</h3>",
		"removed deprecated Foo()",
		"<h3>Changes</h3>",
		"add a new feature",
		"<h2>github.com/happy-sdk/happy/pkg/sub@v0.2.0</h2>",
		"fix a bug in sub",
	} {
		testutils.Assert(t, strings.Contains(content, want), "expected html to contain %q, got:\n%s", want, content)
	}
}

func TestRenderHTMLEscapesContent(t *testing.T) {
	r := NewRelease()
	r.Add("h", "h", "a", "<script>alert(1)</script>", EntryType{Typ: "fix", Kind: EntryKindPatch})
	d := NewDocument("Changelog").AddSection("pkg@v0.1.0", r)

	out, err := d.Render(FormatHTML)
	testutils.NoError(t, err)
	testutils.Assert(t, !strings.Contains(string(out), "<script>alert(1)</script>"), "expected html/template to escape subject content")
	testutils.Assert(t, strings.Contains(string(out), "&lt;script&gt;"), "expected escaped script tag in output")
}

func TestRenderHTMLCustomTemplate(t *testing.T) {
	out, err := fixtureDocument().Render(FormatHTML, WithTemplate("<p>{{.Title}}</p>"))
	testutils.NoError(t, err)
	testutils.Equal(t, "<p>Changelog</p>", string(out), "unexpected custom template output")
}

func TestRenderHTMLCustomTemplateInvalid(t *testing.T) {
	_, err := fixtureDocument().Render(FormatHTML, WithTemplate("{{.NoSuchField"))
	testutils.Error(t, err, "expected an error for an invalid template")
}

func TestRenderHTMLCustomTemplateExecuteError(t *testing.T) {
	_, err := fixtureDocument().Render(FormatHTML, WithTemplate("{{index .Sections 99}}"))
	testutils.Error(t, err, "expected an error when the template fails during execution")
}
