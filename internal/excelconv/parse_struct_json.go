package excelconv

import (
	"encoding/json"
	"fmt"
	"strings"
)

const maxStructJSONUnwrap = 16

// parseTypedefJSONObject 将单元格内容解析为 @Type「结构」对应的 JSON 对象（扁平 map），供写出 JSON 与后续加载一致。
// 支持：BOM、null/空、整格为 JSON 字符串且内层才是 {...}（常见于 Excel 文本格式）。
func parseTypedefJSONObject(typeName, raw string) (map[string]interface{}, error) {
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "\ufeff"))
	if raw == "" || strings.EqualFold(raw, "null") {
		return map[string]interface{}{}, nil
	}
	for unwrap := 0; unwrap < maxStructJSONUnwrap; unwrap++ {
		var v interface{}
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return nil, fmt.Errorf("结构 %s JSON: %w", typeName, err)
		}
		if v == nil {
			return map[string]interface{}{}, nil
		}
		if s, ok := v.(string); ok {
			inner := strings.TrimSpace(s)
			if inner == "" {
				return map[string]interface{}{}, nil
			}
			raw = inner
			continue
		}
		m, ok := v.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("结构 %s 须为 JSON 对象 {...}，得到 %T", typeName, v)
		}
		return m, nil
	}
	return nil, fmt.Errorf("结构 %s: JSON 字符串嵌套层数过多（>%d）", typeName, maxStructJSONUnwrap)
}

// parseStructCell 解析表单元格为 @Type 结构体对应的 map（写出 JSON 的键为字段 Name，与生成 Go 的 json 标签一致）。
// 支持两种写法：1) JSON 对象 {...}（及外层 JSON 字符串包裹）；2) 策划简写「键:值,键:值」（键可用 Name 或 NameCN，中英文逗号、冒号均可）。
func parseStructCell(typeName, raw string, schema *Schema) (map[string]interface{}, error) {
	if schema == nil {
		return nil, fmt.Errorf("结构 %s: schema 为空", typeName)
	}
	fields, ok := schema.Structs[typeName]
	if !ok || len(fields) == 0 {
		return nil, fmt.Errorf("结构 %s 未在 @Type 中定义", typeName)
	}
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "\ufeff"))
	if raw == "" || strings.EqualFold(raw, "null") {
		return map[string]interface{}{}, nil
	}
	if structCellLooksLikeJSON(raw) {
		return parseTypedefJSONObject(typeName, raw)
	}
	return parseKVStructCell(typeName, raw, fields, schema)
}

func structCellLooksLikeJSON(raw string) bool {
	r := strings.TrimSpace(raw)
	if strings.HasPrefix(r, "{") || strings.HasPrefix(r, "[") {
		return true
	}
	// JSON 字符串字面值 "..."
	if len(r) >= 2 && r[0] == '"' {
		return true
	}
	return false
}

func parseKVStructCell(typeName, raw string, fields []StructField, schema *Schema) (map[string]interface{}, error) {
	out := make(map[string]interface{})
	for _, part := range splitStructKVPairs(raw) {
		key, valStr, err := splitKVPair(part)
		if err != nil {
			return nil, fmt.Errorf("结构 %s: %w", typeName, err)
		}
		sf, err := findStructFieldByKey(fields, key)
		if err != nil {
			return nil, fmt.Errorf("结构 %s: %w", typeName, err)
		}
		v, err := parseCellValueForStructField(*sf, valStr, schema)
		if err != nil {
			return nil, fmt.Errorf("结构 %s 字段 %q: %w", typeName, sf.Name, err)
		}
		out[sf.Name] = v
	}
	return out, nil
}

func splitStructKVPairs(raw string) []string {
	var parts []string
	for _, seg := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '，'
	}) {
		s := strings.TrimSpace(seg)
		if s != "" {
			parts = append(parts, s)
		}
	}
	return parts
}

func splitKVPair(part string) (key, val string, err error) {
	idx := strings.Index(part, ":")
	if idx < 0 {
		idx = strings.Index(part, "：")
	}
	if idx < 0 {
		return "", "", fmt.Errorf("片段 %q 缺少键值分隔符 (: 或 ：)", part)
	}
	key = strings.TrimSpace(part[:idx])
	val = strings.TrimSpace(part[idx+1:])
	if key == "" {
		return "", "", fmt.Errorf("片段 %q 键名为空", part)
	}
	return key, val, nil
}

func findStructFieldByKey(fields []StructField, key string) (*StructField, error) {
	key = strings.TrimSpace(key)
	for i := range fields {
		sf := &fields[i]
		if strings.EqualFold(key, sf.Name) || strings.EqualFold(key, strings.TrimSpace(sf.NameCN)) {
			return sf, nil
		}
	}
	return nil, fmt.Errorf("未知字段 %q（可用 @Type 中「Name」或「NameCN」）", key)
}

func zeroForStructField(sf StructField, schema *Schema) interface{} {
	return zeroForField(Field{Name: sf.Name, Type: sf.Type, ArraySplit: sf.ArraySplit}, schema)
}

func parseCellValueForStructField(sf StructField, raw string, schema *Schema) (interface{}, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return zeroForStructField(sf, schema), nil
	}
	if sf.ArraySplit != "" {
		return parseArrayCellForStructField(sf, raw, schema)
	}
	if _, ok := schema.Structs[sf.Type]; ok {
		return parseStructCell(sf.Type, raw, schema)
	}
	return parseScalar(sf.Type, raw, schema)
}

func parseArrayCellForStructField(sf StructField, raw string, schema *Schema) (interface{}, error) {
	parts := strings.Split(raw, sf.ArraySplit)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	if sf.Type == "string" {
		return parts, nil
	}
	out := make([]interface{}, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			z, err := parseScalar(sf.Type, "", schema)
			if err != nil {
				return nil, err
			}
			out = append(out, z)
			continue
		}
		v, err := parseCellValueForStructFieldNested(sf, p, schema)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// parseCellValueForStructFieldNested 解析数组元素（避免 parseScalar("",) 对嵌套结构走空串歧义）。
func parseCellValueForStructFieldNested(sf StructField, p string, schema *Schema) (interface{}, error) {
	if _, ok := schema.Structs[sf.Type]; ok {
		return parseStructCell(sf.Type, p, schema)
	}
	return parseScalar(sf.Type, p, schema)
}
