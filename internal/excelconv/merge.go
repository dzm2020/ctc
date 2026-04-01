package excelconv

import (
	"fmt"
	"sort"
)

// MergeSchemas 合并多个工作簿的 @Type 定义，用于生成汇总后的 Go 类型。
// 同名字段 / 枚举成员 / 结构字段以后出现的配置覆盖前者。
func MergeSchemas(list []*Schema) *Schema {
	if len(list) == 0 {
		return NewSchema()
	}
	if len(list) == 1 {
		return list[0]
	}
	out := NewSchema()

	mergeTableFields := func(tname string, incoming []Field) {
		byName := make(map[string]Field)
		var order []string
		for _, f := range out.Tables[tname] {
			byName[f.Name] = f
			order = append(order, f.Name)
		}
		for _, f := range incoming {
			f.Table = tname
			if _, ok := byName[f.Name]; !ok {
				order = append(order, f.Name)
			}
			byName[f.Name] = f
		}
		next := make([]Field, 0, len(order))
		for _, name := range order {
			f := byName[name]
			f.Table = tname
			next = append(next, f)
		}
		out.Tables[tname] = next
	}

	for _, s := range list {
		for tname, fields := range s.Tables {
			mergeTableFields(tname, fields)
		}
	}

	mergeEnumMembers := func(en string, incoming []EnumMember) {
		byName := make(map[string]EnumMember)
		var order []string
		for _, m := range out.Enums[en] {
			byName[m.Name] = m
			order = append(order, m.Name)
		}
		for _, m := range incoming {
			m.Enum = en
			if _, ok := byName[m.Name]; !ok {
				order = append(order, m.Name)
			}
			byName[m.Name] = m
		}
		next := make([]EnumMember, 0, len(order))
		for _, name := range order {
			next = append(next, byName[name])
		}
		out.Enums[en] = next
	}

	for _, s := range list {
		for en, members := range s.Enums {
			mergeEnumMembers(en, members)
		}
	}

	mergeStructFields := func(sn string, incoming []StructField) {
		byName := make(map[string]StructField)
		var order []string
		for _, f := range out.Structs[sn] {
			byName[f.Name] = f
			order = append(order, f.Name)
		}
		for _, f := range incoming {
			f.Struct = sn
			if _, ok := byName[f.Name]; !ok {
				order = append(order, f.Name)
			}
			byName[f.Name] = f
		}
		next := make([]StructField, 0, len(order))
		for _, name := range order {
			next = append(next, byName[name])
		}
		out.Structs[sn] = next
	}

	for _, s := range list {
		for sn, fields := range s.Structs {
			mergeStructFields(sn, fields)
		}
	}

	for _, s := range list {
		for en, m := range s.EnumValue {
			if out.EnumValue[en] == nil {
				out.EnumValue[en] = make(map[string]int)
			}
			for k, v := range m {
				out.EnumValue[en][k] = v
			}
		}
	}

	if out.TableIDType == nil {
		out.TableIDType = make(map[string]string)
	}
	for _, s := range list {
		if s.TableIDType == nil {
			continue
		}
		for k, v := range s.TableIDType {
			out.TableIDType[k] = v
		}
	}
	return out
}

// MergeTableMaps 将多张表合并到同一 map。同一表内主键 ID 已存在、或 @Type「索引」复合键冲突则报错。
func MergeTableMaps(dst map[string]map[string]map[string]interface{}, src map[string]map[string]map[string]interface{}, srcFile string, schema *Schema, exportTags []string) error {
	for tname, rows := range src {
		if dst[tname] == nil {
			dst[tname] = make(map[string]map[string]interface{})
		}
		seen, err := fillIndexSeenFromRows(dst[tname], tname, schema, exportTags)
		if err != nil {
			return fmt.Errorf("%s: %w", srcFile, err)
		}
		vis := VisibleTableFields(schema.Tables[tname], exportTags)
		for id, row := range rows {
			if _, exists := dst[tname][id]; exists {
				return fmt.Errorf("表 %q 数据合并: 主键列值 %q 重复（与已加载数据冲突；当前文件: %s）", tname, id, srcFile)
			}
			if len(seen) > 0 {
				if err := addRecordToIndexSeen(row, vis, schema, seen, tname, 0, id); err != nil {
					return fmt.Errorf("表 %q 主键列值 %q（当前文件: %s）: %w", tname, id, srcFile, err)
				}
			}
			dst[tname][id] = row
		}
	}
	return nil
}

// StableTableNames 返回排序后的表名（写出 JSON 顺序稳定）。
func StableTableNames(m map[string]map[string]map[string]interface{}) []string {
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
