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
// tables.tmpl → 每张表一个 table_<slug>_gen.go，loader.tmpl → loader_gen.go（由 GenerateBundle 写出）。
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
