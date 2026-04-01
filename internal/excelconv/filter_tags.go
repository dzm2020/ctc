package excelconv

import (
	"fmt"
	"strings"
)

// NormalizeFilterTag 策划填写的单个标签：去空白并统一为大写（比较用）。
func NormalizeFilterTag(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// ParseFieldFilterTags 解析 @Type「筛选」列：逗号分隔，如 "C,S"、"CS"。
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

// ResolveExportFilterTags 根据配置得到导出用标签列表（大写、去重）。
// filterTags 非空时仅使用其中标签（每项可按逗号再拆分）；否则按 target 推断（both→C+S，client→C，server→S）。
func ResolveExportFilterTags(filterTags []string, target string) ([]string, error) {
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
	switch strings.ToLower(strings.TrimSpace(target)) {
	case "both", "cs", "all", "":
		return []string{"C", "S"}, nil
	case "client", "c":
		return []string{"C"}, nil
	case "server", "s":
		return []string{"S"}, nil
	default:
		return nil, fmt.Errorf("target 无效: %q（可选 both、client、server；或配置 filterTags）", target)
	}
}
