package excelconv

import "strings"

// RowJSONIDKey 行数据 JSON 与 Go Getter 使用的默认主键字段名（首列，全局唯一）。
const RowJSONIDKey = "id"

// DefaultPrimaryKeyType 未在 @Type 中配置「主键」行时的默认主键类型。
const DefaultPrimaryKeyType = "int64"

// FieldFilter @Type「筛选」列原文：逗号分隔标签，如 C,S；空表示全端；CS 兼容为双端。
type FieldFilter string

// Field 表字段（@Type 中「表头」行）。
type Field struct {
	Table      string
	NameCN     string
	Name       string
	Type       string // string | int | int64 | float64 | 枚举名 | 结构名
	ArraySplit string
	Default    string
	Filter     FieldFilter
	Group      string // 分组：用于 Go 行视图 / Table 按组查询；导出的 JSON 仍为扁平字段
	Index      string // 索引：多列组成复合键，表内不可重复；JSON 仍为扁平
}

// EnumMember 枚举成员。
type EnumMember struct {
	Enum    string
	NameCN  string
	Name    string
	Type    string
	Value   string
	Filter  FieldFilter
}

// StructField 结构体字段（「结构」行）。
type StructField struct {
	Struct  string
	NameCN  string
	Name    string
	Type    string
	ArraySplit string
	Default string
	Filter  FieldFilter
}

// Schema 从 @Type 解析得到的全部定义。
type Schema struct {
	Tables      map[string][]Field            // table -> ordered fields
	Enums       map[string][]EnumMember       // enum -> members
	Structs     map[string][]StructField      // structName -> fields
	EnumValue   map[string]map[string]int     // enumName -> memberName -> int value
	TableIDType map[string]string             // 表名 -> 主键类型：int64 | int | string（@Type「主键」行）
}

func NewSchema() *Schema {
	return &Schema{
		Tables:      make(map[string][]Field),
		Enums:       make(map[string][]EnumMember),
		Structs:     make(map[string][]StructField),
		EnumValue:   make(map[string]map[string]int),
		TableIDType: make(map[string]string),
	}
}

// PrimaryKeyTypeForTable 返回该表主键 id 的类型，未配置时为 int64。
func (s *Schema) PrimaryKeyTypeForTable(table string) string {
	if s == nil {
		return DefaultPrimaryKeyType
	}
	t := strings.TrimSpace(s.TableIDType[table])
	if t == "" {
		return DefaultPrimaryKeyType
	}
	return strings.ToLower(t)
}

func (s *Schema) registerEnumValue(enum, member string, v int) {
	if s.EnumValue[enum] == nil {
		s.EnumValue[enum] = make(map[string]int)
	}
	s.EnumValue[enum][member] = v
}

