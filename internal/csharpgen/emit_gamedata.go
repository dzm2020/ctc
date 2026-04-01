package csharpgen

import "ctc/internal/excelconv"

func emitGameDataFile(ns string, schema *excelconv.Schema, binary bool) (string, error) {
	d := buildCSharpGameDataFileData(ns, schema, binary)
	return executeCSharpTemplate("csharp_gamedata", d)
}
