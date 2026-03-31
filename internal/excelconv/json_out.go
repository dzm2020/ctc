package excelconv

import "sort"

// TableRowsToOrderedSlice 将 map[主键]行 转为按主键排序的切片，用于写出 JSON 数组。
func TableRowsToOrderedSlice(rows map[string]map[string]interface{}) []map[string]interface{} {
	if len(rows) == 0 {
		return nil
	}
	keys := make([]string, 0, len(rows))
	for k := range rows {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]map[string]interface{}, 0, len(rows))
	for _, k := range keys {
		out = append(out, rows[k])
	}
	return out
}
