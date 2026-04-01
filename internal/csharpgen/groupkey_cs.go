package csharpgen

import (
	"fmt"
	"strings"

	"ctc/internal/excelconv"
)

// csGroupKeyPart 生成 C# 表达式片段，与 Go goGroupKeyPartExpr 语义一致（用于 string 键）。
func csGroupKeyPart(fld excelconv.Field, schema *excelconv.Schema, rowVar string) string {
	prop := csharpSafeProp(fld.Name)
	path := rowVar + "." + prop
	got := csharpTypeTable(fld, schema)
	switch got {
	case "string":
		return path
	case "int":
		return path + ".ToString()"
	case "long":
		return path + ".ToString()"
	case "double":
		return path + ".ToString(System.Globalization.CultureInfo.InvariantCulture)"
	default:
		if strings.HasSuffix(got, "[]") {
			if got == "string[]" {
				return fmt.Sprintf("string.Join(\",\", %s ?? System.Array.Empty<string>())", path)
			}
			return fmt.Sprintf("System.Text.Json.JsonSerializer.Serialize(%s)", path)
		}
		if schema != nil && schema.Enums[strings.TrimSpace(fld.Type)] != nil {
			return fmt.Sprintf("((int)%s).ToString()", path)
		}
		if schema != nil && schema.Structs[strings.TrimSpace(fld.Type)] != nil {
			return fmt.Sprintf("System.Text.Json.JsonSerializer.Serialize(%s)", path)
		}
		return path
	}
}

func csGroupKeyJoinExpr(fields []excelconv.Field, schema *excelconv.Schema, rowVar string) string {
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		parts = append(parts, csGroupKeyPart(f, schema, rowVar))
	}
	return "string.Join(\"\\u001e\", new string[] { " + strings.Join(parts, ", ") + " })"
}
