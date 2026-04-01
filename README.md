# ctc — Excel 策划表 → JSON / 二进制 / Go / C#

## 本库做什么

从 **`.xlsx`** 中读取 **类型定义**（`@Type` 工作表）与 **数据工作表**，完成：

| 能力 | 说明 |
|------|------|
| **数据导出** | 按表写出 `«表名».json` 或 `«表名».bin`（由配置二选一） |
| **Go 代码** | 枚举、配置结构体、每张表的行类型 + `*Table` 查询（含分组/索引）及 `loader_gen.go` |
| **C# 代码** | 同上语义的类型与 `GameData.Load` 加载入口 |
| **合并** | 多个工作簿可一起参与合并：表数据与 `@Type` 定义会合并，冲突时报错 |

### 主要支持的功能

| 功能 | 说明 |
|------|------|
| **枚举** | 在 `@Type` 中定义枚举类型及成员（整型值唯一）；表字段类型可直接引用枚举名；导出 JSON 为数值，生成 Go/C# 枚举及名称↔数值映射。 |
| **自定义结构体** | `@Type`「结构」定义可复用的字段组合；表列类型可为结构名；单元格支持 **`{...}` JSON** 或 **`键:值,键:值`** 简写（键可用字段名或中文描述）；配合「数组切割」可表示**结构体数组**。 |
| **分组（Group）** | 多列共用同一分组名：生成代码中为行提供**分组视图类型**（由扁平字段组装）；`Table` 上可按分组键查询**多行**（字段均可比较时用强类型键，否则用拼接的 string 键）。 |
| **索引（Index）** | 多列共用同一索引名：组成**复合键**，表内**唯一**校验；`Table` 上可按索引键查询**单行**。分组名与索引名在同一表中**不能重名**。 |
| **数组列** | 「数组切割」填分隔符（如 `\|`）即表示该列为切片/数组；元素可为 `string` / `int` / `int64` / `float64`、枚举或（嵌套）结构。 |
| **主键类型** | 支持 `int` / `int64` / `string`；数据表首列为主键，JSON 与生成代码中统一为字段 **`id`**。 |
| **多端导出（筛选）** | 字段「筛选」列可为**任意标签**（如 `C`、`S`、`GM`、`EDITOR` 等，逗号分隔）；与配置 `filterTags` **求交集**后决定是否导出。**不限于 C/S 两种**。未配置或解析为空时默认 `C+S`（常用双端）；若表内只用自定义标签，须在配置中显式写出相同标签。 |
| **JSON 与二进制** | `binaryExport` 二选一：`.json` 便于调试；`.bin` 为紧凑表格式（`pkg/tablebin`），生成代码仅加载对应一种。 |
| **Go / C# 双栈** | 同一套表结构可同时生成 **Go**（包 + `loader_gen.go`）与 **C#**（命名空间 + `GameData`），语义对齐。 |

命令行入口：`excel2json -config <配置文件.json>`（见 `cmd/excel2json`）。

**注意**：每次生成前会**清空**配置中的 `jsonPath`、`codePath`、以及启用时的 `csharpPath`（路径会去重；拒绝清空盘符根等不安全路径）。

---

## Windows 发布包

预置目录 **`release/windows/`**，包含：

- `excel2json.exe`（需本地构建生成）
- `excel2json.json` — 默认配置（`inputs`: `./tables`，输出 `./output/...`）
- `run.bat` / `run.ps1` — 一键执行转换
- `README-RELEASE.txt` — 使用说明
- `tables/` — 放置 `.xlsx`

**构建 exe 并打包 zip：**

```bat
scripts\build-windows-release.bat
powershell -ExecutionPolicy Bypass -File scripts\zip-windows-release.ps1
```

得到 `release\ctc-excel2json-windows-amd64.zip`，可直接分发给策划（解压后把表放进 `tables` 再运行 `run.bat`）。

---

## Linux 发布包

预置目录 **`release/linux/`**（`GOOS=linux GOARCH=amd64`，`CGO_ENABLED=0`），包含：

- `excel2json` — 可执行文件（无后缀）
- `excel2json.json` — 与 Windows 包相同的默认配置
- `run.sh` — `chmod +x run.sh excel2json && ./run.sh`
- `README-RELEASE.txt` — 使用说明
- `tables/` — 放置 `.xlsx`

**在 Linux / macOS 上构建：**

```bash
chmod +x scripts/build-linux-release.sh
./scripts/build-linux-release.sh
./scripts/zip-linux-release.sh   # 得到 release/ctc-excel2json-linux-amd64.tar.gz
```

**在 Windows 上交叉编译：**

