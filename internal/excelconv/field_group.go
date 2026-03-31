package excelconv

import "strings"

// RowEmitPiece 描述表行结构体字段的生成顺序（顶栏字段或分组块）。
type RowEmitPiece struct {
	Group string // 非空表示分组块首次出现，JSON key 为 Group 原文（与 Field 互斥）
	Field Field  // 非零时表示顶栏字段（Group 必须为空）
}

// RowStructEmitOrder 按 @Type 中字段顺序：顶栏字段逐条出现；每个分组仅在首次出现时占一位。
func RowStructEmitOrder(visible []Field) []RowEmitPiece {
	var out []RowEmitPiece
	seenGroup := make(map[string]bool)
	for _, f := range visible {
		g := strings.TrimSpace(f.Group)
		if g == "" {
			out = append(out, RowEmitPiece{Field: f})
			continue
		}
		if seenGroup[g] {
			continue
		}
		seenGroup[g] = true
		out = append(out, RowEmitPiece{Group: g})
	}
	return out
}

// DistinctFieldGroups 返回 @Type「分组」列中出现过的 key 列表（顺序与首次出现一致）。
func DistinctFieldGroups(visible []Field) []string {
	emit := RowStructEmitOrder(visible)
	var out []string
	for _, p := range emit {
		if p.Group != "" {
			out = append(out, p.Group)
		}
	}
	return out
}

// FieldsInGroup 返回某分组 key 下（trim 后相等）的全部字段，顺序与 schema 一致。
func FieldsInGroup(visible []Field, groupKey string) []Field {
	g := strings.TrimSpace(groupKey)
	var fs []Field
	for _, f := range visible {
		if strings.TrimSpace(f.Group) == g {
			fs = append(fs, f)
		}
	}
	return fs
}
