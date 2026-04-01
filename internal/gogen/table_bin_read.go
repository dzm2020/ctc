package gogen

import (
	"fmt"
	"strings"

	"ctc/internal/excelconv"
	"ctc/pkg/tablebin"
)

// binLoadAssignLines 生成 tablebin 解码语句（与 excelconv.BuildTableBinSpec / EncodeFile 列顺序一致）。
// recv 为接收体名，如 "row"；priv 为该表字段的私有 Go 名。
func binLoadAssignLines(f excelconv.Field, schema *excelconv.Schema, exportTags []string, recv, priv, goType string) []string {
	bt := strings.TrimSpace(f.Type)
	if f.ArraySplit == "" && schema.Structs[bt] != nil {
		return binLoadStructFields(bt, schema, exportTags, recv+"."+priv, "\t\t\t")
	}
	if f.ArraySplit != "" && schema.Structs[bt] != nil {
		return binLoadSliceOfStruct(recv+"."+priv, bt, goType, schema, exportTags, "\t\t\t")
	}

	k := excelconv.TableBinColumnKind(f, schema)
	switch k {
	case tablebin.KindInt:
		return []string{
			fmt.Sprintf("\t\t\t%s.%s, err = dec.ReadInt()\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}", recv, priv),
		}
	case tablebin.KindInt64:
		return []string{
			fmt.Sprintf("\t\t\t%s.%s, err = dec.ReadInt64Zigzag()\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}", recv, priv),
		}
	case tablebin.KindFloat64:
		return []string{
			fmt.Sprintf("\t\t\t%s.%s, err = dec.ReadFloat64()\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}", recv, priv),
		}
	case tablebin.KindString:
		return []string{
			fmt.Sprintf("\t\t\t%s.%s, err = dec.ReadString()\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}", recv, priv),
		}
	case tablebin.KindEnumInt32:
		en := strings.TrimSpace(f.Type)
		return []string{
			fmt.Sprintf("\t\t\tvar _e int32\n\t\t\t_e, err = dec.ReadInt32Zigzag()\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\t\t\t%s.%s = %s(_e)", recv, priv, en),
		}
	case tablebin.KindSliceInt:
		return []string{
			fmt.Sprintf("\t\t\t%s.%s, err = dec.ReadIntSlice()\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}", recv, priv),
		}
	case tablebin.KindSliceInt64:
		return []string{
			fmt.Sprintf("\t\t\t%s.%s, err = dec.ReadInt64Slice()\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}", recv, priv),
		}
	case tablebin.KindSliceFloat64:
		return []string{
			fmt.Sprintf("\t\t\t%s.%s, err = dec.ReadFloat64Slice()\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}", recv, priv),
		}
	case tablebin.KindSliceString:
		return []string{
			fmt.Sprintf("\t\t\t%s.%s, err = dec.ReadStringSlice()\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}", recv, priv),
		}
	case tablebin.KindSliceEnumInt32:
		en := strings.TrimSpace(f.Type)
		return []string{
			fmt.Sprintf("\t\t\tvar _ev []int32\n\t\t\t_ev, err = dec.ReadInt32ZigzagSlice()\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\t\t\t%s.%s = make([]%s, len(_ev))\n\t\t\tfor _i := range _ev {\n\t\t\t\t%s.%s[_i] = %s(_ev[_i])\n\t\t\t}", recv, priv, en, recv, priv, en),
		}
	case tablebin.KindSliceStruct:
		elem := strings.TrimSpace(strings.TrimPrefix(goType, "[]"))
		return binLoadSliceOfStruct(recv+"."+priv, elem, goType, schema, exportTags, "\t\t\t")
	default:
		return []string{
			fmt.Sprintf("\t\t\t%s.%s, err = dec.ReadString()\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}", recv, priv),
		}
	}
}

func binLoadSliceOfStruct(assignPrefix, elemTypeName, goSliceType string, schema *excelconv.Schema, exportTags []string, indent string) []string {
	tab := indent + "\t"
	lines := []string{
		fmt.Sprintf("%svar _nl int", indent),
		fmt.Sprintf("%s_nl, err = dec.ReadSliceLen()", indent),
		fmt.Sprintf("%sif err != nil {", indent),
		fmt.Sprintf("%s\treturn nil, err", indent),
		fmt.Sprintf("%s}", indent),
		fmt.Sprintf("%s%s = make(%s, _nl)", indent, assignPrefix, goSliceType),
		fmt.Sprintf("%sfor _si := 0; _si < _nl; _si++ {", indent),
	}
	lines = append(lines, binLoadStructFields(elemTypeName, schema, exportTags, assignPrefix+"[_si]", tab)...)
	lines = append(lines, fmt.Sprintf("%s}", indent))
	return lines
}

