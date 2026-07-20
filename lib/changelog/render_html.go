// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package changelog

import (
	"bytes"
	"fmt"
	"html/template"
)

// defaultHTMLTemplate renders a complete, self-contained static HTML page -
// no external stylesheets or scripts - suitable for dropping straight onto
// an HTTP file server.
const defaultHTMLTemplate = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<style>
body{font-family:system-ui,sans-serif;max-width:48rem;margin:2rem auto;padding:0 1rem;line-height:1.5;color:#1a1a1a}
h1{border-bottom:1px solid #ddd;padding-bottom:.5rem}
h2{margin-top:2rem}
h3{margin-bottom:.25rem}
ul{margin-top:.25rem}
code{background:#f4f4f4;padding:.1rem .3rem;border-radius:3px}
@media (prefers-color-scheme: dark){
  body{background:#111;color:#eee}
  h1{border-color:#333}
  code{background:#222}
}
</style>
</head>
<body>
<h1>{{.Title}}</h1>
{{range .Sections}}
<h2>{{.Title}}</h2>
{{if .Release}}
{{if .Release.Breaking}}
<h3>Breaking Changes</h3>
<ul>
{{range .Release.Breaking}}<li><code>{{.ShortHash}}</code> {{.Subject}}</li>
{{end}}
</ul>
{{end}}
{{if .Release.Entries}}
<h3>Changes</h3>
<ul>
{{range .Release.Entries}}<li><code>{{.ShortHash}}</code> {{.Subject}}</li>
{{end}}
</ul>
{{end}}
{{end}}
{{end}}
</body>
</html>
`

func (d *Document) renderHTML(o renderOptions) ([]byte, error) {
	src := defaultHTMLTemplate
	if o.template != "" {
		src = o.template
	}
	tmpl, err := template.New("changelog-html").Parse(src)
	if err != nil {
		return nil, fmt.Errorf("changelog: parsing html template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, d); err != nil {
		return nil, fmt.Errorf("changelog: executing html template: %w", err)
	}
	return buf.Bytes(), nil
}
