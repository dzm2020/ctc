package csharpgen

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

var csharpKeywords = map[string]struct{}{
	"abstract": {}, "as": {}, "base": {}, "bool": {}, "break": {}, "byte": {},
	"case": {}, "catch": {}, "char": {}, "checked": {}, "class": {}, "const": {},
	"continue": {}, "decimal": {}, "default": {}, "delegate": {}, "do": {}, "double": {},
	"else": {}, "enum": {}, "event": {}, "explicit": {}, "extern": {}, "false": {},
	"finally": {}, "fixed": {}, "float": {}, "for": {}, "foreach": {}, "goto": {},
	"if": {}, "implicit": {}, "in": {}, "int": {}, "interface": {}, "internal": {},
	"is": {}, "lock": {}, "long": {}, "namespace": {}, "new": {}, "null": {},
	"object": {}, "operator": {}, "out": {}, "override": {}, "params": {},
	"private": {}, "protected": {}, "public": {}, "readonly": {}, "ref": {},
	"return": {}, "sbyte": {}, "sealed": {}, "short": {}, "sizeof": {},
	"stackalloc": {}, "static": {}, "string": {}, "struct": {}, "switch": {},
	"this": {}, "throw": {}, "true": {}, "try": {}, "typeof": {}, "uint": {},
	"ulong": {}, "unchecked": {}, "unsafe": {}, "ushort": {}, "using": {},
	"virtual": {}, "void": {}, "volatile": {}, "while": {},
}

func isCSharpKeyword(lower string) bool {
	_, ok := csharpKeywords[strings.ToLower(lower)]
	return ok
}

// privateFieldSlug 与 Go gogen 一致，用于文件名等。
func privateFieldSlug(schemaName string) string {
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
	if isCSharpKeyword(lowerFirst) {
		return lowerFirst + "Field"
	}
	return lowerFirst
}

// csharpPublicName 策划字段名 -> C# 公共属性名（与 Go exported 规则一致）。
func csharpPublicName(name string) string {
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

// csharpSafeProp 合法 C# 属性标识符（关键字加 @）。
func csharpSafeProp(name string) string {
	p := csharpPublicName(name)
	if isCSharpKeyword(p) {
		return "@" + p
	}
	return p
}

func tableGroupRecordName(tableName, groupKey string) string {
	return csharpPublicName(tableName) + "RowGrp_" + privateFieldSlug(groupKey)
}

func tableIndexRecordName(tableName, indexKey string) string {
	return csharpPublicName(tableName) + "RowIdx_" + privateFieldSlug(indexKey)
}

func groupStaticCtorName(tableName, groupKey string) string {
	return csharpPublicName(tableName) + "_" + privateFieldSlug(groupKey)
}

func viewAsGroupMethodName(groupKey string) string {
	p := privateFieldSlug(groupKey)
	if p == "" {
		return "ViewAsGroup"
	}
	r, n := utf8.DecodeRuneInString(p)
	if r == utf8.RuneError {
		return "ViewAsGroup"
	}
	return "ViewAs" + string(unicode.ToUpper(r)) + p[n:]
}

func viewAsIndexMethodName(indexKey string, numIndex int) string {
	if numIndex <= 1 {
		return "ViewAsIndex"
	}
	p := privateFieldSlug(indexKey)
	if p == "" {
		return "ViewAsIndex"
	}
	r, n := utf8.DecodeRuneInString(p)
	if r == utf8.RuneError {
		return "ViewAsIndex"
	}
	return "ViewAsIndex" + string(unicode.ToUpper(r)) + p[n:]
}

func gameDataFieldName(tableName string) string {
	return csharpPublicName(tableName)
}
