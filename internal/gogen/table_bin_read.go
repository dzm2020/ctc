package gogen

import (
	"fmt"
	"strings"

	"ctc/internal/excelconv"
	"ctc/pkg/tablebin"
)

// binLoadAssignLines 生成 tablebin 解码语句（与 excelconv.BuildTableBinSpec / EncodeFile 列顺序一致）。
// recv 为接收体名，如 "row"。
func binLoadAssignLines(f excelconv.Field, schema *excelconv.Schema, recv, priv, goType string) []string {
	k := excelconv.TableBinColumnKind(f, schema)
	switch k {
	case tablebin.KindInt:
		return []string{
			fmt.Sprintf("\t\t\t%s.%s, err = dec.ReadInt()\n\t\t\tif err != nil {\n\t\t\t\treturn err\n\t\t\t}", recv, priv),
		}
	case tablebin.KindInt64:
		return []string{
			fmt.Sprintf("\t\t\t%s.%s, err = dec.ReadInt64Zigzag()\n\t\t\tif err != nil {\n\t\t\t\treturn err\n\t\t\t}", recv, priv),
		}
	case tablebin.KindFloat64:
		return []string{
			fmt.Sprintf("\t\t\t%s.%s, err = dec.ReadFloat64()\n\t\t\tif err != nil {\n\t\t\t\treturn err\n\t\t\t}", recv, priv),
		}
	case tablebin.KindString:
		return []string{
			fmt.Sprintf("\t\t\t%s.%s, err = dec.ReadString()\n\t\t\tif err != nil {\n\t\t\t\treturn err\n\t\t\t}", recv, priv),
		}
	case tablebin.KindEnumInt32:
		en := strings.TrimSpace(f.Type)
		return []string{
			fmt.Sprintf("\t\t\tvar _e int32\n\t\t\t_e, err = dec.ReadInt32Zigzag()\n\t\t\tif err != nil {\n\t\t\t\treturn err\n\t\t\t}\n\t\t\t%s.%s = %s(_e)", recv, priv, en),
		}
	case tablebin.KindSliceInt:
		return []string{
			fmt.Sprintf("\t\t\t%s.%s, err = dec.ReadIntSlice()\n\t\t\tif err != nil {\n\t\t\t\treturn err\n\t\t\t}", recv, priv),
		}
	case tablebin.KindSliceInt64:
		return []string{
			fmt.Sprintf("\t\t\t%s.%s, err = dec.ReadInt64Slice()\n\t\t\tif err != nil {\n\t\t\t\treturn err\n\t\t\t}", recv, priv),
		}
	case tablebin.KindSliceFloat64:
		return []string{
			fmt.Sprintf("\t\t\t%s.%s, err = dec.ReadFloat64Slice()\n\t\t\tif err != nil {\n\t\t\t\treturn err\n\t\t\t}", recv, priv),
		}
	case tablebin.KindSliceString:
		return []string{
			fmt.Sprintf("\t\t\t%s.%s, err = dec.ReadStringSlice()\n\t\t\tif err != nil {\n\t\t\t\treturn err\n\t\t\t}", recv, priv),
		}
	case tablebin.KindSliceEnumInt32:
		en := strings.TrimSpace(f.Type)
		return []string{
			fmt.Sprintf("\t\t\tvar _ev []int32\n\t\t\t_ev, err = dec.ReadInt32ZigzagSlice()\n\t\t\tif err != nil {\n\t\t\t\treturn err\n\t\t\t}\n\t\t\t%s.%s = make([]%s, len(_ev))\n\t\t\tfor _i := range _ev {\n\t\t\t\t%s.%s[_i] = %s(_ev[_i])\n\t\t\t}", recv, priv, en, recv, priv, en),
		}
	case tablebin.KindStructJSON:
		return []string{
			fmt.Sprintf("\t\t\tvar _bs string\n\t\t\t_bs, err = dec.ReadString()\n\t\t\tif err != nil {\n\t\t\t\treturn err\n\t\t\t}\n\t\t\tif err := json.Unmarshal([]byte(_bs), &%s.%s); err != nil {\n\t\t\t\treturn err\n\t\t\t}", recv, priv),
		}
	case tablebin.KindSliceStructJSON:
		st := strings.TrimPrefix(goType, "[]")
		return []string{
			fmt.Sprintf("\t\t\tvar _bss []string\n\t\t\t_bss, err = dec.ReadStringSlice()\n\t\t\tif err != nil {\n\t\t\t\treturn err\n\t\t\t}\n\t\t\t%s.%s = make([]%s, len(_bss))\n\t\t\tfor _i := range _bss {\n\t\t\t\tif err := json.Unmarshal([]byte(_bss[_i]), &%s.%s[_i]); err != nil {\n\t\t\t\t\treturn err\n\t\t\t\t}\n\t\t\t}", recv, priv, st, recv, priv),
		}
	default:
		return []string{
			fmt.Sprintf("\t\t\t%s.%s, err = dec.ReadString()\n\t\t\tif err != nil {\n\t\t\t\treturn err\n\t\t\t}", recv, priv),
		}
	}
}
