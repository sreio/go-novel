package pdf

import (
    "github.com/jung-kurt/gofpdf"
)

type Meta struct { Title, Author string }
type Chapter struct { Title, Content string }

func Save(path string, meta Meta, chapters []Chapter) error {
    pdf := gofpdf.New("P", "mm", "A4", "")
    pdf.SetTitle(meta.Title, false)
    pdf.SetAuthor(meta.Author, false)
    for _, ch := range chapters {
        pdf.AddPage()
        pdf.SetFont("Arial", "B", 16)
        pdf.MultiCell(0, 10, ch.Title, "", "L", false)
        pdf.Ln(4)
        pdf.SetFont("Arial", "", 12)
        pdf.MultiCell(0, 6, ch.Content, "", "L", false)
    }
    return pdf.OutputFileAndClose(path)
}
