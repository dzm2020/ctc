package excelconv

import "strings"

// DistinctFieldIndexes 返回 @Type「索引」列中出现过的 key 列表（顺序与字段首次出现一致）。
func DistinctFieldIndexes(visible []Field) []string {
	seen := make(map[string]bool)
	var out []string
	for _, f := range visible {
		ix := strings.TrimSpace(f.Index)
		if ix == "" || seen[ix] {
			continue
		}
		seen[ix] = true
		out = append(out, ix)
	}
	return out
}

// FieldsInIndex 返回某索引 key 下的全部字段，顺序与 schema 一致。
func FieldsInIndex(visible []Field, indexKey string) []Field {
	want := strings.TrimSpace(indexKey)
	var fs []Field
	for _, f := range visible {
		if strings.TrimSpace(f.Index) == want {
			fs = append(fs, f)
		}
	}
	return fs
}
