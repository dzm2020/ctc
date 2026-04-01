package excelconv

import (
	"fmt"
	"strings"

	"ctc/pkg/tablebin"
)

// BuildTableBinSpec 按与生成 Go 行结构相同的可见列顺序构造二进制列描述（不含主键 id，id 单独编码）。
// 自定义结构体列在 bin 中按字段顺序展开写入，不再使用 JSON 字符串。
func BuildTableBinSpec(schema *Schema, tableName string, exportTags []string) (idKind tablebin.IDKind, cols []tablebin.Column, err error) {
	idKind = tablebinIDKind(schema.PrimaryKeyTypeForTable(tableName))
	fields := schema.Tables[tableName]
	vis := VisibleTableFields(fields, exportTags)
	for _, f := range vis {
		bt := strings.TrimSpace(f.Type)
		if f.ArraySplit == "" && schema.Structs[bt] != nil {
			sub, ferr := flattenStructBinColumns(f.Name, bt, "", schema, exportTags)
			if ferr != nil {
				return idKind, nil, fmt.Errorf("表 %q 列 %q: %w", tableName, f.Name, ferr)
			}
			cols = append(cols, sub...)
			continue
		}
		if f.ArraySplit != "" && schema.Structs[bt] != nil {
			el, ferr := structSliceElemFlatten(bt, "", schema, exportTags)
			if ferr != nil {
				return idKind, nil, fmt.Errorf("表 %q 列 %q: %w", tableName, f.Name, ferr)
			}
			cols = append(cols, tablebin.Column{Key: f.Name, Kind: tablebin.KindSliceStruct, SliceElem: el})
			continue
		}
		cols = append(cols, tablebin.Column{Key: f.Name, Kind: tablebinColumnKind(f, schema)})
	}
	return idKind, cols, nil
}

// TableBinColumnKind 与 BuildTableBinSpec 中「非展开」列类型一致，供生成加载代码使用。
func TableBinColumnKind(f Field, schema *Schema) tablebin.ColumnKind {
	return tablebinColumnKind(f, schema)
}

func joinBinSubPath(prefix, name string) string {
	name = strings.TrimSpace(name)
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

func tableBinColumnKindForLeafType(baseType string, asArray bool, schema *Schema) (tablebin.ColumnKind, error) {
	base := strings.TrimSpace(baseType)
	if asArray {
		switch base {
		case "string":
			return tablebin.KindSliceString, nil
		case "int":
			return tablebin.KindSliceInt, nil
		case "int64":
			return tablebin.KindSliceInt64, nil
		case "float64":
			return tablebin.KindSliceFloat64, nil
		default:
			if schema.Enums[base] != nil {
				return tablebin.KindSliceEnumInt32, nil
			}
			return 0, fmt.Errorf("不支持的数组元素类型 %q", base)
		}
	}
	switch base {
	case "string":
		return tablebin.KindString, nil
	case "int":
		return tablebin.KindInt, nil
	case "int64":
		return tablebin.KindInt64, nil
	case "float64":
		return tablebin.KindFloat64, nil
	default:
		if schema.Enums[base] != nil {
			return tablebin.KindEnumInt32, nil
		}
		return 0, fmt.Errorf("不支持的标量类型 %q", base)
	}
}

func structSliceElemFlatten(typeName, prefix string, schema *Schema, exportTags []string) ([]tablebin.SliceElemField, error) {
	sfs := VisibleStructFields(schema.Structs[typeName], exportTags)
	var out []tablebin.SliceElemField
	for _, sf := range sfs {
		p := joinBinSubPath(prefix, sf.Name)
		bt := strings.TrimSpace(sf.Type)
		if sf.ArraySplit != "" {
			if schema.Structs[bt] != nil {
				return nil, fmt.Errorf("结构 %s 字段 %q: 暂不支持「结构体数组的元素」再含结构体数组/结构体切片", typeName, sf.Name)
			}
			k, err := tableBinColumnKindForLeafType(bt, true, schema)
			if err != nil {
				return nil, err
			}
			out = append(out, tablebin.SliceElemField{SubPath: p, Kind: k})
			continue
		}
		if schema.Structs[bt] != nil {
			inner, err := structSliceElemFlatten(bt, p, schema, exportTags)
			if err != nil {
				return nil, err
			}
			out = append(out, inner...)
			continue
		}
		if schema.Enums[bt] != nil {
			out = append(out, tablebin.SliceElemField{SubPath: p, Kind: tablebin.KindEnumInt32})
			continue
		}
		k, err := tableBinColumnKindForLeafType(bt, false, schema)
		if err != nil {
			return nil, err
		}
		out = append(out, tablebin.SliceElemField{SubPath: p, Kind: k})
	}
	return out, nil
}

func flattenStructBinColumns(rowKey, typeName, pathPrefix string, schema *Schema, exportTags []string) ([]tablebin.Column, error) {
	sfs := VisibleStructFields(schema.Structs[typeName], exportTags)
	var cols []tablebin.Column
	for _, sf := range sfs {
		p := joinBinSubPath(pathPrefix, sf.Name)
		bt := strings.TrimSpace(sf.Type)
		if sf.ArraySplit != "" {
			if schema.Structs[bt] != nil {
				elem, err := structSliceElemFlatten(bt, "", schema, exportTags)
				if err != nil {
					return nil, fmt.Errorf("结构 %s 字段 %q: %w", typeName, sf.Name, err)
				}
				cols = append(cols, tablebin.Column{Key: rowKey, SubPath: p, Kind: tablebin.KindSliceStruct, SliceElem: elem})
				continue
			}
			k, err := tableBinColumnKindForLeafType(bt, true, schema)
			if err != nil {
				return nil, err
			}
			cols = append(cols, tablebin.Column{Key: rowKey, SubPath: p, Kind: k})
			continue
		}
		if schema.Structs[bt] != nil {
			sub, err := flattenStructBinColumns(rowKey, bt, p, schema, exportTags)
			if err != nil {
				return nil, err
			}
			cols = append(cols, sub...)
			continue
		}
		if schema.Enums[bt] != nil {
			cols = append(cols, tablebin.Column{Key: rowKey, SubPath: p, Kind: tablebin.KindEnumInt32})
			continue
		}
		k, err := tableBinColumnKindForLeafType(bt, false, schema)
		if err != nil {
			return nil, err
		}
		cols = append(cols, tablebin.Column{Key: rowKey, SubPath: p, Kind: k})
	}
	return cols, nil
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
				return tablebin.KindSliceStruct
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
		return tablebin.KindString
	}
}
