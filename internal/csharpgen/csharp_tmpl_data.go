package csharpgen

import (
	"fmt"
	"sort"
	"strings"

	"ctc/internal/excelconv"
)

// --- shared small payloads ---

type csharpNamespaceData struct {
	Namespace string
}

// --- enums ---

type csharpEnumMemberTmpl struct {
	ConstName string
	Value     int
	Last      bool
}

type csharpEnumTmpl struct {
	Name    string
	Members []csharpEnumMemberTmpl
}

type csharpEnumsFileTmpl struct {
	Namespace string
	Enums     []csharpEnumTmpl
}

func buildCSharpEnumsFileData(ns string, schema *excelconv.Schema) csharpEnumsFileTmpl {
	var names []string
	for n := range schema.Enums {
		names = append(names, n)
	}
	sort.Strings(names)
	var enums []csharpEnumTmpl
	for _, en := range names {
		members := schema.Enums[en]
		var ms []csharpEnumMemberTmpl
		for i, m := range members {
			v := 0
			if schema.EnumValue[en] != nil {
				v = schema.EnumValue[en][m.Name]
			}
			ms = append(ms, csharpEnumMemberTmpl{
				ConstName: csharpPublicName(m.Name),
				Value:     v,
				Last:      i == len(members)-1,
			})
		}
		enums = append(enums, csharpEnumTmpl{Name: csharpPublicName(en), Members: ms})
	}
	return csharpEnumsFileTmpl{Namespace: ns, Enums: enums}
}

// --- structs ---

type csharpStructFieldLineTmpl struct {
	JsonName string
	CsType   string
	Prop     string
}

type csharpStructOneTmpl struct {
	Name           string
	Fields         []csharpStructFieldLineTmpl
	ReadFromBody   string
}

type csharpStructsFileTmpl struct {
	Namespace string
	Binary    bool
	Structs   []csharpStructOneTmpl
}

func buildCSharpStructsFileData(ns string, schema *excelconv.Schema, exportTags []string, binary bool) csharpStructsFileTmpl {
	var names []string
	for n := range schema.Structs {
		names = append(names, n)
	}
	sort.Strings(names)
	var structs []csharpStructOneTmpl
	for _, sn := range names {
		vis := excelconv.VisibleStructFields(schema.Structs[sn], exportTags)
		var fields []csharpStructFieldLineTmpl
		for _, sf := range vis {
			fields = append(fields, csharpStructFieldLineTmpl{
				JsonName: sf.Name,
				CsType:   csharpTypeStruct(sf, schema),
				Prop:     csharpSafeProp(sf.Name),
			})
		}
		one := csharpStructOneTmpl{Name: csharpPublicName(sn), Fields: fields}
		if binary {
			one.ReadFromBody = emitStructReadFromBody(sn, schema, exportTags)
		}
		structs = append(structs, one)
	}
	return csharpStructsFileTmpl{Namespace: ns, Binary: binary, Structs: structs}
}

// --- GameData ---

type csharpGameDataTableTmpl struct {
	Var      string
	TableCls string
	FileName string
}

type csharpGameDataFileTmpl struct {
	Namespace       string
	DataExtComment  string
	Tables          []csharpGameDataTableTmpl
}

func buildCSharpGameDataFileData(ns string, schema *excelconv.Schema, binary bool) csharpGameDataFileTmpl {
	var tnames []string
	for n := range schema.Tables {
		tnames = append(tnames, n)
	}
	sort.Strings(tnames)
	ext := ".json"
	if binary {
		ext = ".bin"
	}
	comment := ext
	var tabs []csharpGameDataTableTmpl
	for _, tn := range tnames {
		tabs = append(tabs, csharpGameDataTableTmpl{
			Var:      gameDataFieldName(tn),
			TableCls: csharpPublicName(tn) + "Table",
			FileName: tn + ext,
		})
	}
	return csharpGameDataFileTmpl{Namespace: ns, DataExtComment: comment, Tables: tabs}
}

// --- csproj ---

type csharpCsprojTmpl struct {
	TargetFramework string
	RootNamespace   string
}

// --- table file ---

type csharpParamTmpl struct {
	CsType string
	Prop   string
}

type csharpRecordDefTmpl struct {
	Name   string
	Fields []csharpParamTmpl
}

