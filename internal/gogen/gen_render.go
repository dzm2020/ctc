package gogen

import (
	"bytes"
	"embed"
	"sort"
	"sync"
	"text/template"

	"ctc/internal/excelconv"
)

// 模板按生成文件拆分：enums.tmpl → enums_gen.go，structs.tmpl → structs_gen.go，
// tables.tmpl → tables_gen.go，loader.tmpl → loader_gen.go（由 GenerateBundle 写出）。
//
//go:embed templates/enums.tmpl templates/structs.tmpl templates/tables.tmpl templates/loader.tmpl
var tmplFS embed.FS

var (
	codegenParseOnce sync.Once
	codegenTmpl      *template.Template
)

func codegenRoot() *template.Template {
	codegenParseOnce.Do(func() {
		root := template.New("codegen").Option("missingkey=error")
		var err error
		codegenTmpl, err = root.ParseFS(tmplFS,
			"templates/enums.tmpl",
			"templates/structs.tmpl",
			"templates/tables.tmpl",
			"templates/loader.tmpl",
		)
		if err != nil {
			panic(err)
		}
	})
	return codegenTmpl
}

// --- enums ---

type pkgTmpl struct {
	Pkg string
}

type enumMemberTmpl struct {
	ParentEnum string
	ConstName  string
	Value      int
	Comment    string
}

type enumTmpl struct {
	Name    string
	Members []enumMemberTmpl
}

type enumsBodyTmpl struct {
	Enums []enumTmpl
}

