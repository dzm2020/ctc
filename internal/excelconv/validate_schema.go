package excelconv

import (
	"fmt"
	"strings"
)

// ValidateSchemaRules 校验 @Type 语义：枚举默认值与数值唯一、数组列不可参与分组/索引。
// 用于单文件解析后或多文件 MergeSchemas 之后（合并后的定义可能产生枚举值冲突）。
func ValidateSchemaRules(s *Schema) error {
	if s == nil {
		return nil
	}
	for en, members := range s.Enums {
		if err := validateEnumMembers(en, members); err != nil {
			return err
		}
	}
	for tname, fields := range s.Tables {
		for _, f := range fields {
			if f.ArraySplit != "" && (strings.TrimSpace(f.Group) != "" || strings.TrimSpace(f.Index) != "") {
				return fmt.Errorf("表 %q 字段 %q: 已配置「数组切割」的列不能声明「分组」或「索引」", tname, f.Name)
			}
		}
	}
	return nil
}

func validateEnumMembers(en string, members []EnumMember) error {
	seenName := make(map[string]struct{})
	seenVal := make(map[int]string)
	for _, m := range members {
		if strings.TrimSpace(m.Name) == "" {
			continue
		}
		if strings.TrimSpace(m.Value) == "" {
			return fmt.Errorf("枚举 %q 成员 %q: 「默认值」不能为空", en, m.Name)
		}
		v, err := parseIntDefault(m.Value, 0)
		if err != nil {
			return fmt.Errorf("枚举 %q 成员 %q: 默认值 %q 无法解析为整数: %w", en, m.Name, m.Value, err)
		}
		if _, dup := seenName[m.Name]; dup {
			return fmt.Errorf("枚举 %q: 成员名 %q 重复", en, m.Name)
		}
		seenName[m.Name] = struct{}{}
		if prev, dup := seenVal[v]; dup {
			return fmt.Errorf("枚举 %q: 数值 %d 被成员 %q 与 %q 重复使用", en, v, prev, m.Name)
		}
		seenVal[v] = m.Name
	}
	return nil
}
