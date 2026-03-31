package excelconv

import (
	"reflect"
	"testing"
)

func TestDistinctFieldGroups(t *testing.T) {
	vis := []Field{
		{Name: "a", Group: ""},
		{Name: "b", Group: "g1"},
		{Name: "c", Group: "g1"},
		{Name: "d", Group: "g2"},
	}
	got := DistinctFieldGroups(vis)
	want := []string{"g1", "g2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DistinctFieldGroups: got %v want %v", got, want)
	}
}
