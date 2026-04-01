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
	// FilterTags 导出时使用的标签集合（如 C、S）；与 @Type「筛选」列中逗号分隔的标签求交集，有交集则导出该字段/成员。
	// 非空时优先于 Target；省略或全为空元素时按 Target 推断（both→C+S）。
	FilterTags []string `json:"filterTags"`
	// Target 兼容旧配置：both | client | server；仅在 FilterTags 无效（未配置或可解析标签为空）时使用。
	Target string `json:"target"`
	// PrettyJSON 为 true 时 JSON 缩进格式化；省略则为 true。
	PrettyJSON *bool `json:"prettyJson"`
	// GoPackage 生成 Go 代码的包名，默认 gamedata。
	GoPackage string `json:"goPackage"`
	// SkipGo 为 true 时不生成 Go 代码，默认 false。
	SkipGo bool `json:"skipGo"`
	// BinaryExport 为 true：仅写出 «表名».bin（pkg/tablebin 紧凑格式），且生成仅支持 .bin 的加载代码；为 false：仅写出 «表名».json 且生成仅支持 JSON 加载。二者互斥，运行时只存在一种表文件。
	BinaryExport bool `json:"binaryExport"`
}

// Output 描述生成物输出路径。
type Output struct {
	CodePath string `json:"codePath"` // Go 代码目录
	JsonPath string `json:"jsonPath"` // JSON 表目录
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

// TargetOrDefault 返回小写 target 字符串。
func (c *Config) TargetOrDefault() string {
	s := strings.ToLower(strings.TrimSpace(c.Target))
	if s == "" {
		return "both"
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
