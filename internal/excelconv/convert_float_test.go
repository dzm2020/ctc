package excelconv

import "testing"

func TestParseScalarFloat64(t *testing.T) {
	s := NewSchema()
	v, err := parseScalar("float64", "3.14", s)
	if err != nil {
		t.Fatal(err)
	}
	if f, ok := v.(float64); !ok || f < 3.13 || f > 3.15 {
		t.Fatalf("got %v (%T)", v, v)
	}
	v2, err := parseScalar("float64", "-2.5e2", s)
	if err != nil {
		t.Fatal(err)
	}
	if f := v2.(float64); f != -250 {
		t.Fatalf("got %v", f)
	}
}

func TestZeroForFieldFloat64(t *testing.T) {
	s := NewSchema()
	z := zeroForField(Field{Type: "float64"}, s)
	if z.(float64) != 0 {
		t.Fatal(z)
	}
}