type csharpRowFieldTmpl struct {
	JsonName string
	CsType   string
	Prop     string
}

type csharpViewAsTmpl struct {
	ReturnType string
	Method     string
	FieldRefs  []string
}

type csharpGroupIndexDictTmpl struct {
	Suffix     string
	Comparable bool
	RecordType string
}

type csharpStaticCtorTmpl struct {
	ResultType string
	FuncName   string
	Params     []csharpParamTmpl
	NewArgs    []string
}

type csharpGetRowsMethodTmpl struct {
	MethodName string
	RowName    string
	Comparable bool
	Params     []csharpParamTmpl
	FuncName   string
	Suffix     string
	Arity      int
}

type csharpGetIndexMethodTmpl struct {
	MethodName string
	RowName    string
	Comparable bool
	Params     []csharpParamTmpl
	FuncName   string
	Suffix     string
	Arity      int
}

type csharpTableFileTmpl struct {
	Namespace string

	ComparableGroupRecords []csharpRecordDefTmpl
	ComparableIndexRecords []csharpRecordDefTmpl

	RowName, TableCls, PkType, PkProp, IDJsonKey string
	RowFields                                  []csharpRowFieldTmpl
	ViewAsGroups, ViewAsIndexes                []csharpViewAsTmpl
	Binary                                     bool
	RowReadFromBody                            string

	GroupDicts         []csharpGroupIndexDictTmpl
	IndexDicts         []csharpGroupIndexDictTmpl
	StaticGroupCtors   []csharpStaticCtorTmpl
	StaticIndexCtors   []csharpStaticCtorTmpl
	GroupClearSuffixes []string
	IndexClearSuffixes []string
	InitForeachLines   []string
	LoadBinary         bool
	GetRowsMethods     []csharpGetRowsMethodTmpl
	GetIndexMethods    []csharpGetIndexMethodTmpl
}