```bat
powershell -ExecutionPolicy Bypass -File scripts\build-linux-release.ps1
powershell -ExecutionPolicy Bypass -File scripts\zip-linux-release.ps1
```

---

## 快速开始

```bash
go build -o excel2json ./cmd/excel2json
./excel2json -config examples/excel2json.example.json
```

示例配置见 [`examples/excel2json.example.json`](examples/excel2json.example.json)。输入目录下需有含 `@Type` 与数据 sheet 的 `.xlsx`（可参考 `tables/Test.xlsx`）。

流水线、包职责与合并规则见 **[docs/LOGIC.md](docs/LOGIC.md)**。

---

## 配置文件字段说明

配置文件为 **JSON**，通过 `-config` 指定。

### 顶层

| 字段 | 类型 | 含义 |
|------|------|------|
| `inputs` | `string[]` | **必填**。目录或 `.xlsx` 文件路径；目录会**递归**收集其下所有 `.xlsx`。可多条，结果会去重、排序。 |
| `output` | 对象 | 输出目录，见下表。 |
| `filterTags` | `string[]` | 本次导出要匹配的筛选标签，与 @Type「筛选」列做交集；**可为任意自定义名**（与表内一致即可，比较时统一大写），不限于 `C`/`S`。每项可内含逗号再拆分。**省略或解析后为空时默认 `["C","S"]`**。示例：`["C","S"]`、`["GM"]`、`["C","GM"]`。 |
| `prettyJson` | `bool` | 是否美化 JSON；**省略默认为 `true`**。 |
| `goPackage` | `string` | 生成 Go 代码的包名；**默认 `gamedata`**。 |
| `skipGo` | `bool` | `true` 时不生成 Go 代码；**默认 `false`**。 |
| `csharpNamespace` | `string` | 生成 C# 的根命名空间；**默认 `GameData`**。 |
| `skipCSharp` | `bool` | `true` 时不生成 C#；**默认 `false`**。 |
| `binaryExport` | `bool` | `true`：只写 `.bin`（`pkg/tablebin` 格式），生成代码只支持加载 `.bin`；`false`：只写 `.json`，代码只支持 JSON。**二者互斥**。 |

### `output` 对象

| 字段 | 含义 |
|------|------|
| `jsonPath` | 表数据输出目录（`.json` 或 `.bin`）。默认 `generated/json`。 |
| `codePath` | Go 代码输出目录。默认 `generated/gamedata`。 |
| `csharpPath` | C# 工程输出目录；**空字符串表示不生成 C#**。 |

### 配置示例

```json
{
  "inputs": ["./tables"],
  "output": {
    "codePath": "./output/gamedata",
    "jsonPath": "./output/json",
    "csharpPath": "./output/csharp"
  },
  "filterTags": ["C", "S"],
  "binaryExport": false
}
```

开启二进制导出时，将 `binaryExport` 设为 `true`；运行时加载路径需与生成时一致（如 `LoadGameData("...")` / `GameData.Load("...")`）。

---

## 策划表：@Type（类型定义）

每个工作簿中需有工作表 **`@Type`**（名称固定）。**第一行前 8 列**表头必须为：

`种类` · `对象类型` · `中文描述` · `字段名` · `字段类型` · `数组切割` · `默认值` · `筛选`

另可增加列 **`分组`**、**`索引`**（列名精确匹配即可，位置可在第 9 列及之后任意列）。

### 行种类（`种类` 列）

1. **`表头`** — 定义**数据表**的一列（非主键字段见下「主键」）。  
   - `对象类型`：表名，须与数据工作表 **sheet 名**一致。  
   - `中文描述` / `字段名` / `字段类型` / `数组切割` / `默认值` / `筛选` / `分组` / `索引`：见下文「表字段列含义」。  

2. **`枚举`** — 定义枚举类型及成员。  
   - `对象类型`：枚举类型名。  
   - `中文描述`、`字段名`：成员说明、成员名。  
   - **`默认值`（第 7 列）**：成员对应整数值，**必填且全局唯一**。  

3. **`结构`** — 定义可在表字段中引用的**结构体**（单元格可为 JSON 或 `键:值` 简写）。  

4. **`主键`** — 指定某张表首列主键类型。  
   - `对象类型`：表名。  
   - `字段类型`：`int` / `int64` / `string`；省略则按 **`int64`**。  
   - JSON 与生成代码中主键字段名固定为 **`id`**（见 `excelconv.RowJSONIDKey`）。  
   - 若 @Type 中某行「表头」的 **`字段名` 为 `id`**，该行描述主键列，且不会作为普通数据列重复导出（与首列单元格解析一致）。

