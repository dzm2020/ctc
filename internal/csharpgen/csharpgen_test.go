package csharpgen

import (
	"os"
	"path/filepath"
	"testing"

	"ctc/internal/excelconv"
)

func TestWritePackageMinimal(t *testing.T) {
	dir := t.TempDir()
	s := excelconv.NewSchema()
	s.Tables["Demo"] = []excelconv.Field{
		{Name: "Name", Type: "string"},
	}
	s.TableIDType["Demo"] = "int64"
	if err := WritePackage(dir, "TestGameData", s, nil, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "GameData.csproj")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "Table_demo.gen.cs")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "GameData.gen.cs")); err != nil {
		t.Fatal(err)
	}
}
