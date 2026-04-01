package excelconv

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ConvertWorkbook 将工作簿中除 @Type 外、且在 Schema 中有表定义的 sheet 转为 map[表名][主键]行数据。
func ConvertWorkbook(f *excelize.File, schema *Schema, exportTags []string) (map[string]map[string]map[string]interface{}, error) {
	out := make(map[string]map[string]map[string]interface{})
	for _, name := range f.GetSheetList() {
		if name == typeSheetName {
			continue
		}
		if _, ok := schema.Tables[name]; !ok {
			continue
		}
		m, err := convertDataSheet(f, name, schema, exportTags)
		if err != nil {
			return nil, fmt.Errorf("sheet %q: %w", name, err)
		}
		out[name] = m
	}
	return out, nil
}

func convertDataSheet(f *excelize.File, sheet string, schema *Schema, exportTags []string) (map[string]map[string]interface{}, error) {
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return make(map[string]map[string]interface{}), nil
	}
	header := rows[0]
	if len(header) == 0 || strings.TrimSpace(header[0]) != "ArrayDict" {
		return nil, fmt.Errorf("表 %q 第 1 行 列 %s(%q): 第一列必须是 ArrayDict", sheet, ExcelColumnLetter(0), strings.TrimSpace(header[0]))
	}

	fields := schema.Tables[sheet]
	visible := VisibleTableFields(fields, exportTags)
	fieldByName := make(map[string]Field, len(fields))
	for _, fld := range fields {
		fieldByName[fld.Name] = fld
	}
	indexSeen := make(map[string]map[string]indexSeenEntry)
	for _, ix := range DistinctFieldIndexes(visible) {
		indexSeen[ix] = make(map[string]indexSeenEntry)
	}

	colNames := make([]string, len(header))
	skipCol := make([]bool, len(header))
	for i := range header {
		h := strings.TrimSpace(header[i])
		colNames[i] = h
		if strings.HasPrefix(h, "#") {
			skipCol[i] = true
		}
	}

	result := make(map[string]map[string]interface{})
	idFirstRow := make(map[string]int)
	for ridx := 1; ridx < len(rows); ridx++ {
		row := rows[ridx]
		if len(row) == 0 {
			continue
		}
		if k := strings.TrimSpace(firstCol(row)); k == "" || strings.HasPrefix(k, "#") {
			continue
		}

		key := strings.TrimSpace(row[0])
		if prevRow, exists := idFirstRow[key]; exists {
			col0Name := colNames[0]
			if col0Name == "" {
				col0Name = RowJSONIDKey
			}
			return nil, fmt.Errorf("表 %q 第 %d 行 列 %s(%q): 主键 ID 值 %q 与第 %d 行重复（主索引唯一）", sheet, ridx+1, ExcelColumnLetter(0), col0Name, key, prevRow)
		}

		idType := schema.PrimaryKeyTypeForTable(sheet)
		idVal, err := ParsePrimaryKeyCell(key, idType)
		if err != nil {
			col0Name := colNames[0]
			if col0Name == "" {
				col0Name = RowJSONIDKey
			}
			return nil, fmt.Errorf("表 %q 第 %d 行 列 %s(%q): %w", sheet, ridx+1, ExcelColumnLetter(0), col0Name, err)
		}

		rec := make(map[string]interface{})
		rec[RowJSONIDKey] = idVal

		for c := 1; c < len(colNames); c++ {
			if c >= len(skipCol) || skipCol[c] {
				continue
			}
			name := colNames[c]
			if name == "" || strings.HasPrefix(name, "#") {
				continue
			}
			if strings.EqualFold(name, RowJSONIDKey) {
				continue
			}
			fld, ok := fieldByName[name]
			if !ok {
				continue
			}
			if !FieldVisible(fld.Filter, exportTags) {
				continue
			}
			cell := ""
			if c < len(row) {
				cell = row[c]
			}
			val, err := cellValue(fld, cell, schema)
			if err != nil {
				return nil, fmt.Errorf("表 %q 第 %d 行 列 %s(%q): %w", sheet, ridx+1, ExcelColumnLetter(c), name, err)
			}
			// JSON 始终扁平：字段名即列名，不参与 @Type「分组/索引」嵌套。
			rec[fld.Name] = val
		}
		if len(indexSeen) > 0 {
			if err := addRecordToIndexSeen(rec, visible, schema, indexSeen, sheet, ridx+1, key); err != nil {
				return nil, err
			}
		}
		idFirstRow[key] = ridx + 1
		result[key] = rec
	}
	return result, nil
}

func firstCol(row []string) string {
	if len(row) == 0 {
		return ""
	}
	return row[0]
}

func cellValue(fld Field, raw string, schema *Schema) (interface{}, error) {
	raw = strings.TrimSpace(raw)
	if fld.ArraySplit != "" {
		parts := strings.Split(raw, fld.ArraySplit)
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		if fld.Type == "string" {
			return parts, nil
		}
		out := make([]interface{}, 0, len(parts))
		for _, p := range parts {
			if p == "" {
				out = append(out, zeroForField(fld, schema))
				continue
			}
			v, err := parseScalar(fld.Type, p, schema)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	}
	if raw == "" && fld.Default != "" {
		raw = fld.Default
	}
	if raw == "" {
		return zeroForField(fld, schema), nil
	}
	return parseScalar(fld.Type, raw, schema)
}

func zeroForField(fld Field, schema *Schema) interface{} {
	if fld.ArraySplit != "" {
		switch fld.Type {
		case "string":
			return []string{}
		default:
			// int / int64 / float64 / 枚举 / 自定义结构：JSON 均序列化为数组
			return []interface{}{}
		}
	}
	switch fld.Type {
	case "string":
		return ""
	case "int":
		return 0
	case "int64":
		return int64(0)
	case "float64":
		return float64(0)
	default:
		if schema != nil && schema.Enums[fld.Type] != nil {
			return 0
		}
		if schema != nil && schema.Structs[fld.Type] != nil {
			return map[string]interface{}{}
		}
		return ""
	}
}

func parseScalar(typeName, raw string, schema *Schema) (interface{}, error) {
	switch typeName {
	case "string":
		return raw, nil
	case "int":
		return parseIntDefault(raw, 0)
	case "int64":
		s := strings.TrimSpace(raw)
		if s == "" {
			return int64(0), nil
		}
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			f, ferr := strconv.ParseFloat(s, 64)
			if ferr != nil {
				return nil, err
			}
			return int64(f), nil
		}
		return v, nil
	case "float64":
		s := strings.TrimSpace(raw)
		if s == "" {
			return float64(0), nil
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, err
		}
		return f, nil
	default:
		if schema.Enums[typeName] != nil {
			if m, ok := schema.EnumValue[typeName]; ok {
				if iv, ok2 := m[raw]; ok2 {
					return iv, nil
				}
			}
			v, err := parseIntDefault(raw, 0)
			if err == nil {
				return v, nil
			}
			return nil, fmt.Errorf("枚举 %s 无成员 %q", typeName, raw)
		}
		if _, ok := schema.Structs[typeName]; ok {
			return parseStructCell(typeName, raw, schema)
		}
		return raw, nil
	}
}
