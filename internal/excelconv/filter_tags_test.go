package excelconv

import "testing"

func TestFieldVisibleIntersection(t *testing.T) {
	if !FieldVisible("C", []string{"C", "S"}) {
		t.Fatal("C field with C,S export")
	}
	if FieldVisible("S", []string{"C"}) {
		t.Fatal("S field should hide when only C")
	}
	if !FieldVisible("C,S", []string{"S"}) {
		t.Fatal("C,S should match S")
	}
	if !FieldVisible("", []string{"C"}) {
		t.Fatal("empty filter = all")
	}
	if !FieldVisible("CS", []string{"C"}) {
		t.Fatal("CS legacy expands to C")
	}
}

func TestFieldVisibleCustomTagGM(t *testing.T) {
	// 任意自定义标签（如 gm）与 C/S 一样：配置与字段需有交集
	if !FieldVisible("gm", []string{"GM"}) {
		t.Fatal("gm field should match GM export tag")
	}
	if !FieldVisible("GM,Client", []string{"gm"}) {
		t.Fatal("comma-separated should include GM")
	}
	if FieldVisible("gm", []string{"C", "S"}) {
		t.Fatal("gm-only field must not match default C,S export")
	}
}

func TestResolveExportFilterTags(t *testing.T) {
	got, err := ResolveExportFilterTags([]string{"C", " S "})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "C" || got[1] != "S" {
		t.Fatalf("got %v", got)
	}
	got2, err := ResolveExportFilterTags([]string{"C,S"})
	if err != nil || len(got2) != 2 {
		t.Fatalf("got2 %v err %v", got2, err)
	}
	got3, err := ResolveExportFilterTags(nil)
	if err != nil || len(got3) != 2 || got3[0] != "C" || got3[1] != "S" {
		t.Fatalf("empty filterTags should default to C,S, got %v", got3)
	}
	got3b, err := ResolveExportFilterTags([]string{})
	if err != nil || len(got3b) != 2 {
		t.Fatalf("got3b %v", got3b)
	}
	gotClient, err := ResolveExportFilterTags([]string{"C"})
	if err != nil || len(gotClient) != 1 || gotClient[0] != "C" {
		t.Fatalf("gotClient %v", gotClient)
	}
	got4, err := ResolveExportFilterTags([]string{"gm"})
	if err != nil || len(got4) != 1 || got4[0] != "GM" {
		t.Fatalf("got4 %v", got4)
	}
}
