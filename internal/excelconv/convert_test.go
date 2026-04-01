package excelconv

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestWallpaperSample(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(dir, "..", "..")
	matches, err := filepath.Glob(filepath.Join(root, "*.xlsx"))
	if err != nil || len(matches) == 0 {
		t.Skip("no xlsx in repo root")
	}
	f, err := excelize.OpenFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	schema, err := ParseTypeSheet(f)
	if err != nil {
		t.Fatal(err)
	}
	tables, err := ConvertWorkbook(f, schema, []string{"C", "S"})
	if err != nil {
		t.Fatal(err)
	}
	w, ok := tables["Wallpaper"]
	if !ok {
		t.Fatal("missing Wallpaper table")
	}
	row, ok := w["1"]
	if !ok {
		t.Fatal("missing id 1")
	}
	if id := row["id"].(int64); id != 1 {
		t.Fatalf("id = %v want int64 1", id)
	}
	if row["Type"].(int) != 1 {
		t.Fatalf("Type = %v", row["Type"])
	}
	if row["Cover"].(string) == "" {
		t.Fatal("Cover empty")
	}
	b, _ := json.Marshal(w["2"])
	t.Logf("row 2: %s", b)
}
