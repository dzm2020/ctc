package csharpgen

import "ctc/internal/excelconv"

func emitEnumsFile(ns string, schema *excelconv.Schema) (string, error) {
	d := buildCSharpEnumsFileData(ns, schema)
	return executeCSharpTemplate("csharp_enums", d)
}
