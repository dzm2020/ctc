package gogen

import (
	"fmt"
	"strings"

	"ctc/internal/excelconv"
)

// rowGroupKeyStrFuncName 分组字段含不可比较类型时，从行拼 string 键的函数名。
func rowGroupKeyStrFuncName(tname, groupKey string) string {
	return "rowGroupKeyStr_" + tname + "_" + privateFieldIdent(groupKey)
}

// rowIndexKeyStrFuncName 索引字段含不可比较类型时，从行拼 string 键的函数名。
func rowIndexKeyStrFuncName(tname, indexKey string) string {
	return "rowIndexKeyStr_" + tname + "_" + privateFieldIdent(indexKey)
}

// groupValueCtorName 分组键构造函数：<表名>_<分组标识>，形参与组内字段顺序、类型一致（导出函数）。
func groupValueCtorName(tname, groupKey string) string {
	return tname + "_" + privateFieldIdent(groupKey)
}

// tableFileGroupColKeyImports 全表扫描（如 loader 等需汇总 import 时使用）。
func tableFileGroupColKeyImports(schema *excelconv.Schema, exportTags []string) (needStrconv, needFmt bool) {
	for _, tname := range sortedTableKeys(schema.Tables) {
		s, f := tableFileGroupColKeyImportsForTable(schema, tname, exportTags)
		if s {
			needStrconv = true
		}
		if f {
			needFmt = true
		}
	}
	return needStrconv, needFmt
}

// tableFileGroupColKeyImportsForTable 单表生成文件时计算该行组 string 键所需的 strconv/fmt。
func tableFileGroupColKeyImportsForTable(schema *excelconv.Schema, tname string, exportTags []string) (needStrconv, needFmt bool) {
	visible := visibleTableFields(schema.Tables[tname], exportTags)
	for _, g := range excelconv.DistinctFieldGroups(visible) {
		gf := excelconv.FieldsInGroup(visible, g)
		if groupFieldsComparable(gf, schema) {
			continue
		}
		for _, f := range gf {
			_, impStrconv, impFmt := goGroupKeyPartExpr(f, schema, "r")
			if impStrconv {
				needStrconv = true
			}
			if impFmt {
				needFmt = true
			}
		}
	}
	for _, ix := range excelconv.DistinctFieldIndexes(visible) {
		gf := excelconv.FieldsInIndex(visible, ix)
		if groupFieldsComparable(gf, schema) {
			continue
		}
		for _, f := range gf {
			_, impStrconv, impFmt := goGroupKeyPartExpr(f, schema, "r")
			if impStrconv {
				needStrconv = true
			}
			if impFmt {
				needFmt = true
			}
		}
	}
	return needStrconv, needFmt
}

func anyTableHasGroupsOrIndexes(schema *excelconv.Schema, exportTags []string) bool {
	for _, tname := range sortedTableKeys(schema.Tables) {
		visible := visibleTableFields(schema.Tables[tname], exportTags)
		if len(excelconv.DistinctFieldGroups(visible)) > 0 || len(excelconv.DistinctFieldIndexes(visible)) > 0 {
			return true
		}
	}
	return false
}

func goGroupKeyPartExpr(fld excelconv.Field, schema *excelconv.Schema, rowVar string) (code string, impStrconv, impFmt bool) {
	priv := privateFieldIdent(fld.Name)
	path := rowVar + "." + priv
	got := goFieldTypeTable(fld, schema)
	switch got {
	case "string":
		return path, false, false
	case "int":
		return fmt.Sprintf("strconv.Itoa(%s)", path), true, false
	case "int64":
		return fmt.Sprintf("strconv.FormatInt(%s, 10)", path), true, false
	case "float64":
		return fmt.Sprintf("strconv.FormatFloat(%s, 'g', -1, 64)", path), true, false
	default:
		if strings.HasPrefix(got, "[]") {
			if got == "[]string" {
				return fmt.Sprintf("strings.Join(%s, \",\")", path), false, false
			}
			return fmt.Sprintf("fmt.Sprint(%s)", path), false, true
		}
		if schema != nil && schema.Enums[fld.Type] != nil {
			return fmt.Sprintf("strconv.FormatInt(int64(%s), 10)", path), true, false
		}
		if schema != nil && schema.Structs[fld.Type] != nil {
			return fmt.Sprintf("fmt.Sprint(%s)", path), false, true
		}
		return path, false, false
	}
}

// groupFieldsComparable 分组内字段可否作为 map 键（仅含可比较标量；含切片/结构体等则退回 string 键）。
func groupFieldsComparable(fields []excelconv.Field, schema *excelconv.Schema) bool {
	for _, fld := range fields {
		if fld.ArraySplit != "" {
			return false
		}
		got := goFieldTypeTable(fld, schema)
		if strings.HasPrefix(got, "[]") {
			return false
		}
		if schema != nil && schema.Structs[fld.Type] != nil {
			return false
		}
	}
	return true
}