### 表字段列含义（`表头` 行）

| 列 | 含义 |
|----|------|
| 中文描述 | 生成 Go/C# 时写入注释（单行化后接在字段后）。 |
| 字段名 | JSON 键名、生成代码字段名依据。 |
| 字段类型 | `string` / `int` / `int64` / `float64`，或 @Type 中已定义的**枚举名** / **结构名**。 |
| 数组切割 | 非空时表示**数组列**，单元格用该字符切分（如 `\|`）。**与「分组」「索引」互斥**。 |
| 默认值 | 标量字段单元格为空时可用的默认（解析逻辑见代码）。 |
| 筛选 | 逗号分隔的**任意**标签（如 `C,S`、`GM`、`EDITOR,QA`）；与配置的 `filterTags` 有**任一**相同即导出该字段。**空 = 不限制**，对该字段视为全端导出。兼容写法 `CS` 视为 `C`+`S`。 |
| 分组 | 同一分组名的一组字段可在生成代码中组成**行视图**并按组**查多行**。 |
| 索引 | 同一索引名的一组字段组成**复合键**，表内**唯一**，用于按索引**查单行**。 |

**约束**：同一张表里，**分组名与索引名不能相同**；带数组切割的字段不能填分组或索引。

### @Type 片段示例

| 种类 | 对象类型 | 中文描述 | 字段名 | 字段类型 | 数组切割 | 默认值 | 筛选 | 分组 | 索引 |
|------|----------|----------|--------|----------|----------|--------|------|------|------|
| 主键 | Hero | | | int64 | | | | | |
| 表头 | Hero | 英雄名 | Name | string | | | | | |
| 表头 | Hero | 等级 | Level | int | | | | | |
| 枚举 | Rarity | 白 | White | int | | 0 | | | |
| 表头 | Hero | 稀有度 | Rarity | Rarity | | | | | |
| 结构 | Reward | 道具ID | ItemId | int64 | | | | | |
| 结构 | Reward | 数量 | Count | int | | | | | |
| 表头 | Hero | 奖励 | FirstReward | Reward | | | | | |

---

## 策划表：数据工作表

- **工作表名** = @Type 里 **`表头` 的 `对象类型`（表名）**。  
- **不参与导出的表**：无 @Type 定义或名为 `@Type` 的 sheet。  

### 第 1 行（表头）

- **第 1 列**必须为 **`ArrayDict`**（固定标记）。  
- 其余列：列名 = @Type 中该表的 **`字段名`**（与 `表头` 行一致）。  
- 列名以 **`#`** 开头：该列**不参与**导出（如备注列）。  
- 第 1 列在数据行中填**主键**，写入 JSON 的 **`id`**（类型由 `主键` 行决定）。

### 数据行

- **第 1 列**：主键，表内唯一。  
- 空行、首列空白、首列以 `#` 开头的行：**跳过**。  
- **数组列**：用 `数组切割` 字符拆分；空分段在部分类型下可能影响二进制编码，建议用显式占位值（见工具链实践）。  
- **结构体列**：支持 **`{"ItemId":1,"Count":10}`** 或简写 **`ItemId:1,数量:10`**（键可用字段名或中文描述）。  
- **索引**：索引列组合在表内必须唯一，否则转换报错。

### 数据表示例（表名 `Hero`）

| ArrayDict | Name | Level | Rarity |
|-----------|------|-------|--------|
| 1 | 战士 | 10 | White |
| 2 | 法师 | 8 | White |

导出 JSON 行为类似：

```json
[
  { "id": 1, "Name": "战士", "Level": 10, "Rarity": 0 },
  { "id": 2, "Name": "法师", "Level": 8, "Rarity": 0 }
]
```

（枚举在 JSON 中为数值，与 @Type 中成员默认值一致。）

---

## 生成物概要

| 类型 | 说明 |
|------|------|
| 数据文件 | `jsonPath` 下 `表名.json` 或 `表名.bin` |
| Go | `codePath` 下 `enums_gen.go`、`structs_gen.go`、`table_*.go`、`loader_gen.go` 等 |
| C# | `csharpPath` 下 `*.gen.cs`、`GameData.csproj`；二进制模式下含 `TableBinDecoder.cs` |

多文件输入时，**同表名**数据与定义会合并；主键或索引冲突会失败。

---

## 依赖与模块

- Go **1.22+**  
- [`github.com/xuri/excelize/v2`](https://github.com/xuri/excelize) 读写 xlsx  

更细的校验规则、二进制列布局等实现细节见 `internal/excelconv`、`internal/gogen`、`internal/csharpgen`、`pkg/tablebin`。
