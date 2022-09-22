// Copyright 2022 The Happy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"fmt"
	"os"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/cli/color"
)

type banner struct {
	cliTmplParser
	Info struct {
		Title          string
		CopyrightBy    string
		CopyrightSince int
		License        string
		Version        string
		Description    string
	}
}

func Banner(s happy.Session) {
	b := &banner{}
	b.Defaults()
	b.Info.Title = s.Get("app.title").String()
	b.Info.CopyrightBy = s.Get("app.copyright.by").String()
	b.Info.CopyrightSince = s.Get("app.copyright.since").Int()
	b.Info.License = s.Get("app.license").String()
	b.Info.Version = s.Get("app.version").String()
	b.Info.Description = s.Get("app.description").String()
	b.Print()
}

func (b *banner) Defaults() {
	b.SetTemplate(`  {{ .Title }}{{ if .Version }}
  {{ .Version }}{{end}}{{ if .CopyrightBy }}
  Copyright © {{ if .CopyrightSince }}{{ .CopyrightSince }} {{ end }}{{ if (gt funcYear  .CopyrightSince) }}- {{ funcYear }} {{ end }}{{ .CopyrightBy }}. All rights reserved.{{end}}{{ if .License }}
  License:      {{ .License }}{{ end }}
  {{ if .Description }}
  {{ .Description }}{{end}}
  `)
}

// Print application header.
func (b *banner) Print() error {
	usecolor := "yellow"
	var col []byte
	switch usecolor {
	case "red":
		col = color.Red
	case "green":
		col = color.Green
	case "white":
		col = color.White
	case "gray":
		col = color.Gray
	case "darkgray":
		col = color.GrayDark
	case "yellow":
		col = color.Yellow
	case "cyan":
		col = color.Cyan
	case "blue":
		col = color.Blue
	case "black":
		col = color.Black
	default:
		col = color.Gray
	}
	b.buffer.Write(col)
	err := b.ParseTmpl("header-tmpl", b.Info, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	b.buffer.Write([]byte{27, 91, 48, 109})
	fmt.Fprintln(os.Stdout, b.buffer.String())
	return nil
}
