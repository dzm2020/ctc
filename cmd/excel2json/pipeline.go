package main

// 本文件实现 excel2json 主流水线（配置 → 读表 → 合并 → 清空输出 → 写数据与代码）。
// 阶段划分与包职责见 ../../docs/LOGIC.md。

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"ctc/internal/config"
	"ctc/internal/csharpgen"
	"ctc/internal/excelconv"
	"ctc/internal/gogen"
	"ctc/internal/outputs"
	"ctc/pkg/tablebin"

	"github.com/xuri/excelize/v2"
)

// infoLog 标准输出、无前缀时间戳，便于脚本解析或与人工阅读。
var infoLog = log.New(os.Stdout, "", 0)

func runPipeline(configPath string) error {
	configPath = strings.TrimSpace(configPath)
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("加载配置: %w", err)
	}
	infoLog.Printf("[excel2json] 配置文件: %s", configPath)

	jsonOut := cfg.JsonPathOrDefault()
	goOut := cfg.CodePathOrDefault()
	exportTags, err := excelconv.ResolveExportFilterTags(cfg.FilterTags)
	if err != nil {
		return err
	}
	infoLog.Printf("[excel2json] 导出筛选标签: %q（与 @Type「筛选」列求交集）", strings.Join(exportTags, ","))

	files, err := resolveXlsxInputs(cfg)
	if err != nil {
		return err
	}
	infoLog.Printf("[excel2json] 待处理 .xlsx 数量: %d", len(files))
	if len(files) <= 12 {
		for _, p := range files {
			infoLog.Printf("[excel2json]   - %s", p)
		}
	}

	mergedTables := make(map[string]map[string]map[string]interface{})
	var schemas []*excelconv.Schema
	var schemaMergedForKeys *excelconv.Schema

	for _, xlsxPath := range files {
		f, err := excelize.OpenFile(xlsxPath)
		if err != nil {
			return fmt.Errorf("打开表格 %s: %w", xlsxPath, err)
		}
		schema, err := excelconv.ParseTypeSheet(f)
		if err != nil {
			_ = f.Close()
			return fmt.Errorf("解析 @Type (%s): %w", xlsxPath, err)
		}
		tables, err := excelconv.ConvertWorkbook(f, schema, exportTags)
		_ = f.Close()
		if err != nil {
			return fmt.Errorf("转表 (%s): %w", xlsxPath, err)
		}
		schemas = append(schemas, schema)
		if schemaMergedForKeys == nil {
			schemaMergedForKeys = schema
		} else {
			schemaMergedForKeys = excelconv.MergeSchemas([]*excelconv.Schema{schemaMergedForKeys, schema})
		}
		if err := excelconv.MergeTableMaps(mergedTables, tables, xlsxPath, schemaMergedForKeys, exportTags); err != nil {
			return err
		}
		infoLog.Printf("[excel2json] 已解析: %s", xlsxPath)
	}

	schemaMerged := excelconv.MergeSchemas(schemas)
	if err := excelconv.ValidateSchemaRules(schemaMerged); err != nil {
		return fmt.Errorf("@Type 合并后校验: %w", err)
	}
	infoLog.Printf("[excel2json] Schema 合并完成，表数量: %d", len(schemaMerged.Tables))

	var outDirs []string
	outDirs = append(outDirs, jsonOut)
	if !cfg.SkipGo {
		if g := strings.TrimSpace(goOut); g != "" {
			outDirs = append(outDirs, g)
		}
	}
	if !cfg.SkipCSharp {
		if cs := cfg.CSharpPathOrDefault(); cs != "" {
			outDirs = append(outDirs, cs)
		}
	}
	if err := outputs.ClearDirectoriesUnique(outDirs); err != nil {
		return fmt.Errorf("清空输出目录: %w", err)
	}
	logClearedOutDirs(outDirs)

	indent := ""
	if cfg.PrettyJSONOrDefault() {
		indent = "  "
	}

	tableNames := excelconv.StableTableNames(mergedTables)
	for _, name := range tableNames {
		rows := mergedTables[name]
		arr := excelconv.TableRowsToOrderedSlice(rows)
		if cfg.BinaryExport {
			idKind, cols, err := excelconv.BuildTableBinSpec(schemaMerged, name, exportTags)
			if err != nil {
				return fmt.Errorf("表 %s 二进制列描述: %w", name, err)
			}
			binPath := filepath.Join(jsonOut, name+".bin")
			if err := tablebin.EncodeFile(binPath, excelconv.RowJSONIDKey, idKind, cols, arr); err != nil {
				return fmt.Errorf("写入 %s: %w", binPath, err)
			}
			infoLog.Printf("[excel2json] 写出 %s（%d 行）", binPath, len(rows))
		} else {
			jsonFile := filepath.Join(jsonOut, name+".json")
			if err := writeJSON(jsonFile, arr, indent); err != nil {
				return fmt.Errorf("写入 %s: %w", jsonFile, err)
			}
			infoLog.Printf("[excel2json] 写出 %s（%d 行）", jsonFile, len(rows))
		}
	}

	goPkg := cfg.GoPackageOrDefault()
	if !cfg.SkipGo {
		goOut = strings.TrimSpace(goOut)
		if goOut != "" {
			if err := gogen.WritePackage(goOut, goPkg, schemaMerged, exportTags, cfg.BinaryExport); err != nil {
				return fmt.Errorf("生成 Go 包: %w", err)
			}
			if err := gogen.GenerateBundle(goOut, goPkg, schemaMerged, cfg.BinaryExport); err != nil {
				return fmt.Errorf("生成 loader_gen.go: %w", err)
			}
			infoLog.Printf("[excel2json] Go 代码: %s (package %s)", goOut, goPkg)
			if cfg.BinaryExport {
				infoLog.Printf("[excel2json]   加载: LoadGameData(%q) → .bin", jsonOut)
			} else {
				infoLog.Printf("[excel2json]   加载: LoadGameData(%q) → .json", jsonOut)
			}
		}
	}

	if !cfg.SkipCSharp {
		csOut := cfg.CSharpPathOrDefault()
		if csOut != "" {
			ns := cfg.CSharpNamespaceOrDefault()
			if err := csharpgen.WritePackage(csOut, ns, schemaMerged, exportTags, cfg.BinaryExport); err != nil {
				return fmt.Errorf("生成 C#: %w", err)
			}
			infoLog.Printf("[excel2json] C# 工程: %s (namespace %s)", csOut, ns)
			if cfg.BinaryExport {
				infoLog.Printf("[excel2json]   加载: GameData.Load(%q) → .bin", jsonOut)
			} else {
				infoLog.Printf("[excel2json]   加载: GameData.Load(%q) → .json", jsonOut)
			}
		}
	}

	infoLog.Printf("[excel2json] 完成（共 %d 张数据表）", len(tableNames))
	return nil
}

func logClearedOutDirs(outDirs []string) {
	seen := make(map[string]struct{})
	var parts []string
	for _, d := range outDirs {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		abs, err := filepath.Abs(filepath.Clean(d))
		if err != nil {
			continue
		}
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		parts = append(parts, d)
	}
	if len(parts) > 0 {
		infoLog.Printf("[excel2json] 已清空输出目录: %s", strings.Join(parts, ", "))
	}
}

func writeJSON(path string, v interface{}, indent string) error {
	var b []byte
	var err error
	if indent != "" {
		b, err = json.MarshalIndent(v, "", indent)
	} else {
		b, err = json.Marshal(v)
	}
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
