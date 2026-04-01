package excelconv

import "testing"

func TestIndexColumnIndex(t *testing.T) {
	h := []string{"种类", "对象类型", "中文描述", "字段名", "字段类型", "数组切割", "默认值", "筛选", "分组", "索引"}
	if got := indexColumnIndex(h); got != 9 {
		t.Fatalf("indexColumnIndex: got %d want 9", got)
	}
	h2 := []string{"种类", "对象类型", "中文描述", "字段名", "字段类型", "数组切割", "默认值", "筛选", "索引", "备注", "分组"}
	if got := indexColumnIndex(h2); got != 8 {
		t.Fatalf("indexColumnIndex: got %d want 8", got)
	}
	if indexColumnIndex([]string{"种类", "对象类型", "中文描述", "字段名", "字段类型", "数组切割", "默认值", "筛选"}) != -1 {
		t.Fatal("want -1")
	}
}

func TestDistinctFieldIndexes(t *testing.T) {
	vis := []Field{
		{Name: "a", Index: ""},
		{Name: "b", Index: "u1"},
		{Name: "c", Index: "u1"},
		{Name: "d", Index: "u2"},
	}
	got := DistinctFieldIndexes(vis)
	want := []string{"u1", "u2"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestValidateGroupIndexNamesDisjoint(t *testing.T) {
	err := ValidateGroupIndexNamesDisjoint("T", []Field{
		{Name: "a", Group: "g", Index: ""},
		{Name: "b", Group: "", Index: "g"},
	})
	if err == nil {
		t.Fatal("expected error when group and index name collide")
	}
	if err := ValidateGroupIndexNamesDisjoint("T", []Field{
		{Name: "a", Group: "g1", Index: "i1"},
	}); err != nil {
		t.Fatal(err)
	}
}
