package excelconv

import "testing"

func TestValidateSchemaRulesEnumDuplicateValue(t *testing.T) {
	s := NewSchema()
	s.Enums["E"] = []EnumMember{
		{Name: "A", Value: "1"},
		{Name: "B", Value: "1"},
	}
	if err := ValidateSchemaRules(s); err == nil {
		t.Fatal("expected duplicate enum value error")
	}
}

func TestValidateSchemaRulesEnumEmptyDefault(t *testing.T) {
	s := NewSchema()
	s.Enums["E"] = []EnumMember{
		{Name: "A", Value: ""},
	}
	if err := ValidateSchemaRules(s); err == nil {
		t.Fatal("expected empty default error")
	}
}

func TestValidateSchemaRulesArrayInIndex(t *testing.T) {
	s := NewSchema()
	s.Tables["T"] = []Field{
		{Name: "x", Type: "string", ArraySplit: "|", Index: "i1"},
	}
	if err := ValidateSchemaRules(s); err == nil {
		t.Fatal("expected array+index error")
	}
}
