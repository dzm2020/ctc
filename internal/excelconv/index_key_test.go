package excelconv

import "testing"

func TestRowIndexKeyString(t *testing.T) {
	schema := NewSchema()
	fields := []Field{
		{Name: "a", Type: "string"},
		{Name: "b", Type: "int"},
	}
	rec := map[string]interface{}{
		"id": int64(1),
		"a":  "x",
		"b":  float64(2),
	}
	k, err := RowIndexKeyString(rec, fields, schema)
	if err != nil {
		t.Fatal(err)
	}
	k2, err := RowIndexKeyString(rec, fields, schema)
	if err != nil {
		t.Fatal(err)
	}
	if k != k2 {
		t.Fatalf("unstable key %q vs %q", k, k2)
	}
}