func binLoadStructFields(typeName string, schema *excelconv.Schema, exportTags []string, assignPrefix, indent string) []string {
	vis := excelconv.VisibleStructFields(schema.Structs[typeName], exportTags)
	var lines []string
	for _, sf := range vis {
		fp := privateFieldIdent(sf.Name)
		sub := assignPrefix + "." + fp
		bt := strings.TrimSpace(sf.Type)
		if sf.ArraySplit != "" {
			if schema.Structs[bt] != nil {
				goSl := goFieldTypeStruct(sf, schema)
				lines = append(lines, binLoadSliceOfStruct(sub, bt, goSl, schema, exportTags, indent)...)
				continue
			}
			lines = append(lines, binLoadScalarArrayLines(indent, sub, bt, schema)...)
			continue
		}
		if schema.Structs[bt] != nil {
			lines = append(lines, binLoadStructFields(bt, schema, exportTags, sub, indent)...)
			continue
		}
		lines = append(lines, binLoadScalarLeafLines(indent, sub, bt, schema)...)
	}
	return lines
}

func binLoadScalarLeafLines(indent, assignDest, baseType string, schema *excelconv.Schema) []string {
	bt := strings.TrimSpace(baseType)
	switch bt {
	case "int":
		return []string{
			fmt.Sprintf("%s%s, err = dec.ReadInt()", indent, assignDest),
			fmt.Sprintf("%sif err != nil {", indent),
			fmt.Sprintf("%s\treturn nil, err", indent),
			fmt.Sprintf("%s}", indent),
		}
	case "int64":
		return []string{
			fmt.Sprintf("%s%s, err = dec.ReadInt64Zigzag()", indent, assignDest),
			fmt.Sprintf("%sif err != nil {", indent),
			fmt.Sprintf("%s\treturn nil, err", indent),
			fmt.Sprintf("%s}", indent),
		}
	case "float64":
		return []string{
			fmt.Sprintf("%s%s, err = dec.ReadFloat64()", indent, assignDest),
			fmt.Sprintf("%sif err != nil {", indent),
			fmt.Sprintf("%s\treturn nil, err", indent),
			fmt.Sprintf("%s}", indent),
		}
	case "string":
		return []string{
			fmt.Sprintf("%s%s, err = dec.ReadString()", indent, assignDest),
			fmt.Sprintf("%sif err != nil {", indent),
			fmt.Sprintf("%s\treturn nil, err", indent),
			fmt.Sprintf("%s}", indent),
		}
	default:
		if schema.Enums[bt] != nil {
			return []string{
				fmt.Sprintf("%svar _e int32", indent),
				fmt.Sprintf("%s_e, err = dec.ReadInt32Zigzag()", indent),
				fmt.Sprintf("%sif err != nil {", indent),
				fmt.Sprintf("%s\treturn nil, err", indent),
				fmt.Sprintf("%s}", indent),
				fmt.Sprintf("%s%s = %s(_e)", indent, assignDest, bt),
			}
		}
		return []string{
			fmt.Sprintf("%s%s, err = dec.ReadString()", indent, assignDest),
			fmt.Sprintf("%sif err != nil {", indent),
			fmt.Sprintf("%s\treturn nil, err", indent),
			fmt.Sprintf("%s}", indent),
		}
	}
}

func binLoadScalarArrayLines(indent, assignDest, elemBase string, schema *excelconv.Schema) []string {
	bt := strings.TrimSpace(elemBase)
	switch bt {
	case "int":
		return []string{
			fmt.Sprintf("%s%s, err = dec.ReadIntSlice()", indent, assignDest),
			fmt.Sprintf("%sif err != nil {", indent),
			fmt.Sprintf("%s\treturn nil, err", indent),
			fmt.Sprintf("%s}", indent),
		}
	case "int64":
		return []string{
			fmt.Sprintf("%s%s, err = dec.ReadInt64Slice()", indent, assignDest),
			fmt.Sprintf("%sif err != nil {", indent),
			fmt.Sprintf("%s\treturn nil, err", indent),
			fmt.Sprintf("%s}", indent),
		}
	case "float64":
		return []string{
			fmt.Sprintf("%s%s, err = dec.ReadFloat64Slice()", indent, assignDest),
			fmt.Sprintf("%sif err != nil {", indent),
			fmt.Sprintf("%s\treturn nil, err", indent),
			fmt.Sprintf("%s}", indent),
		}
	case "string":
		return []string{
			fmt.Sprintf("%s%s, err = dec.ReadStringSlice()", indent, assignDest),
			fmt.Sprintf("%sif err != nil {", indent),
			fmt.Sprintf("%s\treturn nil, err", indent),
			fmt.Sprintf("%s}", indent),
		}
	default:
		if schema.Enums[bt] != nil {
			en := bt
			return []string{
				fmt.Sprintf("%svar _ev []int32", indent),
				fmt.Sprintf("%s_ev, err = dec.ReadInt32ZigzagSlice()", indent),
				fmt.Sprintf("%sif err != nil {", indent),
				fmt.Sprintf("%s\treturn nil, err", indent),
				fmt.Sprintf("%s}", indent),
				fmt.Sprintf("%s%s = make([]%s, len(_ev))", indent, assignDest, en),
				fmt.Sprintf("%sfor _i := range _ev {", indent),
				fmt.Sprintf("%s\t%s[_i] = %s(_ev[_i])", indent, assignDest, en),
				fmt.Sprintf("%s}", indent),
			}
		}
		return []string{
			fmt.Sprintf("%s%s, err = dec.ReadStringSlice()", indent, assignDest),
			fmt.Sprintf("%sif err != nil {", indent),
			fmt.Sprintf("%s\treturn nil, err", indent),
			fmt.Sprintf("%s}", indent),
		}
	}
}
