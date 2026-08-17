package service

import (
	"bytes"
	"sort"
	"strings"

	"github.com/go-pdf/fpdf"
)

// Page geometry (A4 portrait, 15mm margins => 180mm of content).
const (
	plPageLeft     = 15.0
	plPageRight    = 195.0
	plPageBottom   = 282.0
	plContentWidth = plPageRight - plPageLeft
	// plLineHeight is one line of wrapped cell text; plRowHeight is what a single-line row therefore costs.
	plLineHeight    = 3.2
	plCellPadding   = 1.8
	plRowHeight     = plLineHeight + plCellPadding
	plHeaderHeight  = 6.5
	plBodyFontSize  = 6.5
	plPriceFontSize = 7.5
	// plCellMargin is the horizontal space fpdf keeps inside a cell (1mm per side), which is what wrapping and width measurement have to leave free.
	plCellMargin = 2.0
	// plColumnSlack keeps a content-sized column off its longest value.
	plColumnSlack = 2.5
	// Bounds on the content-sized columns. The minimum keeps a narrow column legible; the maximum stops one long attribute value from eating the table.
	plMinColumnWidth = 11.0
	plMaxColumnWidth = 45.0
	// plMinInfoWidth is the floor for the product information column, which is the one column that is expected to wrap.
	plMinInfoWidth = 42.0
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

// plCell is one drawn cell: its text already wrapped and encoded, and how many rows it swallows. A span of 0 means the cell above swallowed this row.
type plCell struct {
	Lines []string
	Span  int
	Align string
	Style string
	Size  float64
}

// plColumn is one resolved column: how its body cells are drawn, and how wide the column ended up.
type plColumn struct {
	Header string
	Width  float64
	Align  string
	Style  string
	Size   float64
}

// plTable is a section resolved to geometry: final column widths, wrapped cells and the height every row needs.
type plTable struct {
	Columns      []plColumn
	Header       []plCell
	HeaderHeight float64
	// Cells is indexed [row][column].
	Cells   [][]plCell
	Heights []float64
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
	table := r.buildTable(line, section)
	if len(table.Cells) == 0 {
		return
	}

	first := true
	for start := 0; start < len(table.Cells); {
		if r.pdf.GetY()+table.HeaderHeight+table.Heights[start] > plPageBottom {
			r.pdf.AddPage()
			r.productLineHeader(line)
			first = true
		}

		if first && section.Heading != "" {
			r.sectionHeading(section.Heading)
		}
		r.tableHeader(table)

		// At least one row per page, even where it cannot fit: an unbreakable row that always defers would loop forever.
		end := start
		for y := r.pdf.GetY(); end < len(table.Cells) && (end == start || y+table.Heights[end] <= plPageBottom); end++ {
			y += table.Heights[end]
		}
		if end < len(table.Cells) {
			table.splitAt(end)
		}
		r.tableRows(table, start, end)
		start = end
		first = false

		if start < len(table.Cells) {
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

// buildTable resolves a section into drawable geometry: the columns and their widths, every cell wrapped to the width it landed on, and the row heights that wrapping implies.
func (r *priceListRenderer) buildTable(line priceListLine, section priceListSection) *plTable {
	columns := plTableColumns(line, section)
	texts := plTableTexts(section)
	spans := plTableSpans(section)
	r.resolveColumnWidths(columns, texts, plInfoColumn(section))

	table := &plTable{Columns: columns, Cells: make([][]plCell, len(texts)), Heights: make([]float64, len(texts))}

	table.Header = make([]plCell, len(columns))
	for c, column := range columns {
		lines := make([]string, 0, 2)
		for part := range strings.SplitSeq(strings.ToUpper(column.Header), "\n") {
			lines = append(lines, r.wrap(part, column.Width, "B", plBodyFontSize)...)
		}
		table.Header[c] = plCell{Lines: lines, Span: 1, Align: "C", Style: "B", Size: plBodyFontSize}
	}
	table.HeaderHeight = max(plHeaderHeight, plCellHeight(plMaxCellLines(table.Header)))

	for i := range texts {
		table.Cells[i] = make([]plCell, len(columns))
		table.Heights[i] = plRowHeight
		for c, column := range columns {
			span := spans[i][c]
			if span == 0 {
				continue
			}
			table.Cells[i][c] = plCell{
				Lines: r.wrap(texts[i][c], column.Width, column.Style, column.Size),
				Span:  span,
				Align: column.Align,
				Style: column.Style,
				Size:  column.Size,
			}
		}
	}
	table.growRowsToFitCells()
	return table
}

// plTableColumns names the table: one column per varying attribute, then the catalog number, description and pack, then one column per surviving volume tier.
func plTableColumns(line priceListLine, section priceListSection) []plColumn {
	columns := make([]plColumn, 0, len(section.Columns)+3+len(section.Tiers))
	for _, name := range section.Columns {
		columns = append(columns, plColumn{Header: name, Align: "C", Size: plBodyFontSize})
	}
	columns = append(columns,
		plColumn{Header: "Catalog #", Align: "C", Size: plBodyFontSize},
		plColumn{Header: "Product Information", Align: "L", Size: plBodyFontSize},
		plColumn{Header: "Packing", Align: "C", Size: plBodyFontSize},
	)
	for _, tier := range section.Tiers {
		columns = append(columns, plColumn{Header: plCostHeader(line, section, tier), Align: "C", Style: "B", Size: plPriceFontSize})
	}
	return columns
}

// plCostHeader heads a price column with the unit its price is per, because a price list quotes a bare number and the reader has no other way to know whether it buys a pair or a carton. Volume columns carry the break they apply at above it — the quantity a price is quoted at and the unit it is charged in are not the same thing, and a column that shows only the break invites reading the price as the price of the break.
func plCostHeader(line priceListLine, section priceListSection, tier priceListTier) string {
	// A tier with no unit of its own was priced against each product's own base unit, which within one product line is this line's.
	unit := tier.UnitName
	if unit == "" {
		unit = line.BaseUnitName
	}

	header := "Cost"
	if unit != "" {
		header = "Cost Per " + unit
	}
	if len(section.Tiers) > 1 {
		header = tier.Label + "\n" + header
	}
	return header
}

// plInfoColumn is the index of the product information column, the one column allowed to absorb whatever width the others leave.
func plInfoColumn(section priceListSection) int {
	return len(section.Columns) + 1
}

// plTableTexts flattens a section's rows into the table's column order.
func plTableTexts(section priceListSection) [][]string {
	attributes := len(section.Columns)
	texts := make([][]string, len(section.Rows))
	for i, row := range section.Rows {
		cells := make([]string, attributes+3+len(section.Tiers))
		copy(cells, row.Values)
		cells[attributes] = row.SKU
		cells[attributes+1] = row.Description
		cells[attributes+2] = row.Packing
		for t := range section.Tiers {
			if t < len(row.Prices) {
				cells[attributes+3+t] = row.Prices[t]
			}
		}
		texts[i] = cells
	}
	return texts
}

// plTableSpans computes every column's vertical merges: attributes nest, description, pack and each price column merge on their own, and the catalog number never merges because it is what makes a row a row.
func plTableSpans(section priceListSection) [][]int {
	attributes := len(section.Columns)
	width := attributes + 3 + len(section.Tiers)

	spans := make([][]int, len(section.Rows))
	for i := range spans {
		spans[i] = make([]int, width)
		spans[i][attributes] = 1
	}

	attributeValues := make([][]string, len(section.Rows))
	descriptions := make([]string, len(section.Rows))
	packs := make([]string, len(section.Rows))
	for i, row := range section.Rows {
		attributeValues[i] = row.Values
		descriptions[i] = row.Description
		packs[i] = row.Packing
	}

	for i, nested := range mergeSpansNested(attributeValues, attributes) {
		copy(spans[i], nested)
	}
	for i, span := range mergeSpans(descriptions) {
		spans[i][attributes+1] = span
	}
	for i, span := range mergeSpans(packs) {
		spans[i][attributes+2] = span
	}
	for t := range section.Tiers {
		column := make([]string, len(section.Rows))
		for i, row := range section.Rows {
			if t < len(row.Prices) {
				column[i] = row.Prices[t]
			}
		}
		for i, span := range mergeSpans(column) {
			spans[i][attributes+3+t] = span
		}
	}
	return spans
}

// resolveColumnWidths sizes every column to its own content and hands the slack to the product information column, which is the one column whose text is long enough to be worth wrapping. When the natural widths overflow the page the surplus comes off the information column first, and only then off the rest.
func (r *priceListRenderer) resolveColumnWidths(columns []plColumn, texts [][]string, info int) {
	total := 0.0
	for c := range columns {
		natural := 0.0
		// A header that names its own lines is measured a line at a time; measuring it whole would size the column to a width it never draws at.
		for part := range strings.SplitSeq(strings.ToUpper(columns[c].Header), "\n") {
			natural = max(natural, r.measure(part, "B", plBodyFontSize))
		}
		r.pdf.SetFont("Helvetica", columns[c].Style, columns[c].Size)
		for _, row := range texts {
			natural = max(natural, r.pdf.GetStringWidth(r.tr(row[c])))
		}
		natural += plCellMargin + plColumnSlack
		if c == info {
			natural = max(natural, plMinInfoWidth)
		} else {
			natural = min(max(natural, plMinColumnWidth), plMaxColumnWidth)
		}
		columns[c].Width = natural
		total += natural
	}

	switch {
	case total < plContentWidth:
		columns[info].Width += plContentWidth - total
	case total > plContentWidth:
		// Shrink the information column down to its floor first, then take what is still owed proportionally from the columns that are sized to fit.
		overflow := total - plContentWidth
		reclaimed := min(overflow, columns[info].Width-plMinInfoWidth)
		columns[info].Width -= reclaimed
		overflow -= reclaimed
		if overflow <= 0 {
			return
		}
		others := total - reclaimed - columns[info].Width
		if others <= 0 {
			return
		}
		for c := range columns {
			if c == info {
				continue
			}
			columns[c].Width -= overflow * columns[c].Width / others
		}
	}
}

func (r *priceListRenderer) measure(text, style string, size float64) float64 {
	if text == "" {
		return 0
	}
	r.pdf.SetFont("Helvetica", style, size)
	return r.pdf.GetStringWidth(r.tr(text))
}

// wrap encodes text into the font's cp1252 and breaks it into the lines it takes at this width.
func (r *priceListRenderer) wrap(text string, width float64, style string, size float64) []string {
	if text == "" {
		return nil
	}
	r.pdf.SetFont("Helvetica", style, size)
	chunks := r.pdf.SplitLines([]byte(r.tr(text)), width)
	lines := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		lines = append(lines, string(chunk))
	}
	return lines
}

func plCellHeight(lines int) float64 {
	return max(1, float64(lines))*plLineHeight + plCellPadding
}

// plLinesThatFit inverts plCellHeight. It rounds rather than truncates because a height built by summing line heights lands a hair under its own line count in binary — truncating there drops the last line of a cell that was measured to fit exactly.
func plLinesThatFit(height float64) int {
	return max(1, int((height-plCellPadding)/plLineHeight+0.001))
}

func plMaxCellLines(cells []plCell) int {
	longest := 1
	for _, cell := range cells {
		longest = max(longest, len(cell.Lines))
	}
	return longest
}

// growRowsToFitCells raises row heights until every cell has room for its wrapped text, spreading a merged cell's requirement across the rows it spans. Shortest spans are settled first so a merged cell only claims the height its rows still lack.
func (t *plTable) growRowsToFitCells() {
	type need struct {
		row, span int
		height    float64
	}
	needs := make([]need, 0)
	for i := range t.Cells {
		for _, cell := range t.Cells[i] {
			if cell.Span > 0 && len(cell.Lines) > 1 {
				needs = append(needs, need{row: i, span: cell.Span, height: plCellHeight(len(cell.Lines))})
			}
		}
	}
	sort.SliceStable(needs, func(i, j int) bool { return needs[i].span < needs[j].span })

	for _, n := range needs {
		available := 0.0
		for _, height := range t.Heights[n.row : n.row+n.span] {
			available += height
		}
		if n.height <= available {
			continue
		}
		for row := n.row; row < n.row+n.span; row++ {
			t.Heights[row] += (n.height - available) / float64(n.span)
		}
	}
}

// splitAt closes every merged run that straddles the row, reopening it at that row so the part carried onto the next page repeats its value.
func (t *plTable) splitAt(row int) {
	for c := range t.Columns {
		if t.Cells[row][c].Span > 0 {
			continue
		}
		start := row - 1
		for start >= 0 && t.Cells[start][c].Span == 0 {
			start--
		}
		if start < 0 {
			continue
		}
		open := t.Cells[start][c]
		t.Cells[start][c].Span = row - start
		open.Span = start + open.Span - row
		t.Cells[row][c] = open
	}
}

// tableRows draws the rows in [start, end); no merged run crosses either bound.
func (r *priceListRenderer) tableRows(table *plTable, start, end int) {
	y := r.pdf.GetY()
	for i := start; i < end; i++ {
		x := plPageLeft
		for c, column := range table.Columns {
			if cell := table.Cells[i][c]; cell.Span > 0 {
				height := 0.0
				for _, rowHeight := range table.Heights[i : i+cell.Span] {
					height += rowHeight
				}
				r.drawCell(x, y, column.Width, height, cell, false)
			}
			x += column.Width
		}
		y += table.Heights[i]
	}
	r.pdf.SetXY(plPageLeft, y)
}

func (r *priceListRenderer) tableHeader(table *plTable) {
	r.pdf.SetFillColor(248, 248, 248)
	x, y := plPageLeft, r.pdf.GetY()
	for c, column := range table.Columns {
		r.drawCell(x, y, column.Width, table.HeaderHeight, table.Header[c], true)
		x += column.Width
	}
	r.pdf.SetXY(plPageLeft, y+table.HeaderHeight)
}

// drawCell draws one bordered cell of arbitrary height with its wrapped text vertically centered. Text is clipped to the lines that fit, which only bites where a page break left a reopened merge shorter than the run it came from.
func (r *priceListRenderer) drawCell(x, y, width, height float64, cell plCell, fill bool) {
	style := "D"
	if fill {
		style = "FD"
	}
	r.pdf.Rect(x, y, width, height, style)
	if len(cell.Lines) == 0 {
		return
	}

	r.pdf.SetFont("Helvetica", cell.Style, cell.Size)
	lines := cell.Lines
	if fits := plLinesThatFit(height); len(lines) > fits {
		lines = append([]string{}, lines[:fits]...)
		lines[fits-1] = r.clip(lines[fits-1], width)
	}

	top := y + (height-float64(len(lines))*plLineHeight)/2
	for i, line := range lines {
		r.pdf.SetXY(x, top+float64(i)*plLineHeight)
		r.pdf.CellFormat(width, plLineHeight, line, "", 0, cell.Align, false, 0, "")
	}
}

// clip trims text to the cell width and marks it as cut. It trims by byte because the text has already been encoded to the font's cp1252, where every character is one byte — trimming the runes of a cp1252 string would corrupt anything above ASCII.
func (r *priceListRenderer) clip(text string, width float64) string {
	padded := width - plCellMargin
	for len(text) > 0 {
		if r.pdf.GetStringWidth(text+"...") <= padded {
			return text + "..."
		}
		text = text[:len(text)-1]
	}
	return ""
}
