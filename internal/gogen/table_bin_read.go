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
		return []string{
			fmt.Sprintf("\t\t\tv := &%s{}", bt),
			"\t\t\tif err = v.deserialize(dec); err != nil {",
			"\t\t\t\treturn nil, err",
			"\t\t\t}",
			fmt.Sprintf("\t\t\t%s.%s = v", recv, priv),
		}
	}
	if f.ArraySplit != "" && schema.Structs[bt] != nil {
		return binLoadSliceOfStructDeserialize(recv+"."+priv, bt, goType, schema, exportTags, "\t\t\t", true)
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
		elem := goSliceElemSchemaName(goType)
		return binLoadSliceOfStructDeserialize(recv+"."+priv, elem, goType, schema, exportTags, "\t\t\t", true)
	default:
		return []string{
			fmt.Sprintf("\t\t\t%s.%s, err = dec.ReadString()\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}", recv, priv),
		}
	}
}

// goSliceElemSchemaName 从 Go 切片类型（如 "[]*Foo"）得到 @Type 结构体名 "Foo"，供 bin 解码与 schema 查找。
func goSliceElemSchemaName(goSliceType string) string {
	s := strings.TrimSpace(strings.TrimPrefix(goSliceType, "[]"))
	return strings.TrimPrefix(s, "*")
}

// binLoadSliceOfStructDeserialize 解码 []*T 或 []T（元素为 @Type 结构体）：每元素调用 deserialize。
// retNilErr 为 true 时错误分支为 return nil, err（表行反序列化）；false 时为 return err（配置结构体 deserialize 内）。
func binLoadSliceOfStructDeserialize(assignPrefix, elemTypeName, goSliceType string, schema *excelconv.Schema, exportTags []string, indent string, retNilErr bool) []string {
	errRet := "return err"
	if retNilErr {
		errRet = "return nil, err"
	}
	lines := []string{
		fmt.Sprintf("%svar _nl int", indent),
		fmt.Sprintf("%s_nl, err = dec.ReadSliceLen()", indent),
		fmt.Sprintf("%sif err != nil {", indent),
		fmt.Sprintf("%s\t%s", indent, errRet),
		fmt.Sprintf("%s}", indent),
		fmt.Sprintf("%s%s = make(%s, _nl)", indent, assignPrefix, goSliceType),
		fmt.Sprintf("%sfor _si := 0; _si < _nl; _si++ {", indent),
		fmt.Sprintf("%s\tv := &%s{}", indent, elemTypeName),
		fmt.Sprintf("%s\tif err = v.deserialize(dec); err != nil {", indent),
		fmt.Sprintf("%s\t\t%s", indent, errRet),
		fmt.Sprintf("%s\t}", indent),
		fmt.Sprintf("%s\t%s[_si] = v", indent, assignPrefix),
		fmt.Sprintf("%s}", indent),
	}
	return lines
}

// structDeserializeFieldLines 生成 (s *T) deserialize 方法体中的语句（不含方法签名与最后的 return nil）。
func structDeserializeFieldLines(typeName string, schema *excelconv.Schema, exportTags []string, assignPrefix, indent string) []string {
	vis := excelconv.VisibleStructFields(schema.Structs[typeName], exportTags)
	var lines []string
	for _, sf := range vis {
		fp := privateFieldIdent(sf.Name)
		sub := assignPrefix + "." + fp
		bt := strings.TrimSpace(sf.Type)
		if sf.ArraySplit != "" {
			if schema.Structs[bt] != nil {
				goSl := goFieldTypeStruct(sf, schema)
				lines = append(lines, binLoadSliceOfStructDeserialize(sub, bt, goSl, schema, exportTags, indent, false)...)
				continue
			}
			lines = append(lines, binLoadScalarArrayLines(indent, sub, bt, schema, false)...)
			continue
		}
		if schema.Structs[bt] != nil {
			lines = append(lines, fmt.Sprintf("%s%s = &%s{}", indent, sub, bt))
			lines = append(lines,
				fmt.Sprintf("%sif err = %s.deserialize(dec); err != nil {", indent, sub),
				fmt.Sprintf("%s\treturn err", indent),
				fmt.Sprintf("%s}", indent),
			)
			continue
		}
		lines = append(lines, binLoadScalarLeafLines(indent, sub, bt, schema, false)...)
	}
	return lines
}

