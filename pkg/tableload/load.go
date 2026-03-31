package tableload

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadRawTable 加载 {表名}.json，格式为 map[主键字符串]行对象（与 excel2json 输出一致）。
func LoadRawTable(path string) (map[string]map[string]interface{}, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out map[string]map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}
	return out, nil
}

// LoadEnums 从自定义 JSON 加载枚举映射；excel2json 不再生成 __enums__.json，枚举请用生成的 Go 常量。
func LoadEnums(path string) (map[string]map[string]int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out map[string]map[string]int
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}
	return out, nil
}
