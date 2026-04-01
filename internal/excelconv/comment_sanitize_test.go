package excelconv

import "testing"

func TestSanitizeOneLineComment(t *testing.T) {
	if got := SanitizeOneLineComment(""); got != "" {
		t.Fatalf("empty: %q", got)
	}
	if got := SanitizeOneLineComment("  中文  "); got != "中文" {
		t.Fatalf("trim: %q", got)
	}
	if got := SanitizeOneLineComment("a\nb\tc"); got != "a b c" {
		t.Fatalf("newline: %q", got)
	}
}
