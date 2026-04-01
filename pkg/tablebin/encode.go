package tablebin

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
)

const (
	magic0 = 'T'
	magic1 = 'B'
	magic2 = 'B'
	ver    = byte(1)
)

const maxSliceLen = 10_000_000

// EncodeFile 将行数据按列描述写出紧凑二进制：魔数、版本、行数、字符串池、按行顺序的定长/变长字段。
// rows 为与 JSON 一致的 []map；idKey 一般为 "id"。
func EncodeFile(path string, idKey string, idKind IDKind, cols []Column, rows []map[string]interface{}) error {
	pool := buildStringPool(idKey, idKind, cols, rows)
	idx := make(map[string]uint32, len(pool))
	for i, s := range pool {
		idx[s] = uint32(i)
	}

	var body []byte
	for _, row := range rows {
		var err error
		body, err = appendRow(body, idKey, idKind, cols, row, idx)
		if err != nil {
			return err
		}
	}

	var out []byte
	out = append(out, magic0, magic1, magic2, ver)
	out = appendUvarint(out, uint64(len(rows)))
	out = appendUvarint(out, uint64(len(pool)))
	for _, s := range pool {
		b := []byte(s)
		out = appendUvarint(out, uint64(len(b)))
		out = append(out, b...)
	}
	out = append(out, body...)

	return os.WriteFile(path, out, 0o644)
}

func buildStringPool(idKey string, idKind IDKind, cols []Column, rows []map[string]interface{}) []string {
	seen := make(map[string]struct{})
	var pool []string
	add := func(s string) {
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		pool = append(pool, s)
	}

	for _, row := range rows {
		if idKind == IDString {
			add(asString(row[idKey]))
		}
		for _, c := range cols {
			switch c.Kind {
			case KindString:
				add(asString(row[c.Key]))
			case KindSliceString:
				for _, s := range asStringSlice(row[c.Key]) {
					add(s)
				}
			case KindStructJSON, KindSliceStructJSON:
				for _, blob := range jsonBlobsForCell(row[c.Key], c.Kind) {
					add(blob)
				}
			}
		}
	}
	return pool
}

func jsonBlobsForCell(v interface{}, k ColumnKind) []string {
	if k == KindStructJSON {
		if v == nil {
			return []string{"null"}
		}
		b, err := json.Marshal(v)
		if err != nil {
			return []string{"null"}
		}
		return []string{string(b)}
	}
	// KindSliceStructJSON
	arr, ok := asSlice(v)
	if !ok || len(arr) == 0 {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, el := range arr {
		b, err := json.Marshal(el)
		if err != nil {
			out = append(out, "null")
			continue
		}
		out = append(out, string(b))
	}
	return out
}

func appendRow(dst []byte, idKey string, idKind IDKind, cols []Column, row map[string]interface{}, pool map[string]uint32) ([]byte, error) {
	var err error
	dst, err = appendID(dst, row[idKey], idKind, pool)
	if err != nil {
		return dst, err
	}
	for _, c := range cols {
		dst, err = appendCell(dst, c, row[c.Key], pool)
		if err != nil {
			return dst, fmt.Errorf("column %q: %w", c.Key, err)
		}
	}
	return dst, nil
}

func appendID(dst []byte, v interface{}, idKind IDKind, pool map[string]uint32) ([]byte, error) {
	switch idKind {
	case IDString:
		dst = appendUvarint(dst, uint64(pool[asString(v)]))
		return dst, nil
	case IDInt:
		i, err := asInt64(v)
		if err != nil {
			return dst, err
		}
		return appendZigzag64(dst, i), nil
	default: // IDInt64
		i, err := asInt64(v)
		if err != nil {
			return dst, err
		}
		return appendZigzag64(dst, i), nil
	}
}

func appendCell(dst []byte, c Column, v interface{}, pool map[string]uint32) ([]byte, error) {
	switch c.Kind {
	case KindInt:
		i, err := asInt64(v)
		if err != nil {
			return dst, err
		}
		return appendZigzag64(dst, i), nil
	case KindInt64:
		i, err := asInt64(v)
		if err != nil {
			return dst, err
		}
		return appendZigzag64(dst, i), nil
	case KindFloat64:
		f, err := asFloat64(v)
		if err != nil {
			return dst, err
		}
		var scratch [8]byte
		binary.LittleEndian.PutUint64(scratch[:], math.Float64bits(f))
		return append(dst, scratch[:]...), nil
	case KindString:
		dst = appendUvarint(dst, uint64(pool[asString(v)]))
		return dst, nil
	case KindEnumInt32:
		i, err := asInt64(v)
		if err != nil {
			return dst, err
		}
		return appendZigzag64(dst, i), nil
	case KindSliceInt:
		sl, err := asIntSlice(v)
		if err != nil {
			return dst, err
		}
		if len(sl) > maxSliceLen {
			return dst, fmt.Errorf("slice too long")
		}
		dst = appendUvarint(dst, uint64(len(sl)))
		for _, x := range sl {
			dst = appendZigzag64(dst, int64(x))
		}
		return dst, nil
	case KindSliceInt64:
		sl, err := asInt64Slice(v)
		if err != nil {
			return dst, err
		}
		if len(sl) > maxSliceLen {
			return dst, fmt.Errorf("slice too long")
		}
		dst = appendUvarint(dst, uint64(len(sl)))
		for _, x := range sl {
			dst = appendZigzag64(dst, x)
		}
		return dst, nil
	case KindSliceFloat64:
		sl, err := asFloat64Slice(v)
		if err != nil {
			return dst, err
		}
		if len(sl) > maxSliceLen {
			return dst, fmt.Errorf("slice too long")
		}
		dst = appendUvarint(dst, uint64(len(sl)))
		var scratch [8]byte
		for _, f := range sl {
			binary.LittleEndian.PutUint64(scratch[:], math.Float64bits(f))
			dst = append(dst, scratch[:]...)
		}
		return dst, nil
	case KindSliceString:
		ss := asStringSlice(v)
		if len(ss) > maxSliceLen {
			return dst, fmt.Errorf("slice too long")
		}
		dst = appendUvarint(dst, uint64(len(ss)))
		for _, s := range ss {
			dst = appendUvarint(dst, uint64(pool[s]))
		}
		return dst, nil
	case KindSliceEnumInt32:
		sl, err := asInt64SliceFromIfaceSlice(v)
		if err != nil {
			return dst, err
		}
		if len(sl) > maxSliceLen {
			return dst, fmt.Errorf("slice too long")
		}
		dst = appendUvarint(dst, uint64(len(sl)))
		for _, x := range sl {
			dst = appendZigzag64(dst, x)
		}
		return dst, nil
	case KindStructJSON:
		blob := jsonBlobsForCell(v, KindStructJSON)
		s := "null"
		if len(blob) > 0 {
			s = blob[0]
		}
		dst = appendUvarint(dst, uint64(pool[s]))
		return dst, nil
	case KindSliceStructJSON:
		blobs := jsonBlobsForCell(v, KindSliceStructJSON)
		if len(blobs) > maxSliceLen {
			return dst, fmt.Errorf("slice too long")
		}
		dst = appendUvarint(dst, uint64(len(blobs)))
		for _, s := range blobs {
			dst = appendUvarint(dst, uint64(pool[s]))
		}
		return dst, nil
	default:
		return dst, fmt.Errorf("unknown column kind %d", c.Kind)
	}
}

func asString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	case fmt.Stringer:
		return x.String()
	default:
		return fmt.Sprint(x)
	}
}

