package excelconv

import "strings"

// VisibleTableFields 按导出标签与字段「筛选」交集过滤表字段（不含主键列伪字段）。
func VisibleTableFields(fields []Field, exportTags []string) []Field {
	var v []Field
	for _, fld := range fields {
		if !FieldVisible(fld.Filter, exportTags) {
			continue
		}
		if strings.EqualFold(fld.Name, RowJSONIDKey) {
			continue
		}
		v = append(v, fld)
	}
	return v
}
