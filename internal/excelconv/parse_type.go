package excelconv

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

const typeSheetName = "@Type"

var typeHeader = []string{"种类", "对象类型", "中文描述", "字段名", "字段类型", "数组切割", "默认值", "筛选", "分组"}

// typeHeaderIndexName @Type 表头中可选列「索引」，与「分组」列一样可位于第 9 列及之后任意位置。
const typeHeaderIndexName = "索引"

// ParseTypeSheet 解析名为 @Type 的 sheet。
func ParseTypeSheet(f *excelize.File) (*Schema, error) {
	rows, err := f.GetRows(typeSheetName)
	if err != nil {
		return nil, fmt.Errorf("@Type sheet: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("@Type: 空表")
	}
	h0 := rows[0]
	if !headerMatches(h0) {
		return nil, fmt.Errorf("@Type 第一行表头不匹配，期望前 8 列为 %v 得到 %v", typeHeader[:8], h0)
	}
	groupCol := groupColumnIndex(h0)
	indexCol := indexColumnIndex(h0)
	rowPadLen := 8
	if groupCol >= 0 && groupCol+1 > rowPadLen {
		rowPadLen = groupCol + 1
	}
	if indexCol >= 0 && indexCol+1 > rowPadLen {
		rowPadLen = indexCol + 1
	}

	s := NewSchema()
	for i := 1; i < len(rows); i++ {
		row := padRow(rows[i], rowPadLen)
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
			grp := ""
			if groupCol >= 0 {
				grp = readTypeSheetCellAt(f, groupCol, i)
			}
			if grp == "" {
				grp = cellAt(row, groupCol)
			}
			idx := ""
			if indexCol >= 0 {
				idx = readTypeSheetCellAt(f, indexCol, i)
			}
			if idx == "" {
				idx = cellAt(row, indexCol)
			}
			fld := Field{
				Table:      tbl,
				NameCN:     strings.TrimSpace(row[2]),
				Name:       strings.TrimSpace(row[3]),
				Type:       strings.TrimSpace(row[4]),
				ArraySplit: strings.TrimSpace(row[5]),
				Default:    strings.TrimSpace(row[6]),
				Filter:     FieldFilter(strings.TrimSpace(row[7])),
				Group:      grp,
				Index:      idx,
			}
			if fld.Name == "" || fld.Type == "" {
				return nil, fmt.Errorf("@Type 第 %d 行: 表 %s 字段名或类型为空", i+1, tbl)
			}
			if fld.ArraySplit != "" && (strings.TrimSpace(fld.Group) != "" || strings.TrimSpace(fld.Index) != "") {
				return nil, fmt.Errorf("@Type 工作表 第 %d 行: 表 %q 字段 %q 含「数组切割」，不能填写「分组」或「索引」", i+1, tbl, fld.Name)
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
				return nil, fmt.Errorf("@Type 工作表 第 %d 行: 枚举 %s 成员名为空", i+1, en)
			}
			if strings.TrimSpace(row[6]) == "" {
				return nil, fmt.Errorf("@Type 工作表 第 %d 行 列 %s(%q): 枚举 %q 成员 %q 必须填写「默认值」（整数，不可为空）", i+1, ExcelColumnLetter(6), typeHeader[6], en, m.Name)
			}
			v, perr := parseIntDefault(m.Value, 0)
			if perr != nil {
				return nil, fmt.Errorf("@Type 工作表 第 %d 行 列 %s(%q): 枚举 %q 成员 %q 默认值 %q: %w", i+1, ExcelColumnLetter(6), typeHeader[6], en, m.Name, m.Value, perr)
			}
			for _, ex := range s.Enums[en] {
				if ex.Name == m.Name {
					return nil, fmt.Errorf("@Type 工作表 第 %d 行: 枚举 %q 成员名 %q 重复", i+1, en, m.Name)
				}
				ev, _ := parseIntDefault(strings.TrimSpace(ex.Value), 0)
				if ev == v {
					return nil, fmt.Errorf("@Type 工作表 第 %d 行: 枚举 %q 成员 %q 的默认值数值 %d 与成员 %q 重复", i+1, en, m.Name, v, ex.Name)
				}
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
	for tname, fields := range s.Tables {
		if err := ValidateGroupIndexNamesDisjoint(tname, fields); err != nil {
			return nil, err
		}
	}
	if err := ValidateSchemaRules(s); err != nil {
		return nil, err
	}
	return s, nil
}

func headerMatches(h []string) bool {
	if len(h) < 8 {
		return false
	}
	for i := 0; i < 8; i++ {
		if strings.TrimSpace(h[i]) != typeHeader[i] {
			return false
		}
	}
	return true
}

// groupColumnIndex 在表头行中查找列名「分组」的下标（任意列位置）；未找到返回 -1。
func groupColumnIndex(header []string) int {
	for i, c := range header {
		if typeHeaderCellMatch(c, typeHeader[8]) {
			return i
		}
	}
	return -1
}

// indexColumnIndex 在表头行中查找列名「索引」的下标；未找到返回 -1。
func indexColumnIndex(header []string) int {
	for i, c := range header {
		if typeHeaderCellMatch(c, typeHeaderIndexName) {
			return i
		}
	}
	return -1
}

func typeHeaderCellMatch(cell, want string) bool {
	t := strings.TrimSpace(strings.ReplaceAll(cell, "\u00a0", ""))
	return t == strings.TrimSpace(want)
}

// readTypeSheetCellAt 用工作表坐标读 @Type 的某一列（分组、索引等），合并单元格时与 Excel 一致。
func readTypeSheetCellAt(f *excelize.File, colIndex0, rowIndexInRows int) string {
	if f == nil || colIndex0 < 0 {
		return ""
	}
	col, err := excelize.ColumnNumberToName(colIndex0 + 1)
	if err != nil {
		return ""
	}
	ref, err := excelize.JoinCellName(col, rowIndexInRows+1)
	if err != nil {
		return ""
	}
	v, err := f.GetCellValue(typeSheetName, ref)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(v)
}

func cellAt(row []string, col int) string {
	if col < 0 || col >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[col])
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
