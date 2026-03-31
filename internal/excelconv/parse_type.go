package excelconv

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

const typeSheetName = "@Type"

var typeHeader = []string{"种类", "对象类型", "中文描述", "字段名", "字段类型", "数组切割", "默认值", "筛选"}

// ParseTypeSheet 解析名为 @Type 的 sheet。
func ParseTypeSheet(f *excelize.File) (*Schema, error) {
	rows, err := f.GetRows(typeSheetName)
	if err != nil {
		return nil, fmt.Errorf("@Type sheet: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("@Type: 空表")
	}
	if !headerMatches(rows[0]) {
		return nil, fmt.Errorf("@Type 第一行表头不匹配，期望 %v 得到 %v", typeHeader, rows[0])
	}
	s := NewSchema()
	for i := 1; i < len(rows); i++ {
		row := padRow(rows[i], 8)
		if rowAllEmpty(row) {
			continue
		}
		kind := strings.TrimSpace(row[0])
		switch kind {
		case "表头":
			tbl := strings.TrimSpace(row[1])
			if tbl == "" {
				return nil, fmt.Errorf("@Type 第 %d 行: 表头缺少表名", i+1)
			}
			fld := Field{
				Table:      tbl,
				NameCN:     strings.TrimSpace(row[2]),
				Name:       strings.TrimSpace(row[3]),
				Type:       strings.TrimSpace(row[4]),
				ArraySplit: strings.TrimSpace(row[5]),
				Default:    strings.TrimSpace(row[6]),
				Filter:     FieldFilter(strings.TrimSpace(row[7])),
			}
			if fld.Name == "" || fld.Type == "" {
				return nil, fmt.Errorf("@Type 第 %d 行: 表 %s 字段名或类型为空", i+1, tbl)
			}
			s.Tables[tbl] = append(s.Tables[tbl], fld)
		case "枚举":
			en := strings.TrimSpace(row[1])
			if en == "" {
				return nil, fmt.Errorf("@Type 第 %d 行: 枚举缺少类型名", i+1)
			}
			m := EnumMember{
				Enum:   en,
				NameCN: strings.TrimSpace(row[2]),
				Name:   strings.TrimSpace(row[3]),
				Type:   strings.TrimSpace(row[4]),
				Value:  strings.TrimSpace(row[6]),
				Filter: FieldFilter(strings.TrimSpace(row[7])),
			}
			if m.Name == "" {
				return nil, fmt.Errorf("@Type 第 %d 行: 枚举 %s 成员名为空", i+1, en)
			}
			v, perr := parseIntDefault(m.Value, 0)
			if perr != nil {
				return nil, fmt.Errorf("@Type 第 %d 行: 枚举默认值 %q: %w", i+1, m.Value, perr)
			}
			s.Enums[en] = append(s.Enums[en], m)
			s.registerEnumValue(en, m.Name, v)
		case "结构":
			st := strings.TrimSpace(row[1])
			if st == "" {
				return nil, fmt.Errorf("@Type 第 %d 行: 结构缺少名称", i+1)
			}
			sf := StructField{
				Struct:     st,
				NameCN:     strings.TrimSpace(row[2]),
				Name:       strings.TrimSpace(row[3]),
				Type:       strings.TrimSpace(row[4]),
				ArraySplit: strings.TrimSpace(row[5]),
				Default:    strings.TrimSpace(row[6]),
				Filter:     FieldFilter(strings.TrimSpace(row[7])),
			}
			if sf.Name == "" || sf.Type == "" {
				return nil, fmt.Errorf("@Type 第 %d 行: 结构 %s 字段名或类型为空", i+1, st)
			}
			s.Structs[st] = append(s.Structs[st], sf)
		case "主键":
			// 列：对象类型=表名，字段类型= int64（默认）| int | string
			tbl := strings.TrimSpace(row[1])
			if tbl == "" {
				return nil, fmt.Errorf("@Type 第 %d 行: 主键缺少表名（对象类型）", i+1)
			}
			typ := strings.TrimSpace(row[4])
			if typ == "" {
				typ = DefaultPrimaryKeyType
			}
			low := strings.ToLower(typ)
			if low != "int" && low != "int64" && low != "string" {
				return nil, fmt.Errorf("@Type 第 %d 行: 表 %q 主键类型仅支持 int、int64、string，得到 %q", i+1, tbl, typ)
			}
			if s.TableIDType == nil {
				s.TableIDType = make(map[string]string)
			}
			s.TableIDType[tbl] = low
		default:
			return nil, fmt.Errorf("@Type 第 %d 行: 未知种类 %q", i+1, kind)
		}
	}
	return s, nil
}

func headerMatches(h []string) bool {
	if len(h) < len(typeHeader) {
		return false
	}
	for i := range typeHeader {
		if strings.TrimSpace(h[i]) != typeHeader[i] {
			return false
		}
	}
	return true
}

func padRow(r []string, n int) []string {
	out := make([]string, n)
	copy(out, r)
	for i := len(r); i < n; i++ {
		out[i] = ""
	}
	return out
}

func rowAllEmpty(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func parseIntDefault(s string, def int) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return def, nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		f, ferr := strconv.ParseFloat(s, 64)
		if ferr != nil {
			return 0, err
		}
		return int(f), nil
	}
	return int(v), nil
}
