package csharpgen

import (
	"strings"

	"ctc/internal/excelconv"
)

func csharpTypeTable(f excelconv.Field, schema *excelconv.Schema) string {
	base := csharpBaseType(f.Type, schema)
	if f.ArraySplit != "" {
		return base + "[]"
	}
	return base
}

func csharpTypeStruct(sf excelconv.StructField, schema *excelconv.Schema) string {
	base := csharpBaseType(sf.Type, schema)
	if sf.ArraySplit != "" {
		return base + "[]"
	}
	return base
}

func csharpBaseType(typeName string, schema *excelconv.Schema) string {
	switch strings.TrimSpace(typeName) {
	case "string":
		return "string"
	case "int":
		return "int"
	case "int64":
		return "long"
	case "float64":
		return "double"
	default:
		if schema.Structs[typeName] != nil {
			return csharpPublicName(typeName)
		}
		if schema.Enums[typeName] != nil {
			return csharpPublicName(typeName)
		}
		return "string"
	}
}

func csharpPrimaryKeyType(schema *excelconv.Schema, table string) string {
	switch schema.PrimaryKeyTypeForTable(table) {
	case "int":
		return "int"
	case "string":
		return "string"
	default:
		return "long"
	}
}

func groupFieldsComparable(fields []excelconv.Field, schema *excelconv.Schema) bool {
	for _, fld := range fields {
		if fld.ArraySplit != "" {
			return false
		}
		got := csharpTypeTable(fld, schema)
		if strings.HasSuffix(got, "[]") {
			return false
		}
		if schema.Structs[strings.TrimSpace(fld.Type)] != nil {
			return false
		}
	}
	return true
}
