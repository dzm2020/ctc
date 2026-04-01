package gogen

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"

	"ctc/internal/excelconv"
)

// WritePackage 根据 Schema 写出可编译的 Go 包：
//   - enums_gen.go、structs_gen.go
//   - 每张数据表一个 table_<slug>_gen.go（原 tables_gen.go 已废弃，会尝试删除）
// loader_gen.go 由 GenerateBundle 单独写出。
func WritePackage(dir, pkg string, schema *excelconv.Schema, exportTags []string, binaryData bool) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if pkg == "" {
		pkg = "gamedata"
	}

	var files []fileOut

	// 枚举
	var enames []string
	for n := range schema.Enums {
		enames = append(enames, n)
	}
	sort.Strings(enames)
	if len(enames) > 0 {
		src, err := renderEnumsFile(pkg, schema)
		if err != nil {
			return fmt.Errorf("enums template: %w", err)
		}
		files = append(files, fileOut{"enums_gen.go", src})
	}

	// 策划「结构」
	var snames []string
	for n := range schema.Structs {
		snames = append(snames, n)
	}
	sort.Strings(snames)
	if len(snames) > 0 {
		src, err := renderStructsFile(pkg, snames, schema, exportTags)
		if err != nil {
			return fmt.Errorf("structs template: %w", err)
		}
		files = append(files, fileOut{"structs_gen.go", src})
	}

	tkeys := sortedTableKeys(schema.Tables)
	if len(tkeys) > 0 {
		_ = os.Remove(filepath.Join(dir, "tables_gen.go"))
		slugUse := make(map[string]int)
		for _, tname := range tkeys {
			slug := privateFieldIdent(tname)
			if slug == "" {
				slug = "table"
			}
			n := slugUse[slug]
			slugUse[slug]++
			var fname string
			if n == 0 {
				fname = fmt.Sprintf("table_%s_gen.go", slug)
			} else {
				fname = fmt.Sprintf("table_%s_%d_gen.go", slug, n+1)
			}
			src, err := renderOneTableGoFile(pkg, tname, schema, exportTags, binaryData)
			if err != nil {
				return fmt.Errorf("table %q (%s): %w", tname, fname, err)
			}
			files = append(files, fileOut{fname, src})
		}
	}

	type formatted struct {
		name string
		data []byte
	}
	out := make([]formatted, 0, len(files))
	for _, fo := range files {
		src, err := format.Source([]byte(fo.content))
		if err != nil {
			return fmt.Errorf("format %s: %w\n%s", fo.name, err, fo.content)
		}
		out = append(out, formatted{fo.name, src})
	}
	for _, fo := range out {
		path := filepath.Join(dir, fo.name)
		if err := os.WriteFile(path, fo.data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func visibleTableFields(fields []excelconv.Field, exportTags []string) []excelconv.Field {
	return excelconv.VisibleTableFields(fields, exportTags)
}

func visibleStructFields(fields []excelconv.StructField, exportTags []string) []excelconv.StructField {
	var v []excelconv.StructField
	for _, sf := range fields {
		if !excelconv.FieldVisible(sf.Filter, exportTags) {
			continue
		}
		v = append(v, sf)
	}
	return v
}

func tableFieldsNeedSlices(fields []excelconv.Field) bool {
	for _, f := range fields {
		if f.ArraySplit != "" {
			return true
		}
	}
	return false
}

func structFieldsNeedSlices(fields []excelconv.StructField) bool {
	for _, f := range fields {
		if f.ArraySplit != "" {
			return true
		}
	}
	return false
}

func tableRowPrimaryKeyGoType(schema *excelconv.Schema, table string) string {
	switch schema.PrimaryKeyTypeForTable(table) {
	case "int":
		return "int"
	case "string":
		return "string"
	default:
		return "int64"
	}
}

// GenerateBundle 写出 GameData 与 LoadGameData；binaryData 为 true 时仅生成 .bin 加载（与 excel2json binaryExport 一致）。
func GenerateBundle(dir, pkg string, schema *excelconv.Schema, binaryData bool) error {
	var tnames []string
	for n := range schema.Tables {
		tnames = append(tnames, n)
	}
	if len(tnames) == 0 {
		_ = os.Remove(filepath.Join(dir, "loader_gen.go"))
		return nil
	}
	sort.Strings(tnames)

	raw, err := renderLoaderFile(pkg, tnames, binaryData)
	if err != nil {
		return fmt.Errorf("loader template: %w", err)
	}
	src, err := format.Source([]byte(raw))
	if err != nil {
		return fmt.Errorf("format loader_gen.go: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "loader_gen.go"), src, 0o644)
}

type fileOut struct {
	name    string
	content string
}

func sortedTableKeys(m map[string][]excelconv.Field) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func goFieldTypeTable(f excelconv.Field, schema *excelconv.Schema) string {
	base := baseGoType(f.Type, schema)
	if f.ArraySplit != "" {
		return "[]" + base
	}
	return base
}

func goFieldTypeStruct(sf excelconv.StructField, schema *excelconv.Schema) string {
	base := baseGoType(sf.Type, schema)
	if sf.ArraySplit != "" {
		return "[]" + base
	}
	return base
}

func baseGoType(typeName string, schema *excelconv.Schema) string {
	switch typeName {
	case "string":
		return "string"
	case "int":
		return "int"
	case "int64":
		return "int64"
	case "float64":
		return "float64"
	default:
		if schema.Structs[typeName] != nil {
			return typeName
		}
		if schema.Enums[typeName] != nil {
			return typeName
		}
		return "string"
	}
}
