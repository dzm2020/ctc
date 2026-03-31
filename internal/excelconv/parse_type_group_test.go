package excelconv

import (
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestGroupColumnIndex(t *testing.T) {
	h := []string{"种类", "对象类型", "中文描述", "字段名", "字段类型", "数组切割", "默认值", "筛选", "分组"}
	if got := groupColumnIndex(h); got != 8 {
		t.Fatalf("got %d want 8", got)
	}
	h2 := []string{"种类", "对象类型", "中文描述", "字段名", "字段类型", "数组切割", "默认值", "筛选", "备注", "分组"}
	if got := groupColumnIndex(h2); got != 9 {
		t.Fatalf("got %d want 9", got)
	}
	if groupColumnIndex([]string{"种类", "对象类型", "中文描述", "字段名", "字段类型", "数组切割", "默认值", "筛选"}) != -1 {
		t.Fatal("want -1")
	}
}

func TestParseTypeSheetMergedGroupColumn(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	if err := f.SetSheetName("Sheet1", typeSheetName); err != nil {
		t.Fatal(err)
	}
	sheet := typeSheetName
	for c := 1; c <= len(typeHeader); c++ {
		ref, err := excelize.CoordinatesToCellName(c, 1)
		if err != nil {
			t.Fatal(err)
		}
		if err := f.SetCellValue(sheet, ref, typeHeader[c-1]); err != nil {
			t.Fatal(err)
		}
	}
	setRow := func(row int, name, typ string) {
		vals := []string{"表头", "Tmerge", "d", name, typ, "", "", "CS", ""}
		for c, v := range vals {
			ref, err := excelize.CoordinatesToCellName(c+1, row)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.SetCellValue(sheet, ref, v); err != nil {
				t.Fatal(err)
			}
		}
	}
	setRow(2, "f1", "string")
	setRow(3, "f2", "int")
	setRow(4, "f3", "int64")
	refI2, err := excelize.CoordinatesToCellName(9, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.SetCellValue(sheet, refI2, "media"); err != nil {
		t.Fatal(err)
	}
	refI4, err := excelize.CoordinatesToCellName(9, 4)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.MergeCell(sheet, refI2, refI4); err != nil {
		t.Fatal(err)
	}

	s, err := ParseTypeSheet(f)
	if err != nil {
		t.Fatal(err)
	}
	fs := s.Tables["Tmerge"]
	if len(fs) != 3 {
		t.Fatalf("fields len %d", len(fs))
	}
	for _, fld := range fs {
		if fld.Group != "media" {
			t.Fatalf("field %s group %q want media", fld.Name, fld.Group)
		}
	}
}
