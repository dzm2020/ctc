package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ctc/internal/config"
	"ctc/internal/excelconv"
	"ctc/internal/gogen"
	"ctc/pkg/tablebin"

	"github.com/xuri/excelize/v2"
)

func main() {
	configPath := flag.String("config", "", "配置文件 JSON 路径（必选）")
	flag.Parse()

	if strings.TrimSpace(*configPath) == "" {
		fmt.Fprintln(os.Stderr, "用法: excel2json -config <配置文件.json>")
		os.Exit(1)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	jsonOut := cfg.JsonPathOrDefault()
	goOut := cfg.CodePathOrDefault()
	exportTags, err := excelconv.ResolveExportFilterTags(cfg.FilterTags, cfg.TargetOrDefault())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	files, err := resolveXlsxInputs(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	mergedTables := make(map[string]map[string]map[string]interface{})
	var schemas []*excelconv.Schema
	var schemaMergedForKeys *excelconv.Schema
	for _, xlsxPath := range files {
		f, err := excelize.OpenFile(xlsxPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "打开表格 %s: %v\n", xlsxPath, err)
			os.Exit(1)
		}
		schema, err := excelconv.ParseTypeSheet(f)
		if err != nil {
			_ = f.Close()
			fmt.Fprintf(os.Stderr, "解析 @Type (%s): %v\n", xlsxPath, err)
			os.Exit(1)
		}
		tables, err := excelconv.ConvertWorkbook(f, schema, exportTags)
		_ = f.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "转表 (%s): %v\n", xlsxPath, err)
			os.Exit(1)
		}
		schemas = append(schemas, schema)
		if schemaMergedForKeys == nil {
			schemaMergedForKeys = schema
		} else {
			schemaMergedForKeys = excelconv.MergeSchemas([]*excelconv.Schema{schemaMergedForKeys, schema})
		}
		if err := excelconv.MergeTableMaps(mergedTables, tables, xlsxPath, schemaMergedForKeys, exportTags); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		fmt.Printf("已处理 %s\n", xlsxPath)
	}
	schemaMerged := excelconv.MergeSchemas(schemas)
	if err := excelconv.ValidateSchemaRules(schemaMerged); err != nil {
		fmt.Fprintf(os.Stderr, "@Type 合并后校验失败: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(jsonOut, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "创建目录: %v\n", err)
		os.Exit(1)
	}

	indent := ""
	if cfg.PrettyJSONOrDefault() {
		indent = "  "
	}

	for _, name := range excelconv.StableTableNames(mergedTables) {
		rows := mergedTables[name]
		arr := excelconv.TableRowsToOrderedSlice(rows)
		if cfg.BinaryExport {
			idKind, cols := excelconv.BuildTableBinSpec(schemaMerged, name, exportTags)
			binPath := filepath.Join(jsonOut, name+".bin")
			if err := tablebin.EncodeFile(binPath, excelconv.RowJSONIDKey, idKind, cols, arr); err != nil {
				fmt.Fprintf(os.Stderr, "写入 %s.bin: %v\n", name, err)
				os.Exit(1)
			}
			fmt.Printf("已写入 %s.bin（%d 行）\n", name, len(rows))
		} else {
			if err := writeJSON(filepath.Join(jsonOut, name+".json"), arr, indent); err != nil {
				fmt.Fprintf(os.Stderr, "写入 %s.json: %v\n", name, err)
				os.Exit(1)
			}
			fmt.Printf("已写入 %s.json（%d 行）\n", name, len(rows))
		}
	}

	goPkg := cfg.GoPackageOrDefault()
	if !cfg.SkipGo {
		goOut = strings.TrimSpace(goOut)
		if goOut != "" {
			if err := gogen.WritePackage(goOut, goPkg, schemaMerged, exportTags, cfg.BinaryExport); err != nil {
				fmt.Fprintf(os.Stderr, "生成 Go 包: %v\n", err)
				os.Exit(1)
			}
			if err := gogen.GenerateBundle(goOut, goPkg, schemaMerged, cfg.BinaryExport); err != nil {
				fmt.Fprintf(os.Stderr, "生成 loader_gen.go: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("已生成 Go 加载代码: %s (package %s)\n", goOut, goPkg)
			if cfg.BinaryExport {
				fmt.Printf("  使用 LoadGameData(%q) 加载 .bin（与当前 binaryExport 配置一致）\n", jsonOut)
			} else {
				fmt.Printf("  使用 LoadGameData(%q) 加载 .json（与当前 binaryExport 配置一致）\n", jsonOut)
			}
		}
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
