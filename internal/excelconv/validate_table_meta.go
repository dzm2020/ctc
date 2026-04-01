package excelconv

import (
	"fmt"
	"strings"
)

// ValidateGroupIndexNamesDisjoint 同一张表中 @Type「分组」与「索引」不能使用相同名称（避免 Go 构造函数重名）。
func ValidateGroupIndexNamesDisjoint(table string, fields []Field) error {
	seenG := make(map[string]bool)
	seenI := make(map[string]bool)
	for _, f := range fields {
		if g := strings.TrimSpace(f.Group); g != "" {
			seenG[g] = true
		}
		if ix := strings.TrimSpace(f.Index); ix != "" {
			seenI[ix] = true
		}
	}
	for name := range seenG {
		if seenI[name] {
			return fmt.Errorf("@Type 表 %q: 分组名与索引名不能相同 %q", table, name)
		}
	}
	return nil
}
