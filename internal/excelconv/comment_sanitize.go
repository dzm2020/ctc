package excelconv

import "strings"

// SanitizeOneLineComment 将 @Type「中文描述」等压成单行，便于写入 // 或 /// 注释（去换行、折叠空白）。
func SanitizeOneLineComment(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.Join(strings.Fields(s), " ")
}
