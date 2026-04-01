package csharpgen

import "ctc/internal/excelconv"

func emitStructsFile(ns string, schema *excelconv.Schema, exportTags []string, binary bool) (string, error) {
	d := buildCSharpStructsFileData(ns, schema, exportTags, binary)
	return executeCSharpTemplate("csharp_structs", d)
}
