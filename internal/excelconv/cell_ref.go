package excelconv

import "github.com/xuri/excelize/v2"

// ExcelColumnLetter 将 0-based 列下标转为 Excel 列字母（A、B…AA）。
func ExcelColumnLetter(col0Based int) string {
	if col0Based < 0 {
		return "?"
	}
	name, err := excelize.ColumnNumberToName(col0Based + 1)
	if err != nil {
		return "?"
	}
	return name
}
