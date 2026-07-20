// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package changelog

import "encoding/json"

// jsonSchemaID identifies the shape produced by Document.Render(FormatJSON)
// and described by JSONSchema. It's a urn, not a hosted URL - this package
// doesn't publish or resolve it anywhere, it's just a stable identifier.
const jsonSchemaID = "urn:happy-sdk:lib:devel:changelog:document"

// JSONSchema returns the JSON Schema (draft-07) describing the shape
// produced by Document.Render(FormatJSON).
func JSONSchema() []byte {
	return []byte(jsonSchemaDoc)
}

type jsonEntry struct {
	ShortHash string `json:"short_hash,omitempty"`
	LongHash  string `json:"long_hash,omitempty"`
	Author    string `json:"author,omitempty"`
	Subject   string `json:"subject"`
	Type      string `json:"type,omitempty"`
	Scope     string `json:"scope,omitempty"`
}

type jsonSection struct {
	Title    string      `json:"title"`
	Breaking []jsonEntry `json:"breaking,omitempty"`
	Changes  []jsonEntry `json:"changes,omitempty"`
}

type jsonDocument struct {
	Schema   string        `json:"$schema"`
	Title    string        `json:"title,omitempty"`
	Sections []jsonSection `json:"sections"`
}

func entryToJSON(e Entry) jsonEntry {
	return jsonEntry{
		ShortHash: e.ShortHash,
		LongHash:  e.LongHash,
		Author:    e.Author,
		Subject:   e.Subject,
		Type:      e.Typ.Typ,
		Scope:     e.Typ.Scope,
	}
}

func (d *Document) renderJSON() ([]byte, error) {
	doc := jsonDocument{
		Schema:   jsonSchemaID,
		Title:    d.Title,
		Sections: make([]jsonSection, 0, len(d.Sections)),
	}

	for _, s := range d.Sections {
		js := jsonSection{Title: s.Title}
		if s.Release != nil {
			for _, e := range s.Release.Breaking() {
				js.Breaking = append(js.Breaking, entryToJSON(e))
			}
			for _, e := range s.Release.Entries() {
				js.Changes = append(js.Changes, entryToJSON(e))
			}
		}
		doc.Sections = append(doc.Sections, js)
	}

	return json.MarshalIndent(doc, "", "  ")
}

const jsonSchemaDoc = `{
  "$schema": "https://json-schema.org/draft-07/schema#",
  "$id": "urn:happy-sdk:lib:devel:changelog:document",
  "title": "Changelog Document",
  "type": "object",
  "required": ["$schema", "sections"],
  "properties": {
    "$schema": { "type": "string" },
    "title": { "type": "string" },
    "sections": {
      "type": "array",
      "items": { "$ref": "#/definitions/section" }
    }
  },
  "definitions": {
    "section": {
      "type": "object",
      "required": ["title"],
      "properties": {
        "title": { "type": "string" },
        "breaking": {
          "type": "array",
          "items": { "$ref": "#/definitions/entry" }
        },
        "changes": {
          "type": "array",
          "items": { "$ref": "#/definitions/entry" }
        }
      }
    },
    "entry": {
      "type": "object",
      "required": ["subject"],
      "properties": {
        "short_hash": { "type": "string" },
        "long_hash": { "type": "string" },
        "author": { "type": "string" },
        "subject": { "type": "string" },
        "type": { "type": "string" },
        "scope": { "type": "string" }
      }
    }
  }
}
`