func binLoadScalarLeafLines(indent, assignDest, baseType string, schema *excelconv.Schema, retNilPair bool) []string {
	errRet := "\treturn err"
	if retNilPair {
		errRet = "\treturn nil, err"
	}
	bt := strings.TrimSpace(baseType)
	switch bt {
	case "int":
		return []string{
			fmt.Sprintf("%s%s, err = dec.ReadInt()", indent, assignDest),
			fmt.Sprintf("%sif err != nil {", indent),
			fmt.Sprintf("%s%s", indent, errRet),
			fmt.Sprintf("%s}", indent),
		}
	case "int64":
		return []string{
			fmt.Sprintf("%s%s, err = dec.ReadInt64Zigzag()", indent, assignDest),
			fmt.Sprintf("%sif err != nil {", indent),
			fmt.Sprintf("%s%s", indent, errRet),
			fmt.Sprintf("%s}", indent),
		}
	case "float64":
		return []string{
			fmt.Sprintf("%s%s, err = dec.ReadFloat64()", indent, assignDest),
			fmt.Sprintf("%sif err != nil {", indent),
			fmt.Sprintf("%s%s", indent, errRet),
			fmt.Sprintf("%s}", indent),
		}
	case "string":
		return []string{
			fmt.Sprintf("%s%s, err = dec.ReadString()", indent, assignDest),
			fmt.Sprintf("%sif err != nil {", indent),
			fmt.Sprintf("%s%s", indent, errRet),
			fmt.Sprintf("%s}", indent),
		}
	default:
		if schema.Enums[bt] != nil {
			return []string{
				fmt.Sprintf("%svar _e int32", indent),
				fmt.Sprintf("%s_e, err = dec.ReadInt32Zigzag()", indent),
				fmt.Sprintf("%sif err != nil {", indent),
				fmt.Sprintf("%s%s", indent, errRet),
				fmt.Sprintf("%s}", indent),
				fmt.Sprintf("%s%s = %s(_e)", indent, assignDest, bt),
			}
		}
		return []string{
			fmt.Sprintf("%s%s, err = dec.ReadString()", indent, assignDest),
			fmt.Sprintf("%sif err != nil {", indent),
			fmt.Sprintf("%s%s", indent, errRet),
			fmt.Sprintf("%s}", indent),
		}
	}
}

func binLoadScalarArrayLines(indent, assignDest, elemBase string, schema *excelconv.Schema, retNilPair bool) []string {
	errRet := "\treturn err"
	if retNilPair {
		errRet = "\treturn nil, err"
	}
	bt := strings.TrimSpace(elemBase)
	switch bt {
	case "int":
		return []string{
			fmt.Sprintf("%s%s, err = dec.ReadIntSlice()", indent, assignDest),
			fmt.Sprintf("%sif err != nil {", indent),
			fmt.Sprintf("%s%s", indent, errRet),
			fmt.Sprintf("%s}", indent),
		}
	case "int64":
		return []string{
			fmt.Sprintf("%s%s, err = dec.ReadInt64Slice()", indent, assignDest),
			fmt.Sprintf("%sif err != nil {", indent),
			fmt.Sprintf("%s%s", indent, errRet),
			fmt.Sprintf("%s}", indent),
		}
	case "float64":
		return []string{
			fmt.Sprintf("%s%s, err = dec.ReadFloat64Slice()", indent, assignDest),
			fmt.Sprintf("%sif err != nil {", indent),
			fmt.Sprintf("%s%s", indent, errRet),
			fmt.Sprintf("%s}", indent),
		}
	case "string":
		return []string{
			fmt.Sprintf("%s%s, err = dec.ReadStringSlice()", indent, assignDest),
			fmt.Sprintf("%sif err != nil {", indent),
			fmt.Sprintf("%s%s", indent, errRet),
			fmt.Sprintf("%s}", indent),
		}
	default:
		if schema.Enums[bt] != nil {
			en := bt
			return []string{
				fmt.Sprintf("%svar _ev []int32", indent),
				fmt.Sprintf("%s_ev, err = dec.ReadInt32ZigzagSlice()", indent),
				fmt.Sprintf("%sif err != nil {", indent),
				fmt.Sprintf("%s%s", indent, errRet),
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
			fmt.Sprintf("%s%s", indent, errRet),
			fmt.Sprintf("%s}", indent),
		}
	}
}
