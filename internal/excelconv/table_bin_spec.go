package excelconv

import (
	"strings"

	"ctc/pkg/tablebin"
)

// BuildTableBinSpec 按与生成 Go 行结构相同的可见列顺序构造二进制列描述（不含主键 id，id 单独编码）。
func BuildTableBinSpec(schema *Schema, tableName string, exportTags []string) (idKind tablebin.IDKind, cols []tablebin.Column) {
	idKind = tablebinIDKind(schema.PrimaryKeyTypeForTable(tableName))
	fields := schema.Tables[tableName]
	vis := VisibleTableFields(fields, exportTags)
	for _, f := range vis {
		cols = append(cols, tablebin.Column{Key: f.Name, Kind: TableBinColumnKind(f, schema)})
	}
	return idKind, cols
}

// TableBinColumnKind 与 BuildTableBinSpec 中列类型一致，供生成加载代码使用。
func TableBinColumnKind(f Field, schema *Schema) tablebin.ColumnKind {
	return tablebinColumnKind(f, schema)
}

func tablebinIDKind(s string) tablebin.IDKind {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "int":
		return tablebin.IDInt
	case "string":
		return tablebin.IDString
	default:
		return tablebin.IDInt64
	}
}

func tablebinColumnKind(f Field, schema *Schema) tablebin.ColumnKind {
	base := strings.TrimSpace(f.Type)
	arr := f.ArraySplit != ""

	if arr {
		switch base {
		case "string":
			return tablebin.KindSliceString
		case "int":
			return tablebin.KindSliceInt
		case "int64":
			return tablebin.KindSliceInt64
		case "float64":
			return tablebin.KindSliceFloat64
		default:
			if schema.Enums[base] != nil {
				return tablebin.KindSliceEnumInt32
			}
			if schema.Structs[base] != nil {
				return tablebin.KindSliceStructJSON
			}
			return tablebin.KindSliceString
		}
	}

	switch base {
	case "string":
		return tablebin.KindString
	case "int":
		return tablebin.KindInt
	case "int64":
		return tablebin.KindInt64
	case "float64":
		return tablebin.KindFloat64
	default:
		if schema.Enums[base] != nil {
			return tablebin.KindEnumInt32
		}
		if schema.Structs[base] != nil {
			return tablebin.KindStructJSON
		}
		return tablebin.KindString
	}
}
