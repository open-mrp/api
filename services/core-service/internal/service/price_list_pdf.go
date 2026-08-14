package service

import (
	"bytes"
	"strings"

	"github.com/go-pdf/fpdf"
)

// Page geometry (A4 portrait, 15mm margins => 180mm of content).
const (
	plPageLeft     = 15.0
	plPageRight    = 195.0
	plPageBottom   = 282.0
	plContentWidth = plPageRight - plPageLeft
	plRowHeight    = 5.0
	plHeaderHeight = 6.5
)

// priceListDocument is everything the renderer needs; the caller has already priced and grouped the catalog.
type priceListDocument struct {
	MerchantName  string
	CustomerName  string
	DateLong      string
	PaymentTerm   string
	ShippingTerm  string
	LogoImageType string
	LogoImage     []byte
	Lines         []priceListLine
	// Notes are printed on the title page — used to disclose anything the export had to leave out, so the document never overstates its own completeness.
	Notes []string
}

// priceListRenderer carries the fpdf document together with its text translator. fpdf's core fonts are cp1252, so every string has to go through tr or non-ASCII characters (the heading separator, the truncation ellipsis, accented names) render as mojibake.
type priceListRenderer struct {
	pdf *fpdf.Fpdf
	tr  func(string) string
}

// buildPriceListPDF renders the customer price list: a title page, then one section per group of SKUs that share a price, with repeated attribute values merged vertically the way a printed price list reads.
func buildPriceListPDF(doc priceListDocument) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(plPageLeft, 15, 15)
	pdf.SetAutoPageBreak(false, 15)
	r := &priceListRenderer{pdf: pdf, tr: pdf.UnicodeTranslatorFromDescriptor("")}

	r.titlePage(doc)

	for _, line := range doc.Lines {
		pdf.AddPage()
		r.productLineHeader(line)
		for _, section := range line.Sections {
			r.section(line, section)
		}
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (r *priceListRenderer) titlePage(doc priceListDocument) {
	pdf := r.pdf
	pdf.AddPage()
	y := 55.0

	if len(doc.LogoImage) > 0 {
		name := "price-list-logo"
		opts := fpdf.ImageOptions{ImageType: doc.LogoImageType, ReadDpi: true}
		pdf.RegisterImageOptionsReader(name, opts, bytes.NewReader(doc.LogoImage))
		if info := pdf.GetImageInfo(name); info != nil && info.Width() > 0 {
			width, height := 90.0, 90.0*info.Height()/info.Width()
			if height > 45 {
				height = 45
				width = height * info.Width() / info.Height()
			}
			pdf.ImageOptions(name, plPageLeft+(plContentWidth-width)/2, y, width, height, false, opts, 0, "")
			y += height + 6
		}
	}

	pdf.SetXY(plPageLeft, y)
	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(plContentWidth, 8, r.tr(doc.MerchantName), "", 2, "C", false, 0, "")
	pdf.Ln(6)

	pdf.SetFont("Helvetica", "B", 13)
	pdf.CellFormat(plContentWidth, 7, r.tr(strings.ToUpper(doc.CustomerName)+" PRICE LIST"), "", 2, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(plContentWidth, 7, r.tr(doc.DateLong), "", 2, "C", false, 0, "")
	pdf.Ln(10)

	pdf.SetFont("Helvetica", "B", 9.5)
	if doc.PaymentTerm != "" {
		pdf.CellFormat(plContentWidth, 5.5, r.tr("TERMS: "+doc.PaymentTerm), "", 2, "C", false, 0, "")
	}
	if doc.ShippingTerm != "" {
		pdf.CellFormat(plContentWidth, 5.5, r.tr("FREIGHT: "+doc.ShippingTerm), "", 2, "C", false, 0, "")
	}

	if len(doc.Notes) > 0 {
		pdf.SetXY(plPageLeft, 240)
		pdf.SetFont("Helvetica", "", 7.5)
		pdf.SetTextColor(110, 110, 110)
		for _, note := range doc.Notes {
			pdf.CellFormat(plContentWidth, 4, r.tr(note), "", 2, "C", false, 0, "")
		}
		pdf.SetTextColor(0, 0, 0)
	}

	pdf.SetXY(plPageLeft, 270)
	pdf.SetFont("Helvetica", "", 8)
	pdf.CellFormat(plContentWidth, 4, r.tr("Rev. "+doc.DateLong), "", 0, "R", false, 0, "")
}

func (r *priceListRenderer) productLineHeader(line priceListLine) {
	r.pdf.SetXY(plPageLeft, 18)
	r.pdf.SetFont("Helvetica", "B", 15)
	r.pdf.CellFormat(plContentWidth, 8, r.tr(strings.ToUpper(line.ProductLineName)), "", 2, "L", false, 0, "")
	r.pdf.Ln(2)
}

// section renders one table, breaking across pages when it runs out of room. A run of merged cells interrupted by a page break is closed at the bottom and reopened with its value repeated at the top of the next page.
func (r *priceListRenderer) section(line priceListLine, section priceListSection) {
	widths, headers := plColumns(line, section)

	rows := section.Rows
	first := true
	for len(rows) > 0 {
		if r.pdf.GetY()+plHeaderHeight+plRowHeight*2 > plPageBottom {
			r.pdf.AddPage()
			r.productLineHeader(line)
			first = true
		}

		if first && section.Heading != "" {
			r.sectionHeading(section.Heading)
		}
		r.tableHeader(widths, headers)

		fit := min(int((plPageBottom-r.pdf.GetY())/plRowHeight), len(rows))
		if fit <= 0 {
			fit = 1
		}
		r.tableRows(widths, section, rows[:fit])
		rows = rows[fit:]
		first = false

		if len(rows) > 0 {
			r.pdf.AddPage()
			r.productLineHeader(line)
		}
	}
	r.pdf.Ln(5)
}

func (r *priceListRenderer) sectionHeading(heading string) {
	r.pdf.SetFont("Helvetica", "B", 10)
	r.pdf.SetFillColor(238, 238, 238)
	r.pdf.CellFormat(plContentWidth, plHeaderHeight, r.tr(" "+heading), "1", 1, "L", true, 0, "")
}

// plColumns lays out the table: one column per varying attribute, then the catalog number, description and pack, then one column per surviving volume tier.
func plColumns(line priceListLine, section priceListSection) ([]float64, []string) {
	headers := make([]string, 0, len(section.Columns)+3+len(section.Tiers))
	headers = append(headers, section.Columns...)
	headers = append(headers, "Catalog #", "Product Information", "Packing")
	for _, tier := range section.Tiers {
		label := line.BaseUnitName + " Cost"
		if len(section.Tiers) > 1 {
			label = tier.Label
		}
		headers = append(headers, label)
	}

	attrWidth := 0.0
	if len(section.Columns) > 0 {
		attrWidth = 20.0
	}
	priceWidth := 22.0
	remaining := max(plContentWidth-attrWidth*float64(len(section.Columns))-priceWidth*float64(len(section.Tiers)), 70.0)
	catalogWidth := remaining * 0.26
	packWidth := remaining * 0.34
	infoWidth := remaining - catalogWidth - packWidth

	widths := make([]float64, 0, len(headers))
	for range section.Columns {
		widths = append(widths, attrWidth)
	}
	widths = append(widths, catalogWidth, infoWidth, packWidth)
	for range section.Tiers {
		widths = append(widths, priceWidth)
	}
	return widths, headers
}

func (r *priceListRenderer) tableHeader(widths []float64, headers []string) {
	r.pdf.SetFont("Helvetica", "B", 6.5)
	r.pdf.SetFillColor(248, 248, 248)
	for i, header := range headers {
		r.pdf.CellFormat(widths[i], plHeaderHeight, r.fit(strings.ToUpper(header), widths[i]), "1", 0, "C", true, 0, "")
	}
	r.pdf.Ln(-1)
}

// tableRows draws one page's worth of rows, merging vertically repeated attribute, description, pack and price cells into single tall cells.
func (r *priceListRenderer) tableRows(widths []float64, section priceListSection, rows []priceListRow) {
	attrCount := len(section.Columns)

	attrValues := make([][]string, len(rows))
	descriptions := make([]string, len(rows))
	packs := make([]string, len(rows))
	for i, row := range rows {
		attrValues[i] = row.Values
		descriptions[i] = row.Description
		packs[i] = row.Packing
	}
	attrSpans := mergeSpansNested(attrValues, attrCount)
	descriptionSpans := mergeSpans(descriptions)
	packSpans := mergeSpans(packs)

	priceSpans := make([][]int, len(section.Tiers))
	for t := range section.Tiers {
		column := make([]string, len(rows))
		for i, row := range rows {
			if t < len(row.Prices) {
				column[i] = row.Prices[t]
			}
		}
		priceSpans[t] = mergeSpans(column)
	}

	top := r.pdf.GetY()
	for i, row := range rows {
		y := top + float64(i)*plRowHeight
		x := plPageLeft

		r.pdf.SetFont("Helvetica", "", 6.5)
		for c := range attrCount {
			if span := attrSpans[i][c]; span > 0 {
				r.mergedCell(x, y, widths[c], plRowHeight*float64(span), row.Values[c], "C")
			}
			x += widths[c]
		}

		r.mergedCell(x, y, widths[attrCount], plRowHeight, row.SKU, "C")
		x += widths[attrCount]

		if span := descriptionSpans[i]; span > 0 {
			r.mergedCell(x, y, widths[attrCount+1], plRowHeight*float64(span), row.Description, "L")
		}
		x += widths[attrCount+1]

		if span := packSpans[i]; span > 0 {
			r.mergedCell(x, y, widths[attrCount+2], plRowHeight*float64(span), row.Packing, "C")
		}
		x += widths[attrCount+2]

		for t := range section.Tiers {
			width := widths[attrCount+3+t]
			if span := priceSpans[t][i]; span > 0 {
				value := ""
				if t < len(row.Prices) {
					value = row.Prices[t]
				}
				r.pdf.SetFont("Helvetica", "B", 7.5)
				r.mergedCell(x, y, width, plRowHeight*float64(span), value, "C")
				r.pdf.SetFont("Helvetica", "", 6.5)
			}
			x += width
		}
	}
	r.pdf.SetXY(plPageLeft, top+float64(len(rows))*plRowHeight)
}

// mergedCell draws one bordered cell of arbitrary height with its text vertically centered.
func (r *priceListRenderer) mergedCell(x, y, w, h float64, text, align string) {
	r.pdf.Rect(x, y, w, h, "D")
	r.pdf.SetXY(x, y+(h-plRowHeight)/2)
	r.pdf.CellFormat(w, plRowHeight, r.fit(text, w), "", 0, align, false, 0, "")
}

// fit translates text into the font's encoding and trims it to the cell width. Rows are a fixed height and must not wrap, so overlong values are cut — by rune, since cutting mid-sequence would corrupt the character.
func (r *priceListRenderer) fit(text string, width float64) string {
	encoded := r.tr(text)
	padded := width - 2
	if r.pdf.GetStringWidth(encoded) <= padded {
		return encoded
	}
	runes := []rune(text)
	for len(runes) > 1 {
		runes = runes[:len(runes)-1]
		candidate := r.tr(string(runes) + "...")
		if r.pdf.GetStringWidth(candidate) <= padded {
			return candidate
		}
	}
	return r.tr(string(runes))
}
