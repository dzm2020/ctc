package csharpgen

import (
	"ctc/internal/excelconv"
)

func emitTableFile(ns, tableName string, schema *excelconv.Schema, exportTags []string, binary bool) (string, error) {
	d := buildCSharpTableFileData(ns, tableName, schema, exportTags, binary)
	return executeCSharpTemplate("csharp_table", d)
}
