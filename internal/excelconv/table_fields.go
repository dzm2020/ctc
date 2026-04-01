package excelconv

import "strings"

// VisibleTableFields 按导出目标过滤表字段（不含主键列伪字段）。
func VisibleTableFields(fields []Field, target ExportTarget) []Field {
	var v []Field
	for _, fld := range fields {
		if !FieldVisible(fld.Filter, target) {
			continue
		}
		if strings.EqualFold(fld.Name, RowJSONIDKey) {
			continue
		}
		v = append(v, fld)
	}
	return v
}
