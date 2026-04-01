package excelconv

import "strings"

// VisibleStructFields 按导出标签与字段「筛选」交集过滤结构体成员（与生成 structs_gen 顺序一致）。
func VisibleStructFields(fields []StructField, exportTags []string) []StructField {
	var v []StructField
	for _, sf := range fields {
		if !FieldVisible(sf.Filter, exportTags) {
			continue
		}
		v = append(v, sf)
	}
	return v
}

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
