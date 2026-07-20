// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package changelog

import (
	"encoding/json"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func TestRenderJSON(t *testing.T) {
	out, err := fixtureDocument().Render(FormatJSON)
	testutils.NoError(t, err)

	var doc jsonDocument
	testutils.NoError(t, json.Unmarshal(out, &doc), "expected valid JSON")

	testutils.Equal(t, jsonSchemaID, doc.Schema, "unexpected $schema value")
	testutils.Equal(t, "Changelog", doc.Title, "unexpected title")
	testutils.Equal(t, 2, len(doc.Sections), "unexpected section count")

	root := doc.Sections[0]
	testutils.Equal(t, "github.com/happy-sdk/happy@v1.2.0", root.Title, "unexpected root section title")
	testutils.Equal(t, 1, len(root.Breaking), "unexpected root breaking count")
	testutils.Equal(t, 1, len(root.Changes), "unexpected root changes count")
	testutils.Equal(t, "removed deprecated Foo()", root.Breaking[0].Subject, "unexpected breaking subject")
	testutils.Equal(t, "feat", root.Changes[0].Type, "unexpected change type")

	sub := doc.Sections[1]
	testutils.Equal(t, "github.com/happy-sdk/happy/pkg/sub@v0.2.0", sub.Title, "unexpected sub section title")
	testutils.Equal(t, 0, len(sub.Breaking), "expected no breaking changes in sub section")
	testutils.Equal(t, 1, len(sub.Changes), "unexpected sub changes count")
}

func TestRenderJSONEmptyDocument(t *testing.T) {
	out, err := NewDocument("Changelog").Render(FormatJSON)
	testutils.NoError(t, err)

	var doc jsonDocument
	testutils.NoError(t, json.Unmarshal(out, &doc), "expected valid JSON")
	testutils.Equal(t, 0, len(doc.Sections), "expected no sections")
}

func TestJSONSchemaIsValidJSON(t *testing.T) {
	var schema map[string]any
	testutils.NoError(t, json.Unmarshal(JSONSchema(), &schema), "expected JSONSchema() to return valid JSON")
	testutils.Equal(t, "https://json-schema.org/draft-07/schema#", schema["$schema"], "unexpected $schema meta-schema")
	testutils.Equal(t, jsonSchemaID, schema["$id"], "unexpected $id")
}
