package excelconv

import (
	"fmt"
	"strings"
)

// indexSeenEntry 记录某复合索引键首次出现的位置（用于报错）。
type indexSeenEntry struct {
	Row int    // 数据表 1-based 行号；合并阶段无行号时为 0
	PK  string // 该行主键（首列字符串）
}

func joinIndexColumnNames(visible []Field, indexKey string) string {
	flds := FieldsInIndex(visible, indexKey)
	parts := make([]string, len(flds))
	for i, f := range flds {
		parts[i] = f.Name
	}
	return strings.Join(parts, ", ")
}

func addRecordToIndexSeen(
	rec map[string]interface{},
	visible []Field,
	schema *Schema,
	seen map[string]map[string]indexSeenEntry,
	table string,
	row1 int,
	pk string,
) error {
	for _, ix := range DistinctFieldIndexes(visible) {
		k, err := RowIndexKeyString(rec, FieldsInIndex(visible, ix), schema)
		if err != nil {
			return fmt.Errorf("表 %q 索引 %q: %w", table, ix, err)
		}
		if prev, dup := seen[ix][k]; dup {
			cols := joinIndexColumnNames(visible, ix)
			switch {
			case prev.Row > 0 && row1 > 0:
				return fmt.Errorf("表 %q: 复合索引 %q 第 %d 行与第 %d 行键重复（列: %s）", table, ix, row1, prev.Row, cols)
			case prev.Row > 0:
				return fmt.Errorf("表 %q: 复合索引 %q 主键列值 %q 与第 %d 行键重复（列: %s）", table, ix, pk, prev.Row, cols)
			case row1 > 0:
				return fmt.Errorf("表 %q: 复合索引 %q 第 %d 行与已加载主键 %q 的行键重复（列: %s）", table, ix, row1, prev.PK, cols)
			default:
				return fmt.Errorf("表 %q: 复合索引 %q 主键列值 %q 与已存在主键 %q 键重复（列: %s）", table, ix, pk, prev.PK, cols)
			}
		}
		seen[ix][k] = indexSeenEntry{Row: row1, PK: pk}
	}
	return nil
}