func asInt64(v interface{}) (int64, error) {
	if v == nil {
		return 0, nil
	}
	switch x := v.(type) {
	case int:
		return int64(x), nil
	case int32:
		return int64(x), nil
	case int64:
		return x, nil
	case uint:
		return int64(x), nil
	case uint32:
		return int64(x), nil
	case uint64:
		return int64(x), nil
	case float64:
		return int64(x), nil
	case json.Number:
		return x.Int64()
	case string:
		return strconv.ParseInt(x, 10, 64)
	default:
		return 0, fmt.Errorf("not a number: %T", v)
	}
}

func asFloat64(v interface{}) (float64, error) {
	if v == nil {
		return 0, nil
	}
	switch x := v.(type) {
	case float64:
		return x, nil
	case float32:
		return float64(x), nil
	case int:
		return float64(x), nil
	case int64:
		return float64(x), nil
	case json.Number:
		return x.Float64()
	case string:
		return strconv.ParseFloat(x, 64)
	default:
		return 0, fmt.Errorf("not a float: %T", v)
	}
}

func asStringSlice(v interface{}) []string {
	arr, ok := asSlice(v)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, el := range arr {
		out = append(out, asString(el))
	}
	return out
}

func asIntSlice(v interface{}) ([]int, error) {
	arr, ok := asSlice(v)
	if !ok {
		return nil, nil
	}
	out := make([]int, 0, len(arr))
	for _, el := range arr {
		i, err := asInt64(el)
		if err != nil {
			return nil, err
		}
		out = append(out, int(i))
	}
	return out, nil
}

func asInt64Slice(v interface{}) ([]int64, error) {
	arr, ok := asSlice(v)
	if !ok {
		return nil, nil
	}
	out := make([]int64, 0, len(arr))
	for _, el := range arr {
		i, err := asInt64(el)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, nil
}

func asInt64SliceFromIfaceSlice(v interface{}) ([]int64, error) {
	return asInt64Slice(v)
}

func asFloat64Slice(v interface{}) ([]float64, error) {
	arr, ok := asSlice(v)
	if !ok {
		return nil, nil
	}
	out := make([]float64, 0, len(arr))
	for _, el := range arr {
		f, err := asFloat64(el)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

func asSlice(v interface{}) ([]interface{}, bool) {
	if v == nil {
		return nil, true
	}
	sl, ok := v.([]interface{})
	return sl, ok
}
