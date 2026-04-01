package excelconv

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestParseStructCellKVItemConfig(t *testing.T) {
	s := NewSchema()
	s.Structs["ItemConfig"] = []StructField{
		{Struct: "ItemConfig", NameCN: "ID", Name: "ID", Type: "int"},
		{Struct: "ItemConfig", NameCN: "数量", Name: "Num", Type: "int"},
	}
	raw := "ID:10002,数量:60"
	m, err := parseStructCell("ItemConfig", raw, s)
	if err != nil {
		t.Fatal(err)
	}
	if m["ID"].(int) != 10002 {
		t.Fatalf("ID=%v", m["ID"])
	}
	if m["Num"].(int) != 60 {
		t.Fatalf("Num=%v", m["Num"])
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	// encoding/json 对 map 键排序；写出键为 Name，与生成 Go 的 json 标签一致
	if string(b) != `{"ID":10002,"Num":60}` {
		t.Fatalf("json: %s", b)
	}
}

func TestParseTypedefJSONObject(t *testing.T) {
	tests := []struct {
		raw string
		min int // min len keys
	}{
		{`{"a":1,"b":"x"}`, 2},
		{`{}`, 0},
		{``, 0},
		{`null`, 0},
		{"\ufeff{\"k\":2}", 1},
	}
	for _, tc := range tests {
		m, err := parseTypedefJSONObject("S", tc.raw)
		if err != nil {
			t.Fatalf("%q: %v", tc.raw, err)
		}
		if len(m) < tc.min {
			t.Fatalf("%q: got %v", tc.raw, m)
		}
	}
	// 整格为 JSON 字符串，内层才是对象（Excel 常见）
	m, err := parseTypedefJSONObject("S", `"{\"w\":10,\"h\":20}"`)
	if err != nil {
		t.Fatal(err)
	}
	if m["w"].(float64) != 10 || m["h"].(float64) != 20 {
		t.Fatalf("unwrap string json: %v", m)
	}
}

func TestZeroForFieldStructAndArray(t *testing.T) {
	s := NewSchema()
	s.Structs["Box"] = []StructField{{Struct: "Box", Name: "w", Type: "int"}}
	f := Field{Name: "box", Type: "Box"}
	z := zeroForField(f, s)
	if _, ok := z.(map[string]interface{}); !ok {
		t.Fatalf("struct zero want map, got %T", z)
	}
	f2 := Field{Name: "boxes", Type: "Box", ArraySplit: ","}
	z2 := zeroForField(f2, s)
	arr, ok := z2.([]interface{})
	if !ok || len(arr) != 0 {
		t.Fatalf("struct[] zero want empty []interface{}, got %T %v", z2, z2)
	}
	b, _ := json.Marshal(z2)
	if string(b) != "[]" {
		t.Fatalf("json: %s", b)
	}
}

func TestValueToIndexKeyPartStructStable(t *testing.T) {
	s := NewSchema()
	s.Structs["Box"] = []StructField{{Name: "b", Type: "int"}, {Name: "a", Type: "int"}}
	f := Field{Name: "box", Type: "Box"}
	// 故意逆序插入 map，Marshal 应稳定排序键
	m := map[string]interface{}{"b": 2, "a": 1}
	k1, err := valueToIndexKeyPart(m, f, s)
	if err != nil {
		t.Fatal(err)
	}
	m2 := map[string]interface{}{"a": 1, "b": 2}
	k2, err := valueToIndexKeyPart(m2, f, s)
	if err != nil {
		t.Fatal(err)
	}
	if k1 != k2 {
		t.Fatalf("index key unstable: %q vs %q", k1, k2)
	}
	var back map[string]interface{}
	if err := json.Unmarshal([]byte(k1), &back); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back, map[string]interface{}{"a": float64(1), "b": float64(2)}) {
		t.Fatalf("roundtrip: %v", back)
	}
}