func buildCSharpTableFileData(ns, tableName string, schema *excelconv.Schema, exportTags []string, binary bool) csharpTableFileTmpl {
	visible := excelconv.VisibleTableFields(schema.Tables[tableName], exportTags)
	rowName := csharpPublicName(tableName) + "Row"
	tableCls := csharpPublicName(tableName) + "Table"
	pk := csharpPrimaryKeyType(schema, tableName)
	pkProp := csharpSafeProp(excelconv.RowJSONIDKey)

	groups := excelconv.DistinctFieldGroups(visible)
	indexes := excelconv.DistinctFieldIndexes(visible)
	ni := len(indexes)

	data := csharpTableFileTmpl{
		Namespace:  ns,
		RowName:    rowName,
		TableCls:   tableCls,
		PkType:     pk,
		PkProp:     pkProp,
		IDJsonKey:  excelconv.RowJSONIDKey,
		Binary:     binary,
		LoadBinary: binary,
	}

	for _, g := range groups {
		gf := excelconv.FieldsInGroup(visible, g)
		if !groupFieldsComparable(gf, schema) {
			continue
		}
		rt := tableGroupRecordName(tableName, g)
		var fields []csharpParamTmpl
		for _, f := range gf {
			fields = append(fields, csharpParamTmpl{CsType: csharpTypeTable(f, schema), Prop: csharpSafeProp(f.Name)})
		}
		data.ComparableGroupRecords = append(data.ComparableGroupRecords, csharpRecordDefTmpl{Name: rt, Fields: fields})
	}
	for _, ix := range indexes {
		gf := excelconv.FieldsInIndex(visible, ix)
		if !groupFieldsComparable(gf, schema) {
			continue
		}
		rt := tableIndexRecordName(tableName, ix)
		var fields []csharpParamTmpl
		for _, f := range gf {
			fields = append(fields, csharpParamTmpl{CsType: csharpTypeTable(f, schema), Prop: csharpSafeProp(f.Name)})
		}
		data.ComparableIndexRecords = append(data.ComparableIndexRecords, csharpRecordDefTmpl{Name: rt, Fields: fields})
	}

	for _, f := range visible {
		data.RowFields = append(data.RowFields, csharpRowFieldTmpl{
			JsonName: f.Name,
			CsType:   csharpTypeTable(f, schema),
			Prop:     csharpSafeProp(f.Name),
		})
	}

	emit := excelconv.RowStructEmitOrder(visible)
	for _, p := range emit {
		if p.Group == "" {
			continue
		}
		gf := excelconv.FieldsInGroup(visible, p.Group)
		gtyp := tableGroupRecordName(tableName, p.Group)
		mth := viewAsGroupMethodName(p.Group)
		var refs []string
		for _, fld := range gf {
			refs = append(refs, csharpSafeProp(fld.Name))
		}
		data.ViewAsGroups = append(data.ViewAsGroups, csharpViewAsTmpl{ReturnType: gtyp, Method: mth, FieldRefs: refs})
	}
	for _, ix := range indexes {
		gf := excelconv.FieldsInIndex(visible, ix)
		ityp := tableIndexRecordName(tableName, ix)
		mth := viewAsIndexMethodName(ix, ni)
		var refs []string
		for _, fld := range gf {
			refs = append(refs, csharpSafeProp(fld.Name))
		}
		data.ViewAsIndexes = append(data.ViewAsIndexes, csharpViewAsTmpl{ReturnType: ityp, Method: mth, FieldRefs: refs})
	}

	if binary {
		data.RowReadFromBody = emitTableRowReadFrom(tableName, schema, visible, exportTags)
	}

	for _, g := range groups {
		gf := excelconv.FieldsInGroup(visible, g)
		suff := privateFieldSlug(g)
		ent := csharpGroupIndexDictTmpl{Suffix: suff, Comparable: groupFieldsComparable(gf, schema)}
		if ent.Comparable {
			ent.RecordType = tableGroupRecordName(tableName, g)
		}
		data.GroupDicts = append(data.GroupDicts, ent)
		data.GroupClearSuffixes = append(data.GroupClearSuffixes, suff)
	}
	for _, ix := range indexes {
		gf := excelconv.FieldsInIndex(visible, ix)
		suff := privateFieldSlug(ix)
		ent := csharpGroupIndexDictTmpl{Suffix: suff, Comparable: groupFieldsComparable(gf, schema)}
		if ent.Comparable {
			ent.RecordType = tableIndexRecordName(tableName, ix)
		}
		data.IndexDicts = append(data.IndexDicts, ent)
		data.IndexClearSuffixes = append(data.IndexClearSuffixes, suff)
	}

	for _, g := range groups {
		gf := excelconv.FieldsInGroup(visible, g)
		if !groupFieldsComparable(gf, schema) {
			continue
		}
		fn := groupStaticCtorName(tableName, g)
		rt := tableGroupRecordName(tableName, g)
		var params []csharpParamTmpl
		var args []string
		for _, f := range gf {
			params = append(params, csharpParamTmpl{CsType: csharpTypeTable(f, schema), Prop: csharpSafeProp(f.Name)})
			args = append(args, csharpSafeProp(f.Name))
		}
		data.StaticGroupCtors = append(data.StaticGroupCtors, csharpStaticCtorTmpl{ResultType: rt, FuncName: fn, Params: params, NewArgs: args})
	}
	for _, ix := range indexes {
		gf := excelconv.FieldsInIndex(visible, ix)
		if !groupFieldsComparable(gf, schema) {
			continue
		}
		fn := groupStaticCtorName(tableName, ix)
		rt := tableIndexRecordName(tableName, ix)
		var params []csharpParamTmpl
		var args []string
		for _, f := range gf {
			params = append(params, csharpParamTmpl{CsType: csharpTypeTable(f, schema), Prop: csharpSafeProp(f.Name)})
			args = append(args, csharpSafeProp(f.Name))
		}
		data.StaticIndexCtors = append(data.StaticIndexCtors, csharpStaticCtorTmpl{ResultType: rt, FuncName: fn, Params: params, NewArgs: args})
	}

	var initLines []string
	for i, g := range groups {
		gf := excelconv.FieldsInGroup(visible, g)
		suff := privateFieldSlug(g)
		fn := groupStaticCtorName(tableName, g)
		lv := fmt.Sprintf("_lg%d", i)
		if groupFieldsComparable(gf, schema) {
			vn := fmt.Sprintf("kg%d", i)
			var b strings.Builder
			b.WriteString(fmt.Sprintf("\t\t\tvar %s = %s(", vn, fn))
			for j, f := range gf {
				if j > 0 {
					b.WriteString(", ")
				}
				fmt.Fprintf(&b, "row.%s", csharpSafeProp(f.Name))
			}
			b.WriteString(");\n")
			b.WriteString(fmt.Sprintf("\t\t\tif (!_byGroup_%s.TryGetValue(%s, out var %s)) { %s = new List<%s>(); _byGroup_%s[%s] = %s; }\n", suff, vn, lv, lv, rowName, suff, vn, lv))
			b.WriteString(fmt.Sprintf("\t\t\t%s.Add(row);\n", lv))
			initLines = append(initLines, b.String())
		} else {
			expr := csGroupKeyJoinExpr(gf, schema, "row")
			var b strings.Builder
			b.WriteString(fmt.Sprintf("\t\t\tvar ks%d = %s;\n", i, expr))
			b.WriteString(fmt.Sprintf("\t\t\tif (!_byGroup_%s.TryGetValue(ks%d, out var %s)) { %s = new List<%s>(); _byGroup_%s[ks%d] = %s; }\n", suff, i, lv, lv, rowName, suff, i, lv))
			b.WriteString(fmt.Sprintf("\t\t\t%s.Add(row);\n", lv))
			initLines = append(initLines, b.String())
		}
	}
	for j, ix := range indexes {
		gf := excelconv.FieldsInIndex(visible, ix)
		suff := privateFieldSlug(ix)
		if groupFieldsComparable(gf, schema) {
			fn := groupStaticCtorName(tableName, ix)
			vn := fmt.Sprintf("ik%d", j)
			var b strings.Builder
			b.WriteString(fmt.Sprintf("\t\t\tvar %s = %s(", vn, fn))
			for k, f := range gf {
				if k > 0 {
					b.WriteString(", ")
				}
				fmt.Fprintf(&b, "row.%s", csharpSafeProp(f.Name))
			}
			b.WriteString(");\n")
			b.WriteString(fmt.Sprintf("\t\t\t_byIndex_%s[%s] = row;\n", suff, vn))
			initLines = append(initLines, b.String())
		} else {
			expr := csGroupKeyJoinExpr(gf, schema, "row")
			var b strings.Builder
			b.WriteString(fmt.Sprintf("\t\t\tvar kix%d = %s;\n", j, expr))
			b.WriteString(fmt.Sprintf("\t\t\t_byIndex_%s[kix%d] = row;\n", suff, j))
			initLines = append(initLines, b.String())
		}
	}
	data.InitForeachLines = initLines

	for _, g := range groups {
		gf := excelconv.FieldsInGroup(visible, g)
		suff := privateFieldSlug(g)
		mn := "GetRowsByGroup_" + suff
		if len(groups) == 1 {
			mn = "GetRowsByGroupKey"
		}
		fn := groupStaticCtorName(tableName, g)
		var params []csharpParamTmpl
		for _, f := range gf {
			params = append(params, csharpParamTmpl{CsType: csharpTypeTable(f, schema), Prop: csharpSafeProp(f.Name)})
		}
		data.GetRowsMethods = append(data.GetRowsMethods, csharpGetRowsMethodTmpl{
			MethodName: mn,
			RowName:    rowName,
			Comparable: groupFieldsComparable(gf, schema),
			Params:     params,
			FuncName:   fn,
			Suffix:     suff,
			Arity:      len(gf),
		})
	}

	for _, ix := range indexes {
		gf := excelconv.FieldsInIndex(visible, ix)
		suff := privateFieldSlug(ix)
		imn := "GetByIndex_" + suff
		if ni == 1 {
			imn = "GetByIndexKey"
		}
		fn := groupStaticCtorName(tableName, ix)
		var params []csharpParamTmpl
		for _, f := range gf {
			params = append(params, csharpParamTmpl{CsType: csharpTypeTable(f, schema), Prop: csharpSafeProp(f.Name)})
		}
		data.GetIndexMethods = append(data.GetIndexMethods, csharpGetIndexMethodTmpl{
			MethodName: imn,
			RowName:    rowName,
			Comparable: groupFieldsComparable(gf, schema),
			Params:     params,
			FuncName:   fn,
			Suffix:     suff,
			Arity:      len(gf),
		})
	}

	return data
}
