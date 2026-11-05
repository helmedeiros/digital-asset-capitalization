package infrastructure

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
)

func TestXlsxWorkbookWriter_Write(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "alloc.xlsx")
	tabs := []domain.SheetTab{
		{Name: "2026-10", Rows: [][]string{
			{"S0"}, {"issueKey", "hours", "Ann"}, {`="A-1"`, "1.50", "12.50%"},
			{}, {},
			{"S1"}, {"issueKey", "hours", "Ann"}, {`="A-2"`, "0.00", ""},
		}},
		{Name: "2026/11:x", Rows: [][]string{{"S2"}}},
	}

	require.NoError(t, NewXlsxWorkbookWriter().Write(path, tabs))

	f, err := excelize.OpenFile(path)
	require.NoError(t, err)
	defer f.Close()
	assert.Equal(t, []string{"2026-10", "2026-11-x"}, f.GetSheetList())

	rows, err := f.GetRows("2026-10")
	require.NoError(t, err)
	assert.Equal(t, "S0", rows[0][0])
	assert.Equal(t, "A-1", rows[2][0], "CSV text guard dropped")
	assert.Empty(t, rows[3], "blank gap rows")
	assert.Equal(t, "S1", rows[5][0])

	v, err := f.GetCellValue("2026-10", "C3", excelize.Options{RawCellValue: true})
	require.NoError(t, err)
	assert.Equal(t, "0.125", v, "percent stored as a fraction")
}

func TestXlsxWorkbookWriter_NothingToWrite(t *testing.T) {
	t.Parallel()

	assert.Error(t, NewXlsxWorkbookWriter().Write(filepath.Join(t.TempDir(), "x.xlsx"), nil))
}

func TestSheetName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "a-b-c", sheetName("a/b:c"))
	assert.Len(t, sheetName("0123456789012345678901234567890123456789"), 31)
	assert.Equal(t, "Sheet", sheetName(""))
}
