package csharpgen

import (
	"fmt"
	"strings"

	"ctc/internal/excelconv"
	"ctc/pkg/tablebin"
)

// emitTableRowReadFrom 生成 WallpaperRow.ReadFrom(TableBinDecoder dec) 方法体（不含大括号）。
func emitTableRowReadFrom(tname string, schema *excelconv.Schema, vis []excelconv.Field, exportTags []string) string {
	var b strings.Builder
	pk := csharpPrimaryKeyType(schema, tname)
	switch pk {
	case "long":
		fmt.Fprintf(&b, "\t\tId = dec.ReadInt64Zigzag();\n")
	case "int":
		fmt.Fprintf(&b, "\t\tId = dec.ReadInt();\n")
	default:
		fmt.Fprintf(&b, "\t\tId = dec.ReadString();\n")
	}
	for _, f := range vis {
		prop := csharpSafeProp(f.Name)
		bt := strings.TrimSpace(f.Type)
		if f.ArraySplit == "" && schema.Structs[bt] != nil {
			tn := csharpPublicName(bt)
			fmt.Fprintf(&b, "\t\t%s = new %s();\n", prop, tn)
			fmt.Fprintf(&b, "\t\t%s.ReadFrom(dec);\n", prop)
			continue
		}
		if f.ArraySplit != "" && schema.Structs[bt] != nil {
			tn := csharpPublicName(bt)
			fmt.Fprintf(&b, "\t\t{\n\t\t\tint _nl = dec.ReadSliceLen();\n")
			fmt.Fprintf(&b, "\t\t\t%s = new %s[_nl];\n", prop, tn)
			fmt.Fprintf(&b, "\t\t\tfor (int _si = 0; _si < _nl; _si++) {\n")
			fmt.Fprintf(&b, "\t\t\t\t%s[_si] = new %s();\n", prop, tn)
			fmt.Fprintf(&b, "\t\t\t\t%s[_si].ReadFrom(dec);\n", prop)
			fmt.Fprintf(&b, "\t\t\t}\n\t\t}\n")
			continue
		}
		b.WriteString(emitScalarOrSliceFieldRead(f, schema, "this.", prop))
	}
	return b.String()
}

func emitScalarOrSliceFieldRead(f excelconv.Field, schema *excelconv.Schema, recv, prop string) string {
	k := excelconv.TableBinColumnKind(f, schema)
	en := strings.TrimSpace(f.Type)
	target := recv + prop
	switch k {
	case tablebin.KindInt:
		return fmt.Sprintf("\t\t%s = dec.ReadInt();\n", target)
	case tablebin.KindInt64:
		return fmt.Sprintf("\t\t%s = dec.ReadInt64Zigzag();\n", target)
	case tablebin.KindFloat64:
		return fmt.Sprintf("\t\t%s = dec.ReadFloat64();\n", target)
	case tablebin.KindString:
		return fmt.Sprintf("\t\t%s = dec.ReadString();\n", target)
	case tablebin.KindEnumInt32:
		return fmt.Sprintf("\t\t%s = (%s)dec.ReadInt32Zigzag();\n", target, csharpPublicName(en))
	case tablebin.KindSliceInt:
		return fmt.Sprintf("\t\t%s = dec.ReadIntSlice();\n", target)
	case tablebin.KindSliceInt64:
		return fmt.Sprintf("\t\t%s = dec.ReadInt64Slice();\n", target)
	case tablebin.KindSliceFloat64:
		return fmt.Sprintf("\t\t%s = dec.ReadFloat64Slice();\n", target)
	case tablebin.KindSliceString:
		return fmt.Sprintf("\t\t%s = dec.ReadStringSlice();\n", target)
	case tablebin.KindSliceEnumInt32:
		return fmt.Sprintf("\t\t{\n\t\t\tvar _ev = dec.ReadInt32ZigzagSliceAsIntArray();\n"+
			"\t\t\t%s = new %s[_ev.Length];\n"+
			"\t\t\tfor (int _i = 0; _i < _ev.Length; _i++) %s[_i] = (%s)_ev[_i];\n\t\t}\n",
			target, csharpPublicName(en), target, csharpPublicName(en))
	default:
		return fmt.Sprintf("\t\t%s = dec.ReadString();\n", target)
	}
}

// emitStructReadFromBody 配置结构体 ReadFrom 方法体。
func emitStructReadFromBody(typeName string, schema *excelconv.Schema, exportTags []string) string {
	var b strings.Builder
	for _, sf := range excelconv.VisibleStructFields(schema.Structs[typeName], exportTags) {
		prop := csharpSafeProp(sf.Name)
		bt := strings.TrimSpace(sf.Type)
		if sf.ArraySplit == "" && schema.Structs[bt] != nil {
			tn := csharpPublicName(bt)
			fmt.Fprintf(&b, "\t\t%s = new %s();\n", prop, tn)
			fmt.Fprintf(&b, "\t\t%s.ReadFrom(dec);\n", prop)
			continue
		}
		if sf.ArraySplit != "" && schema.Structs[bt] != nil {
			tn := csharpPublicName(bt)
			fmt.Fprintf(&b, "\t\t{\n\t\t\tint _nl = dec.ReadSliceLen();\n")
			fmt.Fprintf(&b, "\t\t\t%s = new %s[_nl];\n", prop, tn)
			fmt.Fprintf(&b, "\t\t\tfor (int _si = 0; _si < _nl; _si++) {\n")
			fmt.Fprintf(&b, "\t\t\t\t%s[_si] = new %s();\n", prop, tn)
			fmt.Fprintf(&b, "\t\t\t\t%s[_si].ReadFrom(dec);\n", prop)
			fmt.Fprintf(&b, "\t\t\t}\n\t\t}\n")
			continue
		}
		f := excelconv.Field{Type: sf.Type, ArraySplit: sf.ArraySplit}
		b.WriteString(emitScalarOrSliceFieldRead(f, schema, "this.", prop))
	}
	return b.String()
}
