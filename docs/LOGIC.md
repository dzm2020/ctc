# ctc 处理逻辑说明

本文描述 `excel2json` 从读取配置到写出产物的**流水线**与各包**职责**，便于维护与二次开发。面向工程实现，与 [README](../README.md) 中的使用说明互补。

---

## 1. 总览流水线

```
配置文件 JSON
    → 加载 config.Config
    → 解析 filterTags → exportTags（与 @Type「筛选」求交用）
    → 收集 inputs 下所有 .xlsx
    → 对每个 xlsx：打开 → ParseTypeSheet(@Type) → ConvertWorkbook(数据 sheet)
    → 合并各文件的 Schema、合并各表 map 数据（主键/索引冲突则失败）
    → ValidateSchemaRules(合并后 Schema)
    → 清空 jsonPath / codePath / csharpPath（去重、安全校验）
    → 按表写出 .json 或 .bin
    → 可选：gogen.WritePackage + GenerateBundle（Go）
    → 可选：csharpgen.WritePackage（C#）
```

**数据流**：Excel 行 → `map[string]interface{}`（键为字段名，`id` 为主键）→ 有序切片 → JSON 数组或 tablebin 编码。

---

## 2. 包与目录职责

| 路径 | 职责 |
|------|------|
| `cmd/excel2json` | CLI 入口；编排上述流水线；日志输出。 |
| `internal/config` | 配置文件 JSON 反序列化；输出路径默认值。 |
| `internal/excelconv` | `@Type` 解析、`Schema`；数据 sheet 转表 map；筛选标签；合并与校验；tablebin 列描述构建。 |
| `internal/gogen` | Go 代码生成（text/template + embed）；单表文件、loader、枚举/结构体。 |
| `internal/csharpgen` | C# 代码生成（text/template + embed）；TableBinDecoder、表类型、GameData。 |
| `internal/outputs` | 生成前清空输出目录（防残留旧表/旧代码）。 |
| `pkg/tablebin` | 表二进制格式编解码（与 C# `TableBinDecoder` 一致）。 |
| `pkg/tableload` | （可选）运行时加载辅助。 |

---

## 3. 核心类型：`Schema`

由 `ParseTypeSheet` 填充，包含：

- `Tables[表名][]Field` — 列定义（含分组、索引、数组切割、筛选等）
- `Enums`、`Structs`、`EnumValue`、`TableIDType`（主键类型）

**可见字段**：`VisibleTableFields` 按 `exportTags` 与字段 `Filter` 求交；名为 `id` 的表头行不参与普通列（主键单独处理）。

---

## 4. 多文件合并规则（要点）

- **Schema**：`MergeSchemas` 合并枚举/结构/表字段；同名后者覆盖前者顺序中的同名字段定义。
- **表数据**：`MergeTableMaps` 将多张 xlsx 的同名表合并到同一 `map[主键]行`；主键重复或索引键冲突报错。

---

## 5. 生成物与配置开关

| 配置 | 影响 |
|------|------|
| `binaryExport` | 数据为 `.bin` 或 `.json`；Go/C# 加载代码只支持对应一种。 |
| `skipGo` / `skipCSharp` | 跳过对应语言生成。 |
| `csharpPath` 空 | 不生成 C#。 |
| `filterTags` | 见 README；空配置解析后默认 `C`+`S`。 |

生成前 **`outputs.ClearDirectoriesUnique`** 会清空本次涉及的输出目录，避免删除旧 `table_*.go` 后仍残留。

---

## 6. 扩展阅读

- 表格式、@Type 列含义： [README](../README.md)
- 二进制列与 `Kind`：`internal/excelconv/table_bin_spec.go`、`pkg/tablebin/kind.go`