func renderEnumsFile(pkg string, schema *excelconv.Schema) (string, error) {
	t := codegenRoot()
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "header_types_only", pkgTmpl{Pkg: pkg}); err != nil {
		return "", err
	}
	var enames []string
	for n := range schema.Enums {
		enames = append(enames, n)
	}
	sort.Strings(enames)
	var enums []enumTmpl
	for _, en := range enames {
		members := schema.Enums[en]
		var ms []enumMemberTmpl
		for _, m := range members {
			v := 0
			if schema.EnumValue[en] != nil {
				v = schema.EnumValue[en][m.Name]
			}
			ms = append(ms, enumMemberTmpl{
				ParentEnum: en,
				ConstName:  constName(en, m.Name),
				Value:      v,
				Comment:    m.NameCN,
			})
		}
		enums = append(enums, enumTmpl{Name: en, Members: ms})
	}
	if err := t.ExecuteTemplate(&buf, "enums_body", enumsBodyTmpl{Enums: enums}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// --- structs ---

type structFileHeaderTmpl struct {
	Pkg        string
	NeedSlices bool
}

type structFieldTmpl struct {
	Priv, GoType, JSONName, Exported, Getter string
	UseSliceGetter                         bool
}

type configStructTmpl struct {
	StructName string
	AuxName    string
	Fields     []structFieldTmpl
}

func renderStructsFile(pkg string, snames []string, schema *excelconv.Schema, exportTags []string) (string, error) {
	t := codegenRoot()
	var buf bytes.Buffer
	structNeedSlices := false
	for _, sn := range snames {
		if structFieldsNeedSlices(visibleStructFields(schema.Structs[sn], exportTags)) {
			structNeedSlices = true
			break
		}
	}
	if err := t.ExecuteTemplate(&buf, "struct_file_header", structFileHeaderTmpl{Pkg: pkg, NeedSlices: structNeedSlices}); err != nil {
		return "", err
	}
	for _, sn := range snames {
		vis := visibleStructFields(schema.Structs[sn], exportTags)
		fields := make([]structFieldTmpl, 0, len(vis))
		for _, sf := range vis {
			priv := privateFieldIdent(sf.Name)
			got := goFieldTypeStruct(sf, schema)
			fields = append(fields, structFieldTmpl{
				Priv:            priv,
				GoType:          got,
				JSONName:        sf.Name,
				Exported:        exportedGoIdent(sf.Name),
				Getter:          getterMethodName(sf.Name),
				UseSliceGetter:  len(got) >= 2 && got[:2] == "[]",
			})
		}
		if err := t.ExecuteTemplate(&buf, "config_struct", configStructTmpl{
			StructName: sn,
			AuxName:    structJSONAuxTypeName(sn),
			Fields:     fields,
		}); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

// --- tables ---

type tableFileHeaderTmpl struct {
	Pkg              string
	NeedSlices       bool
	NeedFmt          bool
	NeedStrconv      bool
	NeedGroupStrings bool
}

type nestedGroupTmpl struct {
	TypeName  string
	TableName string
	GroupKey  string
	IsIndex   bool
	Fields    []structFieldTmpl
}

type groupValueCtorTmpl struct {
	CtorName   string
	GroupKey   string
	GroupType  string
	ParamList  string
	CtorFields []struct{ Priv, Param string }
	ForIndex   bool
}

type rowGroupKeyStrTmpl struct {
	FuncName  string
	TableName string
	GroupKey  string
	KeyParts  []string
	ForIndex  bool
}

type viewAsGroupData struct {
	Method      string
	GroupType   string
	Assignments []struct{ Priv string }
}

type tableRowTmpl struct {
	TableName     string
	IDGoType      string
	IDJSONKey     string
	AuxName       string
	Fields        []structFieldTmpl
	ViewAsGroups  []viewAsGroupData
	ViewAsIndexes []viewAsGroupData
}

type tableContainerNoGroupTmpl struct {
	TableName string
	IDGoType  string
}

type tableGroupSlotTmpl struct {
	MapSuffix   string
	GroupName   string
	GroupType   string
	Comparable  bool
	VarName     string
	RowKeyCall  string
}

type getRowsMethodTmpl struct {
	MethodName   string
	MapSuffix    string
	CtorName     string
	ParamList    string
	QueryArgList string
	Comparable   bool
	N            int
	ParseLines   []string
	KeyJoinElts  []string
}

type tableIndexSlotTmpl struct {
	MapSuffix   string
	IndexName   string
	IndexType   string
	Comparable  bool
	VarName     string
	RowKeyCall  string
}

type getByIndexMethodTmpl struct {
	MethodName   string
	MapSuffix    string
	CtorName     string
	ParamList    string
	QueryArgList string
	Comparable   bool
	N            int
	ParseLines   []string
	KeyJoinElts  []string
}

type tableContainerGroupTmpl struct {
	TableName         string
	IDGoType          string
	GroupSlots        []tableGroupSlotTmpl
	IndexSlots        []tableIndexSlotTmpl
	GetRowsMethods    []getRowsMethodTmpl
	GetByIndexMethods []getByIndexMethodTmpl
}

func fieldToStructFieldTmpl(f excelconv.Field, schema *excelconv.Schema) structFieldTmpl {
	got := goFieldTypeTable(f, schema)
	priv := privateFieldIdent(f.Name)
	return structFieldTmpl{
		Priv:           priv,
		GoType:         got,
		JSONName:       f.Name,
		Exported:       exportedGoIdent(f.Name),
		Getter:         getterMethodName(f.Name),
		UseSliceGetter: len(got) >= 2 && got[:2] == "[]",
	}
}

func renderTablesFile(pkg string, tnames []string, schema *excelconv.Schema, exportTags []string) (string, error) {
	t := codegenRoot()
	var buf bytes.Buffer
	needSlices := false
	for _, tn := range tnames {
		vis := visibleTableFields(schema.Tables[tn], exportTags)
		if tableFieldsNeedSlices(vis) {
			needSlices = true
			break
		}
	}
	hasLookup := anyTableHasGroupsOrIndexes(schema, exportTags)
	needStrconv, needFmt := false, false
	if hasLookup {
		needStrconv, needFmt = tableFileGroupColKeyImports(schema, exportTags)
	}
	needStrings := false
	queryStrconv := false
	for _, tn := range tnames {
		vis := visibleTableFields(schema.Tables[tn], exportTags)
		for _, g := range excelconv.DistinctFieldGroups(vis) {
			gf := excelconv.FieldsInGroup(vis, g)
			comp := groupFieldsComparable(gf, schema)
			if !comp {
				needStrings = true
				qc := buildGroupQuerySwitch(
					g, gf, schema,
					privateFieldIdent(g), tableGroupTypeIdent(tn, g), false,
				)
				if qc.QueryNeedsStrconv() {
					queryStrconv = true
				}
			}
		}
		for _, ix := range excelconv.DistinctFieldIndexes(vis) {
			gf := excelconv.FieldsInIndex(vis, ix)
			comp := groupFieldsComparable(gf, schema)
			if !comp {
				needStrings = true
				qc := buildGroupQuerySwitch(
					ix, gf, schema,
					privateFieldIdent(ix), tableIndexTypeIdent(tn, ix), false,
				)
				if qc.QueryNeedsStrconv() {
					queryStrconv = true
				}
			}
		}
	}
	if queryStrconv {
		needStrconv = true
	}
	if err := t.ExecuteTemplate(&buf, "table_file_header", tableFileHeaderTmpl{
		Pkg:              pkg,
		NeedSlices:       needSlices,
		NeedFmt:          needFmt,
		NeedStrconv:      needStrconv,
		NeedGroupStrings: hasLookup && needStrings,
	}); err != nil {
		return "", err
	}

	for _, tname := range tnames {
		visible := visibleTableFields(schema.Tables[tname], exportTags)
		emit := excelconv.RowStructEmitOrder(visible)

		for _, p := range emit {
			if p.Group == "" {
				continue
			}
			gf := excelconv.FieldsInGroup(visible, p.Group)
			typ := tableGroupTypeIdent(tname, p.Group)
			fields := make([]structFieldTmpl, 0, len(gf))
			for _, fld := range gf {
				fields = append(fields, fieldToStructFieldTmpl(fld, schema))
			}
			if err := t.ExecuteTemplate(&buf, "nested_group", nestedGroupTmpl{
				TypeName:  typ,
				TableName: tname,
				GroupKey:  p.Group,
				IsIndex:   false,
				Fields:    fields,
			}); err != nil {
				return "", err
			}
		}
		indexes := excelconv.DistinctFieldIndexes(visible)
		for _, ix := range indexes {
			gf := excelconv.FieldsInIndex(visible, ix)
			typ := tableIndexTypeIdent(tname, ix)
			fields := make([]structFieldTmpl, 0, len(gf))
			for _, fld := range gf {
				fields = append(fields, fieldToStructFieldTmpl(fld, schema))
			}
			if err := t.ExecuteTemplate(&buf, "nested_group", nestedGroupTmpl{
				TypeName:  typ,
				TableName: tname,
				GroupKey:  ix,
				IsIndex:   true,
				Fields:    fields,
			}); err != nil {
				return "", err
			}
		}

		idGo := tableRowPrimaryKeyGoType(schema, tname)
		fields := make([]structFieldTmpl, 0, len(visible))
		for _, fld := range visible {
			fields = append(fields, fieldToStructFieldTmpl(fld, schema))
		}

		var viewAs []viewAsGroupData
		for _, p := range emit {
			if p.Group == "" {
				continue
			}
			gTyp := tableGroupTypeIdent(tname, p.Group)
			mth := viewAsGroupMethodName(p.Group)
			gf := excelconv.FieldsInGroup(visible, p.Group)
			var asgn []struct{ Priv string }
			for _, fld := range gf {
				asgn = append(asgn, struct{ Priv string }{Priv: privateFieldIdent(fld.Name)})
			}
			viewAs = append(viewAs, viewAsGroupData{Method: mth, GroupType: gTyp, Assignments: asgn})
		}
		ni := len(indexes)
		var viewIdx []viewAsGroupData
		for _, ix := range indexes {
			iTyp := tableIndexTypeIdent(tname, ix)
			mth := viewAsIndexMethodName(ix, ni)
			gf := excelconv.FieldsInIndex(visible, ix)
			var asgn []struct{ Priv string }
			for _, fld := range gf {
				asgn = append(asgn, struct{ Priv string }{Priv: privateFieldIdent(fld.Name)})
			}
			viewIdx = append(viewIdx, viewAsGroupData{Method: mth, GroupType: iTyp, Assignments: asgn})
		}

		if err := t.ExecuteTemplate(&buf, "table_row", tableRowTmpl{
			TableName:     tname,
			IDGoType:      idGo,
			IDJSONKey:     excelconv.RowJSONIDKey,
			AuxName:       rowJSONAuxTypeName(tname),
			Fields:        fields,
			ViewAsGroups:  viewAs,
			ViewAsIndexes: viewIdx,
		}); err != nil {
			return "", err
		}

		groups := excelconv.DistinctFieldGroups(visible)
		for _, g := range groups {
			gf := excelconv.FieldsInGroup(visible, g)
			gtyp := tableGroupTypeIdent(tname, g)
			comp := groupFieldsComparable(gf, schema)
			if comp {
				pl, _ := buildGroupParamAndArgLists(gf, schema)
				var cf []struct{ Priv, Param string }
				for _, fld := range gf {
					p := privateFieldIdent(fld.Name)
					cf = append(cf, struct{ Priv, Param string }{Priv: p, Param: p})
				}
				if err := t.ExecuteTemplate(&buf, "group_value_ctor", groupValueCtorTmpl{
					CtorName:   groupValueCtorName(tname, g),
					GroupKey:   g,
					GroupType:  gtyp,
					ParamList:  pl,
					CtorFields: cf,
					ForIndex:   false,
				}); err != nil {
					return "", err
				}
			} else {
				fn := rowGroupKeyStrFuncName(tname, g)
				parts := make([]string, 0, len(gf))
				for _, fld := range gf {
					part, _, _ := goGroupKeyPartExpr(fld, schema, "r")
					parts = append(parts, part)
				}
				if err := t.ExecuteTemplate(&buf, "row_group_key_str", rowGroupKeyStrTmpl{
					FuncName:  fn,
					TableName: tname,
					GroupKey:  g,
					KeyParts:  parts,
					ForIndex:  false,
				}); err != nil {
					return "", err
				}
			}
		}
		for _, ix := range indexes {
			gf := excelconv.FieldsInIndex(visible, ix)
			ityp := tableIndexTypeIdent(tname, ix)
			comp := groupFieldsComparable(gf, schema)
			if comp {
				pl, _ := buildGroupParamAndArgLists(gf, schema)
				var cf []struct{ Priv, Param string }
				for _, fld := range gf {
					p := privateFieldIdent(fld.Name)
					cf = append(cf, struct{ Priv, Param string }{Priv: p, Param: p})
				}
				if err := t.ExecuteTemplate(&buf, "group_value_ctor", groupValueCtorTmpl{
					CtorName:   groupValueCtorName(tname, ix),
					GroupKey:   ix,
					GroupType:  ityp,
					ParamList:  pl,
					CtorFields: cf,
					ForIndex:   true,
				}); err != nil {
					return "", err
				}
			} else {
				fn := rowIndexKeyStrFuncName(tname, ix)
				parts := make([]string, 0, len(gf))
				for _, fld := range gf {
					part, _, _ := goGroupKeyPartExpr(fld, schema, "r")
					parts = append(parts, part)
				}
				if err := t.ExecuteTemplate(&buf, "row_group_key_str", rowGroupKeyStrTmpl{
					FuncName:  fn,
					TableName: tname,
					GroupKey:  ix,
					KeyParts:  parts,
					ForIndex:  true,
				}); err != nil {
					return "", err
				}
			}
		}

		if len(groups) == 0 && len(indexes) == 0 {
			if err := t.ExecuteTemplate(&buf, "table_container_no_group", tableContainerNoGroupTmpl{
				TableName: tname,
				IDGoType:  idGo,
			}); err != nil {
				return "", err
			}
		} else {
			ng := len(groups)
			var slots []tableGroupSlotTmpl
			var getRows []getRowsMethodTmpl
			for i, g := range groups {
				gf := excelconv.FieldsInGroup(visible, g)
				suff := privateFieldIdent(g)
				gtyp := tableGroupTypeIdent(tname, g)
				comp := groupFieldsComparable(gf, schema)
				ctor := groupValueCtorName(tname, g)
				var rkc string
				if comp {
					rkc = buildRowKeyCall(ctor, gf)
				} else {
					rkc = rowGroupKeyStrFuncName(tname, g) + "(row)"
				}
				slots = append(slots, tableGroupSlotTmpl{
					MapSuffix:   suff,
					GroupName:   g,
					GroupType:   gtyp,
					Comparable:  comp,
					VarName:     indexVarName(i),
					RowKeyCall:  rkc,
				})
				mn := "GetRowsByGroup_" + suff
				if ng == 1 {
					mn = "GetRowsByGroupKey"
				}
				if comp {
					pl, al := buildGroupParamAndArgLists(gf, schema)
					getRows = append(getRows, getRowsMethodTmpl{
						MethodName:   mn,
						MapSuffix:    suff,
						CtorName:     ctor,
						ParamList:    pl,
						QueryArgList: al,
						Comparable:   true,
					})
				} else {
					qc := buildGroupQuerySwitch(g, gf, schema, suff, gtyp, false)
					getRows = append(getRows, getRowsMethodTmpl{
						MethodName:  mn,
						MapSuffix:   suff,
						CtorName:    "",
						Comparable:  false,
						N:           qc.N,
						ParseLines:  qc.ParseLines,
						KeyJoinElts: qc.KeyJoinElts,
					})
				}
			}
			ni := len(indexes)
			var idxSlots []tableIndexSlotTmpl
			var getByIdx []getByIndexMethodTmpl
			for j, ix := range indexes {
				gf := excelconv.FieldsInIndex(visible, ix)
				suff := privateFieldIdent(ix)
				ityp := tableIndexTypeIdent(tname, ix)
				comp := groupFieldsComparable(gf, schema)
				ctor := groupValueCtorName(tname, ix)
				var rkc string
				if comp {
					rkc = buildRowKeyCall(ctor, gf)
				} else {
					rkc = rowIndexKeyStrFuncName(tname, ix) + "(row)"
				}
				idxSlots = append(idxSlots, tableIndexSlotTmpl{
					MapSuffix:  suff,
					IndexName:  ix,
					IndexType:  ityp,
					Comparable: comp,
					VarName:    indexSlotVarName(j),
					RowKeyCall: rkc,
				})
				imn := "GetByIndex_" + suff
				if ni == 1 {
					imn = "GetByIndexKey"
				}
				if comp {
					pl, al := buildGroupParamAndArgLists(gf, schema)
					getByIdx = append(getByIdx, getByIndexMethodTmpl{
						MethodName:   imn,
						MapSuffix:    suff,
						CtorName:     ctor,
						ParamList:    pl,
						QueryArgList: al,
						Comparable:   true,
					})
				} else {
					qc := buildGroupQuerySwitch(ix, gf, schema, suff, ityp, false)
					getByIdx = append(getByIdx, getByIndexMethodTmpl{
						MethodName:  imn,
						MapSuffix:   suff,
						CtorName:    "",
						Comparable:  false,
						N:           qc.N,
						ParseLines:  qc.ParseLines,
						KeyJoinElts: qc.KeyJoinElts,
					})
				}
			}
			if err := t.ExecuteTemplate(&buf, "table_container_group", tableContainerGroupTmpl{
				TableName:         tname,
				IDGoType:          idGo,
				GroupSlots:        slots,
				IndexSlots:        idxSlots,
				GetRowsMethods:    getRows,
				GetByIndexMethods: getByIdx,
			}); err != nil {
				return "", err
			}
		}
	}

	return buf.String(), nil
}

func indexVarName(i int) string {
	const prefix = "kg"
	return prefix + itoaSmall(i)
}

func indexSlotVarName(i int) string {
	const prefix = "ik"
	return prefix + itoaSmall(i)
}

func itoaSmall(i int) string {
	if i == 0 {
		return "0"
	}
	var b [12]byte
	n := len(b)
	ii := i
	for ii > 0 {
		n--
		b[n] = byte('0' + ii%10)
		ii /= 10
	}
	return string(b[n:])
}

// --- loader ---

type loaderTableField struct {
	FieldName string
	TypeName  string
}

type loaderLoadLine struct {
	Receiver string
	JSONName string
}

type loaderTmpl struct {
	Pkg         string
	SingleTable bool
	FirstTable  string
	FirstJSON   string
	TableFields []loaderTableField
	LoadLines   []loaderLoadLine
}

func renderLoaderFile(pkg string, tnames []string) (string, error) {
	t := codegenRoot()
	var buf bytes.Buffer
	data := loaderTmpl{Pkg: pkg}
	if len(tnames) == 1 {
		data.SingleTable = true
		data.FirstTable = tnames[0]
		data.FirstJSON = tnames[0] + ".json"
	} else {
		for _, tn := range tnames {
			data.TableFields = append(data.TableFields, loaderTableField{
				FieldName: gameDataFieldName(tn),
				TypeName:  tn,
			})
			data.LoadLines = append(data.LoadLines, loaderLoadLine{
				Receiver: gameDataFieldName(tn),
				JSONName: tn + ".json",
			})
		}
	}
	if err := t.ExecuteTemplate(&buf, "loader", data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
