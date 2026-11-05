package ports

import "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"

// WorkbookWriter saves tabs of rows as a spreadsheet file.
type WorkbookWriter interface {
	// Write creates (or replaces) the workbook at path, one worksheet per tab
	Write(path string, tabs []domain.SheetTab) error
}
