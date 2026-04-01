package gogen

import (
	"fmt"
	"strings"

	"ctc/internal/excelconv"
)

// groupQuerySwitchTmpl 供不可比较分组的查询方法生成 ParseLines / KeyJoinElts（keyParts…）。
type groupQuerySwitchTmpl struct {
	GroupName   string
	MapSuffix   string
	GroupType   string
	Comparable  bool
	N           int
	ParseLines  []string
	StructLines []string
	KeyJoinElts []string // 非可比较键：strings.Join 的元素表达式

	queryUsesStrconv bool // 供生成器决定是否 import strconv（无模板字段）
}

func buildGroupQuerySwitch(
	g string,
	gf []excelconv.Field,
	schema *excelconv.Schema,
	mapSuffix, groupType string,
	comparable bool,
) groupQuerySwitchTmpl {
	q := groupQuerySwitchTmpl{
		GroupName:  g,
		MapSuffix:  mapSuffix,
		GroupType:  groupType,
		Comparable: comparable,
		N:          len(gf),
	}
	if comparable {
		for i, fld := range gf {
			priv := privateFieldIdent(fld.Name)
			got := goFieldTypeTable(fld, schema)
			switch got {
			case "string":
				q.StructLines = append(q.StructLines, fmt.Sprintf("%s: keyParts[%d]", priv, i))
			case "int":
				v := fmt.Sprintf("_gk%d", i)
				q.ParseLines = append(q.ParseLines,
					fmt.Sprintf("%s, _err := strconv.Atoi(keyParts[%d])", v, i),
					"if _err != nil { return nil }",
				)
				q.StructLines = append(q.StructLines, fmt.Sprintf("%s: %s", priv, v))
			case "int64":
				v := fmt.Sprintf("_gk%d", i)
				q.ParseLines = append(q.ParseLines,
					fmt.Sprintf("%s, _err := strconv.ParseInt(keyParts[%d], 10, 64)", v, i),
					"if _err != nil { return nil }",
				)
				q.StructLines = append(q.StructLines, fmt.Sprintf("%s: %s", priv, v))
			case "float64":
				v := fmt.Sprintf("_gk%d", i)
				q.ParseLines = append(q.ParseLines,
					fmt.Sprintf("%s, _err := strconv.ParseFloat(keyParts[%d], 64)", v, i),
					"if _err != nil { return nil }",
				)
				q.StructLines = append(q.StructLines, fmt.Sprintf("%s: %s", priv, v))
			default:
				if schema != nil && schema.Enums[fld.Type] != nil {
					v := fmt.Sprintf("_gk%d", i)
					q.ParseLines = append(q.ParseLines,
						fmt.Sprintf("%s, _err := strconv.Atoi(keyParts[%d])", v, i),
						"if _err != nil { return nil }",
					)
					q.StructLines = append(q.StructLines,
						fmt.Sprintf("%s: %s(%s)", priv, got, v))
				} else {
					q.StructLines = append(q.StructLines, fmt.Sprintf("%s: keyParts[%d]", priv, i))
				}
			}
		}
		q.markStrconv()
		return q
	}
	for i, fld := range gf {
		got := goFieldTypeTable(fld, schema)
		switch got {
		case "string":
			q.KeyJoinElts = append(q.KeyJoinElts, fmt.Sprintf("keyParts[%d]", i))
		case "int":
			v := fmt.Sprintf("_gs%d", i)
			q.ParseLines = append(q.ParseLines,
				fmt.Sprintf("%s, _err := strconv.Atoi(keyParts[%d])", v, i),
				"if _err != nil { return nil }",
			)
			q.KeyJoinElts = append(q.KeyJoinElts, fmt.Sprintf("strconv.Itoa(%s)", v))
		case "int64":
			v := fmt.Sprintf("_gs%d", i)
			q.ParseLines = append(q.ParseLines,
				fmt.Sprintf("%s, _err := strconv.ParseInt(keyParts[%d], 10, 64)", v, i),
				"if _err != nil { return nil }",
			)
			q.KeyJoinElts = append(q.KeyJoinElts, fmt.Sprintf("strconv.FormatInt(%s, 10)", v))
		case "float64":
			v := fmt.Sprintf("_gs%d", i)
			q.ParseLines = append(q.ParseLines,
				fmt.Sprintf("%s, _err := strconv.ParseFloat(keyParts[%d], 64)", v, i),
				"if _err != nil { return nil }",
			)
			q.KeyJoinElts = append(q.KeyJoinElts, fmt.Sprintf("strconv.FormatFloat(%s, 'g', -1, 64)", v))
		default:
			if schema != nil && schema.Enums[fld.Type] != nil {
				v := fmt.Sprintf("_gs%d", i)
				q.ParseLines = append(q.ParseLines,
					fmt.Sprintf("%s, _err := strconv.Atoi(keyParts[%d])", v, i),
					"if _err != nil { return nil }",
				)
				q.KeyJoinElts = append(q.KeyJoinElts,
					fmt.Sprintf("strconv.FormatInt(int64(%s(%s)), 10)", got, v))
			} else if strings.HasPrefix(got, "[]") {
				q.KeyJoinElts = append(q.KeyJoinElts, fmt.Sprintf("keyParts[%d]", i))
			} else {
				q.KeyJoinElts = append(q.KeyJoinElts, fmt.Sprintf("keyParts[%d]", i))
			}
		}
	}
	q.markStrconv()
	return q
}

func (q *groupQuerySwitchTmpl) markStrconv() {
	for _, l := range q.ParseLines {
		if strings.Contains(l, "strconv.") {
			q.queryUsesStrconv = true
			return
		}
	}
}

func (q *groupQuerySwitchTmpl) QueryNeedsStrconv() bool {
	return q.queryUsesStrconv
}
