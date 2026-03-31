package excelconv

import (
	"fmt"
	"strconv"
	"strings"
)

// ParsePrimaryKeyCell 将首列单元格解析为配置的主键类型。
func ParsePrimaryKeyCell(raw string, idType string) (interface{}, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("主键单元格为空")
	}
	switch strings.ToLower(strings.TrimSpace(idType)) {
	case "string":
		return raw, nil
	case "int":
		v, err := parseIntDefault(raw, 0)
		if err != nil {
			return nil, fmt.Errorf("主键 %q 无法解析为 int: %w", raw, err)
		}
		return v, nil
	case "int64":
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			f, ferr := strconv.ParseFloat(raw, 64)
			if ferr != nil {
				return nil, fmt.Errorf("主键 %q 无法解析为 int64: %w", raw, err)
			}
			return int64(f), nil
		}
		return v, nil
	default:
		return nil, fmt.Errorf("未知主键类型 %q", idType)
	}
}
