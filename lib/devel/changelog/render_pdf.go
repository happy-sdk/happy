// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package changelog

import (
	"bytes"
	"fmt"

	"codeberg.org/go-pdf/fpdf"
)

func (d *Document) renderPDF() ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetTitle(d.Title, true)
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()

	pdf.SetFont("Helvetica", "B", 18)
	pdf.CellFormat(0, 10, d.Title, "", 1, "L", false, 0, "")
	pdf.Ln(4)

	for _, s := range d.Sections {
		pdf.SetFont("Helvetica", "B", 13)
		pdf.CellFormat(0, 8, s.Title, "", 1, "L", false, 0, "")

		if s.Release == nil {
			pdf.Ln(4)
			continue
		}

		if breaking := s.Release.Breaking(); len(breaking) > 0 {
			pdf.SetFont("Helvetica", "B", 11)
			pdf.CellFormat(0, 6, "Breaking Changes", "", 1, "L", false, 0, "")
			pdf.SetFont("Helvetica", "", 10)
			for _, e := range breaking {
				pdf.MultiCell(0, 5, fmt.Sprintf("- %s %s", e.ShortHash, e.Subject), "", "L", false)
			}
			pdf.Ln(2)
		}

		if changes := s.Release.Entries(); len(changes) > 0 {
			pdf.SetFont("Helvetica", "B", 11)
			pdf.CellFormat(0, 6, "Changes", "", 1, "L", false, 0, "")
			pdf.SetFont("Helvetica", "", 10)
			for _, e := range changes {
				pdf.MultiCell(0, 5, fmt.Sprintf("- %s %s", e.ShortHash, e.Subject), "", "L", false)
			}
		}

		pdf.Ln(4)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("changelog: rendering pdf: %w", err)
	}
	return buf.Bytes(), nil
}
