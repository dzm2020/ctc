package excelconv

import "strings"

// NormalizeFilterTag 策划填写的单个标签：去空白并统一为大写（比较用）。
func NormalizeFilterTag(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// ParseFieldFilterTags 解析 @Type「筛选」列：逗号分隔；标签可为任意字符串（如 C、S、GM、EDITOR），不限两种。
// 空字符串或仅空白视为无标签（与任意导出配置均匹配，即全端导出）。
// 兼容旧值 CS：展开为 C 与 S。
func ParseFieldFilterTags(f FieldFilter) []string {
	raw := strings.TrimSpace(string(f))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	seen := make(map[string]struct{})
	var out []string
	for _, p := range parts {
		t := NormalizeFilterTag(p)
		if t == "" {
			continue
		}
		for _, e := range expandLegacyFilterToken(t) {
			if _, ok := seen[e]; ok {
				continue
			}
			seen[e] = struct{}{}
			out = append(out, e)
		}
	}
	return out
}

func expandLegacyFilterToken(t string) []string {
	switch t {
	case "CS":
		return []string{"C", "S"}
	default:
		return []string{t}
	}
}

// FieldVisible 当字段筛选标签与 exportTags 有非空交集时导出；字段未配置标签时视为全端。
func FieldVisible(f FieldFilter, exportTags []string) bool {
	fieldTags := ParseFieldFilterTags(f)
	if len(fieldTags) == 0 {
		return true
	}
	if len(exportTags) == 0 {
		return true
	}
	set := make(map[string]struct{}, len(exportTags))
	for _, t := range exportTags {
		if t == "" {
			continue
		}
		set[strings.ToUpper(strings.TrimSpace(t))] = struct{}{}
	}
	for _, t := range fieldTags {
		if _, ok := set[t]; ok {
			return true
		}
	}
	return false
}

// ResolveExportFilterTags 根据配置得到导出用标签列表（大写、去重）；支持 C、S 及任意自定义标签。
// 每项可按逗号再拆分；解析结果为空时默认 C+S（与旧版 target both 一致；若项目只用自定义标签请在配置中显式列出）。
func ResolveExportFilterTags(filterTags []string) ([]string, error) {
	seen := make(map[string]struct{})
	var out []string
	for _, s := range filterTags {
		for _, part := range strings.Split(s, ",") {
			t := NormalizeFilterTag(part)
			if t == "" {
				continue
			}
			for _, e := range expandLegacyFilterToken(t) {
				if _, ok := seen[e]; ok {
					continue
				}
				seen[e] = struct{}{}
				out = append(out, e)
			}
		}
	}
	if len(out) > 0 {
		return out, nil
	}
	return []string{"C", "S"}, nil
}
