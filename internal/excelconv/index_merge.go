package excelconv

// fillIndexSeenFromRows 从已有行收集各「索引」已出现的复合键；若表内已有重复键则报错。
func fillIndexSeenFromRows(rows map[string]map[string]interface{}, table string, schema *Schema, target ExportTarget) (map[string]map[string]indexSeenEntry, error) {
	visible := VisibleTableFields(schema.Tables[table], target)
	seen := make(map[string]map[string]indexSeenEntry)
	for _, ix := range DistinctFieldIndexes(visible) {
		seen[ix] = make(map[string]indexSeenEntry)
	}
	if len(seen) == 0 {
		return seen, nil
	}
	for pk, rec := range rows {
		if err := addRecordToIndexSeen(rec, visible, schema, seen, table, 0, pk); err != nil {
			return nil, err
		}
	}
	return seen, nil
}
