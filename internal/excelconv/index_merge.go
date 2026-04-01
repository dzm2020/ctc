package excelconv

import "fmt"

// fillIndexSeenFromRows 从已有行收集各「索引」已出现的复合键；若表内已有重复键则报错。
func fillIndexSeenFromRows(rows map[string]map[string]interface{}, table string, schema *Schema, target ExportTarget) (map[string]map[string]struct{}, error) {
	visible := VisibleTableFields(schema.Tables[table], target)
	seen := make(map[string]map[string]struct{})
	for _, ix := range DistinctFieldIndexes(visible) {
		seen[ix] = make(map[string]struct{})
	}
	if len(seen) == 0 {
		return seen, nil
	}
	for _, rec := range rows {
		if err := addRecordToIndexSeen(rec, visible, schema, seen, table); err != nil {
			return nil, err
		}
	}
	return seen, nil
}

func addRecordToIndexSeen(rec map[string]interface{}, visible []Field, schema *Schema, seen map[string]map[string]struct{}, table string) error {
	for _, ix := range DistinctFieldIndexes(visible) {
		k, err := RowIndexKeyString(rec, FieldsInIndex(visible, ix), schema)
		if err != nil {
			return fmt.Errorf("索引 %q: %w", ix, err)
		}
		if _, dup := seen[ix][k]; dup {
			return fmt.Errorf("索引 %q 复合键重复（同表内必须唯一）", ix)
		}
		seen[ix][k] = struct{}{}
	}
	return nil
}
