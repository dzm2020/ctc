package gogen

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// goKeywords 为 Go 关键字；用于生成合法的小写未导出字段名。
var goKeywords = map[string]struct{}{
	"break": {}, "default": {}, "func": {}, "interface": {}, "select": {},
	"case": {}, "defer": {}, "go": {}, "map": {}, "struct": {},
	"chan": {}, "else": {}, "goto": {}, "package": {}, "switch": {},
	"const": {}, "fallthrough": {}, "if": {}, "range": {}, "type": {},
	"continue": {}, "for": {}, "import": {}, "return": {}, "var": {},
}

func isGoKeyword(lower string) bool {
	_, ok := goKeywords[lower]
	return ok
}

// privateFieldIdent 策划字段名 -> 合法的小写未导出 Go 字段标识符。
func privateFieldIdent(schemaName string) string {
	name := strings.TrimSpace(schemaName)
	if name == "" {
		return "x"
	}
	r, n := utf8.DecodeRuneInString(name)
	if r == utf8.RuneError {
		return "x" + strings.ToLower(name)
	}
	if !unicode.IsLetter(r) && r != '_' {
		return "x" + strings.ToLower(name)
	}
	lowerFirst := string(unicode.ToLower(r)) + name[n:]
	if isGoKeyword(lowerFirst) {
		return lowerFirst + "Field"
	}
	return lowerFirst
}

// getterMethodName 策划字段名 -> Go Getter 方法名，如 Cover -> GetCover，id -> GetID。
func getterMethodName(schemaName string) string {
	s := strings.TrimSpace(schemaName)
	if strings.EqualFold(s, "id") {
		return "GetID"
	}
	r, n := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return "GetX"
	}
	return "Get" + string(unicode.ToUpper(r)) + s[n:]
}

func tableGroupTypeIdent(tableName, groupJSONKey string) string {
	return tableName + "_rowGrp_" + privateFieldIdent(groupJSONKey)
}

func viewAsGroupMethodName(groupKey string) string {
	p := privateFieldIdent(groupKey)
	if p == "" {
		return "ViewAsGroup"
	}
	r, n := utf8.DecodeRuneInString(p)
	if r == utf8.RuneError {
		return "ViewAsGroup"
	}
	return "ViewAs" + string(unicode.ToUpper(r)) + p[n:]
}

func rowJSONAuxTypeName(tableName string) string {
	return "rowJSONAux_" + tableName
}

func structJSONAuxTypeName(structName string) string {
	return "structJSONAux_" + structName
}

// gameDataFieldName GameData 中表容器字段名（导出）。
func gameDataFieldName(tableName string) string {
	return exportedGoIdent(tableName)
}

func constName(enum, member string) string {
	runes := []rune(strings.TrimSpace(member))
	if len(runes) > 0 && unicode.IsLower(runes[0]) {
		runes[0] = unicode.ToUpper(runes[0])
	}
	return enum + string(runes)
}

func exportedGoIdent(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "X"
	}
	r, n := utf8.DecodeRuneInString(name)
	if r == utf8.RuneError {
		return "X" + name
	}
	if unicode.IsLetter(r) || r == '_' {
		if unicode.IsLower(r) {
			return string(unicode.ToUpper(r)) + name[n:]
		}
		return name
	}
	return "X" + strings.ToUpper(name[:1]) + name[1:]
}
