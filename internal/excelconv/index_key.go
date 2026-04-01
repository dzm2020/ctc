package excelconv

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const indexKeySep = "\u001e"

// RowIndexKeyString 将一行 map 按索引字段顺序拼成与 rowGroupKeyStr 规则一致的 string（用于唯一性校验）。
func RowIndexKeyString(rec map[string]interface{}, fields []Field, schema *Schema) (string, error) {
	parts := make([]string, 0, len(fields))
	for _, fld := range fields {
		v, ok := rec[fld.Name]
		if !ok {
			v = zeroForField(fld, schema)
		}
		s, err := valueToIndexKeyPart(v, fld, schema)
		if err != nil {
			return "", fmt.Errorf("字段 %q: %w", fld.Name, err)
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, indexKeySep), nil
}

func valueToIndexKeyPart(v interface{}, fld Field, schema *Schema) (string, error) {
	if v == nil {
		v = zeroForField(fld, schema)
	}
	if fld.ArraySplit != "" {
		switch x := v.(type) {
		case []string:
			return strings.Join(x, ","), nil
		case []interface{}:
			ss := make([]string, 0, len(x))
			for _, e := range x {
				s, err := jsonScalarOrObjectForIndex(e)
				if err != nil {
					return "", err
				}
				ss = append(ss, s)
			}
			return strings.Join(ss, ","), nil
		default:
			b, err := json.Marshal(x)
			if err != nil {
				return "", err
			}
			return string(b), nil
		}
	}
	switch fld.Type {
	case "string":
		switch x := v.(type) {
		case string:
			return x, nil
		default:
			return fmt.Sprint(x), nil
		}
	case "int", "int64":
		i, err := coerceInt64(v)
		if err != nil {
			return "", err
		}
		return strconv.FormatInt(i, 10), nil
	case "float64":
		f, err := coerceFloat64(v)
		if err != nil {
			return "", err
		}
		return strconv.FormatFloat(f, 'g', -1, 64), nil
	default:
		if schema != nil && schema.Enums[fld.Type] != nil {
			i, err := coerceInt64(v)
			if err != nil {
				return "", err
			}
			return strconv.FormatInt(i, 10), nil
		}
		if schema != nil && schema.Structs[fld.Type] != nil {
			b, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			return string(b), nil
		}
		return fmt.Sprint(v), nil
	}
}

// jsonScalarOrObjectForIndex 将数组元素稳定序列化（结构体 map 用 json.Marshal，避免 fmt.Sprint(map) 键序随机）。
func jsonScalarOrObjectForIndex(e interface{}) (string, error) {
	switch x := e.(type) {
	case map[string]interface{}:
		b, err := json.Marshal(x)
		if err != nil {
			return "", err
		}
		return string(b), nil
	case string:
		return x, nil
	default:
		b, err := json.Marshal(x)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}

func coerceFloat64(v interface{}) (float64, error) {
	switch x := v.(type) {
	case float64:
		return x, nil
	case float32:
		return float64(x), nil
	case int:
		return float64(x), nil
	case int32:
		return float64(x), nil
	case int64:
		return float64(x), nil
	case json.Number:
		return x.Float64()
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" {
			return 0, nil
		}
		return strconv.ParseFloat(s, 64)
	}
}

func coerceInt64(v interface{}) (int64, error) {
	switch x := v.(type) {
	case int:
		return int64(x), nil
	case int32:
		return int64(x), nil
	case int64:
		return x, nil
	case float64:
		return int64(x), nil
	case json.Number:
		return x.Int64()
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" {
			return 0, nil
		}
		return strconv.ParseInt(s, 10, 64)
	}
}
