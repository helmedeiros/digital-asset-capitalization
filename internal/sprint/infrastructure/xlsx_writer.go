package infrastructure

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
)

// XlsxWorkbookWriter writes allocation tabs as an .xlsx workbook.
type XlsxWorkbookWriter struct{}

// NewXlsxWorkbookWriter creates an XlsxWorkbookWriter.
func NewXlsxWorkbookWriter() *XlsxWorkbookWriter {
	return &XlsxWorkbookWriter{}
}

// maxSheetNameLen is Excel's limit for worksheet names.
const maxSheetNameLen = 31

// Write creates the workbook at path with one worksheet per tab. Single-cell
// rows (the sprint titles) and the header row under them are bold, percentages
// become real percent cells and plain numbers real numeric cells.
func (w *XlsxWorkbookWriter) Write(path string, tabs []domain.SheetTab) error {
	if len(tabs) == 0 {
		return fmt.Errorf("nothing to write: no sprint allocations")
	}
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	bold, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return fmt.Errorf("failed to create style: %w", err)
	}
	title, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 13}})
	if err != nil {
		return fmt.Errorf("failed to create style: %w", err)
	}
	percent, err := f.NewStyle(&excelize.Style{NumFmt: 10})
	if err != nil {
		return fmt.Errorf("failed to create style: %w", err)
	}

	for i, tab := range tabs {
		name := sheetName(tab.Name)
		if i == 0 {
			if err := f.SetSheetName(f.GetSheetName(0), name); err != nil {
				return fmt.Errorf("failed to name sheet %q: %w", name, err)
			}
		} else if _, err := f.NewSheet(name); err != nil {
			return fmt.Errorf("failed to add sheet %q: %w", name, err)
		}
		if err := writeTab(f, name, tab.Rows, title, bold, percent); err != nil {
			return err
		}
	}
	if err := f.SaveAs(path); err != nil {
		return fmt.Errorf("failed to save workbook: %w", err)
	}
	return nil
}

func writeTab(f *excelize.File, sheet string, rows [][]string, title, bold, percent int) error {
	widths := make(map[int]float64)
	afterTitle := false
	for r, row := range rows {
		isTitle := len(row) == 1 && row[0] != ""
		for c, raw := range row {
			cell, err := excelize.CoordinatesToCellName(c+1, r+1)
			if err != nil {
				return fmt.Errorf("bad cell position: %w", err)
			}
			value, pct := cellValue(raw)
			if err := f.SetCellValue(sheet, cell, value); err != nil {
				return fmt.Errorf("failed to set %s!%s: %w", sheet, cell, err)
			}
			switch {
			case isTitle:
				_ = f.SetCellStyle(sheet, cell, cell, title)
			case afterTitle:
				_ = f.SetCellStyle(sheet, cell, cell, bold)
			case pct:
				_ = f.SetCellStyle(sheet, cell, cell, percent)
			}
			if !isTitle {
				widths[c] = max(widths[c], float64(len(raw))+2)
			}
		}
		afterTitle = isTitle
	}
	for c, width := range widths {
		col, err := excelize.ColumnNumberToName(c + 1)
		if err != nil {
			continue
		}
		_ = f.SetColWidth(sheet, col, col, min(width, 60))
	}
	return nil
}

// cellValue converts a CSV field into a typed cell value: percentages become
// fractions flagged for percent formatting, numbers become floats, and the
// ="KEY" text guard used for CSV in Excel is dropped.
func cellValue(raw string) (value any, percent bool) {
	s := strings.TrimSpace(raw)
	if strings.HasPrefix(s, `="`) && strings.HasSuffix(s, `"`) {
		return s[2 : len(s)-1], false
	}
	if strings.HasSuffix(s, "%") {
		if n, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64); err == nil {
			return n / 100, true
		}
	}
	if n, err := strconv.ParseFloat(s, 64); err == nil && s != "" {
		return n, false
	}
	return raw, false
}

// sheetName makes a tab name valid for Excel: no []:*?/\ and at most 31 chars.
func sheetName(name string) string {
	name = strings.Map(func(r rune) rune {
		if strings.ContainsRune(`[]:*?/\`, r) {
			return '-'
		}
		return r
	}, name)
	if len(name) > maxSheetNameLen {
		name = name[:maxSheetNameLen]
	}
	if name == "" {
		name = "Sheet"
	}
	return name
}
