package csharpgen

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ctc/internal/excelconv"
)

// WritePackage 写出 C# 工程文件：TableBinDecoder（仅 binary）、枚举、结构体、各表、GameData、csproj。
func WritePackage(dir, ns string, schema *excelconv.Schema, exportTags []string, binary bool) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil
	}
	ns = strings.TrimSpace(ns)
	if ns == "" {
		ns = "GameData"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	if binary {
		decSrc, err := executeCSharpTemplate("csharp_table_bin_decoder", csharpNamespaceData{Namespace: ns})
		if err != nil {
			return fmt.Errorf("render TableBinDecoder: %w", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "TableBinDecoder.cs"), []byte(decSrc), 0o644); err != nil {
			return err
		}
	}

	if len(schema.Enums) > 0 {
		src, err := emitEnumsFile(ns, schema)
		if err != nil {
			return fmt.Errorf("render Enums.gen.cs: %w", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "Enums.gen.cs"), []byte(src), 0o644); err != nil {
			return err
		}
	}

	if len(schema.Structs) > 0 {
		src, err := emitStructsFile(ns, schema, exportTags, binary)
		if err != nil {
			return fmt.Errorf("render Structs.gen.cs: %w", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "Structs.gen.cs"), []byte(src), 0o644); err != nil {
			return err
		}
	}

	tkeys := sortedTableKeys(schema.Tables)
	slugUse := make(map[string]int)
	for _, tname := range tkeys {
		slug := privateFieldSlug(tname)
		if slug == "" {
			slug = "table"
		}
		n := slugUse[slug]
		slugUse[slug]++
		var fname string
		if n == 0 {
			fname = fmt.Sprintf("Table_%s.gen.cs", slug)
		} else {
			fname = fmt.Sprintf("Table_%s_%d.gen.cs", slug, n+1)
		}
		src, err := emitTableFile(ns, tname, schema, exportTags, binary)
		if err != nil {
			return fmt.Errorf("render %s: %w", fname, err)
		}
		if err := os.WriteFile(filepath.Join(dir, fname), []byte(src), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", fname, err)
		}
	}

	if len(tkeys) > 0 {
		gd, err := emitGameDataFile(ns, schema, binary)
		if err != nil {
			return fmt.Errorf("render GameData.gen.cs: %w", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "GameData.gen.cs"), []byte(gd), 0o644); err != nil {
			return err
		}
	}

	csproj, err := executeCSharpTemplate("csharp_csproj", csharpCsprojTmpl{
		TargetFramework: "net8.0",
		RootNamespace:   ns,
	})
	if err != nil {
		return fmt.Errorf("render GameData.csproj: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "GameData.csproj"), []byte(csproj), 0o644); err != nil {
		return err
	}
	return nil
}

func sortedTableKeys(m map[string][]excelconv.Field) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
