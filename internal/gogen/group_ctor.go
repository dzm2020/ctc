package gogen

import (
	"strings"

	"ctc/internal/excelconv"
)

// buildGroupParamAndArgLists 生成分组构造函数/查询方法的参数列表与调用实参列表。
func buildGroupParamAndArgLists(gf []excelconv.Field, schema *excelconv.Schema) (paramList, argList string) {
	var ps, as []string
	for _, f := range gf {
		priv := privateFieldIdent(f.Name)
		got := goFieldTypeTable(f, schema)
		ps = append(ps, priv+" "+got)
		as = append(as, priv)
	}
	return strings.Join(ps, ", "), strings.Join(as, ", ")
}

func buildRowKeyCall(ctorName string, gf []excelconv.Field) string {
	parts := make([]string, 0, len(gf))
	for _, f := range gf {
		parts = append(parts, "row."+privateFieldIdent(f.Name))
	}
	return ctorName + "(" + strings.Join(parts, ", ") + ")"
}
