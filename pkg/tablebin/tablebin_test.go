package tablebin

import (
	"path/filepath"
	"testing"
)

func TestEncodeDecodeRoundtrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "t.bin")
	idKind := IDInt64
	cols := []Column{
		{Key: "Name", Kind: KindString},
		{Key: "Hp", Kind: KindInt},
		{Key: "Ratio", Kind: KindFloat64},
	}
	rows := []map[string]interface{}{
		{"id": int64(1), "Name": "a", "Hp": 10, "Ratio": 1.5},
		{"id": int64(2), "Name": "b", "Hp": -3, "Ratio": 0.0},
	}
	if err := EncodeFile(p, "id", idKind, cols, rows); err != nil {
		t.Fatal(err)
	}
	dec, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	if dec.NumRows() != 2 {
		t.Fatalf("rows %d", dec.NumRows())
	}
	id1, err := dec.ReadInt64Zigzag()
	if err != nil || id1 != 1 {
		t.Fatalf("id1 %v %v", id1, err)
	}
	s1, err := dec.ReadString()
	if err != nil || s1 != "a" {
		t.Fatalf("s1 %q %v", s1, err)
	}
	hp1, err := dec.ReadInt()
	if err != nil || hp1 != 10 {
		t.Fatalf("hp1 %v", hp1)
	}
	f1, err := dec.ReadFloat64()
	if err != nil || f1 != 1.5 {
		t.Fatalf("f1 %v", f1)
	}
	id2, err := dec.ReadInt64Zigzag()
	if err != nil || id2 != 2 {
		t.Fatalf("id2 %v", id2)
	}
	s2, err := dec.ReadString()
	if err != nil || s2 != "b" {
		t.Fatalf("s2 %q", s2)
	}
	hp2, err := dec.ReadInt()
	if err != nil || hp2 != -3 {
		t.Fatalf("hp2 %v", hp2)
	}
	f2, err := dec.ReadFloat64()
	if err != nil || f2 != 0 {
		t.Fatalf("f2 %v", f2)
	}
	if err := dec.ErrIfTrailing(); err != nil {
		t.Fatal(err)
	}
}

func TestBinStructFlattenRoundtrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "s.bin")
	cols := []Column{
		{Key: "Item", SubPath: "ID", Kind: KindInt},
		{Key: "Item", SubPath: "Num", Kind: KindInt},
	}
	rows := []map[string]interface{}{
		{"id": int64(1), "Item": map[string]interface{}{"ID": 10002, "Num": 60}},
	}
	if err := EncodeFile(p, "id", IDInt64, cols, rows); err != nil {
		t.Fatal(err)
	}
	dec, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dec.ReadInt64Zigzag(); err != nil {
		t.Fatal(err)
	}
	a, err := dec.ReadInt()
	if err != nil || a != 10002 {
		t.Fatalf("ID %v %v", a, err)
	}
	b, err := dec.ReadInt()
	if err != nil || b != 60 {
		t.Fatalf("Num %v %v", b, err)
	}
	if err := dec.ErrIfTrailing(); err != nil {
		t.Fatal(err)
	}
}

func TestStringPoolDedup(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "d.bin")
	cols := []Column{{Key: "A", Kind: KindString}, {Key: "B", Kind: KindString}}
	rows := []map[string]interface{}{
		{"id": int64(1), "A": "x", "B": "x"},
	}
	if err := EncodeFile(p, "id", IDInt64, cols, rows); err != nil {
		t.Fatal(err)
	}
	dec, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = dec.ReadInt64Zigzag()
	a, _ := dec.ReadString()
	b, _ := dec.ReadString()
	if a != "x" || b != "x" {
		t.Fatal(a, b)
	}
}
