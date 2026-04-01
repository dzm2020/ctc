// RegenTBTest 根据 tables/Test.xlsx 中 @Type 的 TBTest 定义，重写 TBTest 工作表表头与示例数据行。
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xuri/excelize/v2"
)

func main() {
	root := findRoot()
	xlsxPath := filepath.Join(root, "tables", "Test.xlsx")
	f, err := excelize.OpenFile(xlsxPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()

	const sheet = "TBTest"
	rows, err := f.GetRows(sheet)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// 从底部向上删数据行，保留第 1 行位置给新表头
	for r := len(rows); r >= 2; r-- {
		if err := f.RemoveRow(sheet, r); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	header := []interface{}{
		"ArrayDict", "#备注", "Field1", "Field2", "Field3", "Field4",
		"Field5", "Field6", "Field7", "Field8", "Field9", "Field10", "Field11", "Field12",
	}
	if err := f.SetSheetRow(sheet, "A1", &header); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// 主键 id 唯一；复合索引 idx1(Field3,Field4) 每组唯一；分组 group1(Field1,Field2) 允许多行相同。
	// 注意：带「|」切割的列勿留空单元格，否则 excelconv 会把空段变成嵌套 []interface{}，binary 编码失败；用单元素 "0" 等占位。
	data := [][]string{
		{"1", "标量+数组+枚举+结构", "1001", "alpha", "1.5", "1", "10|20", "x|y", "0.5|1.5", "1|2", "enum1", "enum1|enum2", `{"ID":1,"Num":100}`, "ID:2,数量:200|ID:3,数量:300"},
		{"2", "同组另一行", "1001", "alpha", "2.5", "2", "0", "a", "0", "0", "enum2", "enum2", "ID:1,数量:1", "ID:1,数量:0"},
		{"3", "另一组", "2002", "beta", "3.25", "3", "100", "single", "2.0", "42", "enum3", "enum3", "ID:10,数量:20", "ID:11,数量:22"},
		{"4", "浮点与整型索引", "2002", "beta", "4.0", "4", "1|2|3", "a|b|c", "1|2|3", "10|20|30", "enum1", "enum2|enum3", "ID:0,数量:0", "ID:1,数量:1"},
		{"5", "KV 结构简写", "3003", "gamma", "5.5", "5", "0", "b", "0", "0", "enum2", "enum1", "ID:100,数量:5", "ID:200,数量:6|ID:201,数量:7"},
		{"6", "单元素数组占位", "3003", "gamma", "6.75", "6", "0", "c", "0", "0", "enum1", "enum3", "ID:0,数量:0", "ID:0,数量:0"},
		{"7", "大整数", "9223372036854775807", "max", "7.0", "7", "0", "d", "0.0", "0", "enum3", "enum1|enum2|enum3", `{"ID":0,"Num":0}`, "ID:0,数量:0"},
		{"8", "负浮点", "-1", "neg", "-0.25", "8", "-1|-2", "m|n", "-1.5|-2.5", "-3|-4", "enum1", "enum1", `{"ID":-1,"Num":-2}`, "ID:5,数量:0"},
	}

	for i, row := range data {
		excelRow := i + 2
		for j, val := range row {
			colName, err := excelize.ColumnNumberToName(j + 1)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			cell := fmt.Sprintf("%s%d", colName, excelRow)
			if err := f.SetCellValue(sheet, cell, val); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	}

	if err := f.Save(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("已重写", xlsxPath, "工作表 TBTest：", len(data), "行数据")
}

func findRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	dir := wd
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	return wd
}
