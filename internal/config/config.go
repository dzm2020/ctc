package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Config 对应 excel2json 配置文件（启动时仅通过 -config 指定本文件）。
type Config struct {
	// Inputs 输入目录或 .xlsx 文件路径列表；目录会递归收集其中所有 .xlsx。至少一条且能解析出文件。
	Inputs []string `json:"inputs"`
	Output Output   `json:"output"`
	// FilterTags 本次导出要保留的筛选标签集合，与 @Type「筛选」列求交集；不限于 C/S，可为 GM、EDITOR 等任意自定义标签（与表内写法一致，比较时统一大写）。
	// 省略或解析后为空时，默认按 C+S（兼容常见客户端+服务端双端）；若表内仅用自定义标签，请显式写出，例如 ["GM"] 或 ["C","S","GM"]。
	FilterTags []string `json:"filterTags"`
	// PrettyJSON 为 true 时 JSON 缩进格式化；省略则为 true。
	PrettyJSON *bool `json:"prettyJson"`
	// GoPackage 生成 Go 代码的包名，默认 gamedata。
	GoPackage string `json:"goPackage"`
	// SkipGo 为 true 时不生成 Go 代码，默认 false。
	SkipGo bool `json:"skipGo"`
	// CSharpNamespace 生成 C# 的根命名空间，默认 GameData。
	CSharpNamespace string `json:"csharpNamespace"`
	// SkipCSharp 为 true 时不生成 C# 代码（即使配置了 csharpPath），默认 false。
	SkipCSharp bool `json:"skipCSharp"`
	// BinaryExport 为 true：仅写出 «表名».bin（pkg/tablebin 紧凑格式），且生成仅支持 .bin 的加载代码；为 false：仅写出 «表名».json 且生成仅支持 JSON 加载。二者互斥，运行时只存在一种表文件。
	BinaryExport bool `json:"binaryExport"`
}

// Output 描述生成物输出路径。
type Output struct {
	CodePath   string `json:"codePath"`   // Go 代码目录
	JsonPath   string `json:"jsonPath"`   // JSON / 二进制表目录
	CSharpPath string `json:"csharpPath"` // C# 代码目录；空表示不生成 C#
}

// Load 读取并解析 JSON 配置文件；path 须为存在的文件。
func Load(path string) (*Config, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("配置文件路径为空")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("解析配置 %s: %w", path, err)
	}
	return &c, nil
}

// JsonPathOrDefault 返回 JSON 输出目录。
func (c *Config) JsonPathOrDefault() string {
	s := strings.TrimSpace(c.Output.JsonPath)
	if s == "" {
		return "generated/json"
	}
	return s
}

// CodePathOrDefault 返回 Go 代码输出目录。
func (c *Config) CodePathOrDefault() string {
	s := strings.TrimSpace(c.Output.CodePath)
	if s == "" {
		return "generated/gamedata"
	}
	return s
}

// PrettyJSONOrDefault 是否美化 JSON。
func (c *Config) PrettyJSONOrDefault() bool {
	if c.PrettyJSON == nil {
		return true
	}
	return *c.PrettyJSON
}

// GoPackageOrDefault 返回 Go 包名。
func (c *Config) GoPackageOrDefault() string {
	s := strings.TrimSpace(c.GoPackage)
	if s == "" {
		return "gamedata"
	}
	return s
}

// CSharpPathOrDefault 返回 C# 输出目录；空表示不生成。
func (c *Config) CSharpPathOrDefault() string {
	return strings.TrimSpace(c.Output.CSharpPath)
}

// CSharpNamespaceOrDefault 返回 C# 命名空间。
func (c *Config) CSharpNamespaceOrDefault() string {
	s := strings.TrimSpace(c.CSharpNamespace)
	if s == "" {
		return "GameData"
	}
	return s
}
