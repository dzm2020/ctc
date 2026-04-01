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
	MemberName string // @Type 成员名字段，用于 X_name / X_value 的字符串键
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
				Comment:    excelconv.SanitizeOneLineComment(m.NameCN),
				MemberName: m.Name,
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
	Pkg string
}

type structFieldTmpl struct {
	Priv, GoType, JSONName, Exported, Getter string
	NameCN       string   // @Type「中文描述」，单行；空则不生注释
	BinReadLines []string // 仅表行字段：tablebin 解码语句块
}

type configStructTmpl struct {
	StructName string
	AuxName    string
	Fields     []structFieldTmpl
}

func renderStructsFile(pkg string, snames []string, schema *excelconv.Schema, exportTags []string) (string, error) {
	t := codegenRoot()
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "struct_file_header", structFileHeaderTmpl{Pkg: pkg}); err != nil {
		return "", err
	}
	for _, sn := range snames {
		vis := visibleStructFields(schema.Structs[sn], exportTags)
		fields := make([]structFieldTmpl, 0, len(vis))
		for _, sf := range vis {
			priv := privateFieldIdent(sf.Name)
			got := goFieldTypeStruct(sf, schema)
			fields = append(fields, structFieldTmpl{
				Priv:     priv,
				GoType:   got,
				JSONName: sf.Name,
				Exported: exportedGoIdent(sf.Name),
				Getter:   getterMethodName(sf.Name),
				NameCN:   excelconv.SanitizeOneLineComment(sf.NameCN),
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
	NeedOS           bool // JSON 加载需要 os.ReadFile；仅 .bin 时为 false
	NeedFmt          bool
	NeedStrconv      bool
	NeedGroupStrings bool
	// NeedTableBin 为 true 时 import ctc/pkg/tablebin。
	NeedTableBin bool
}

type nestedGroupTmpl struct {
	TypeName  string
	TableName string
	GroupKey  string
	IsIndex   bool
	Fields    []structFieldTmpl
}

type ctorFieldLineTmpl struct {
	Priv, Param string
	NameCN      string
}

type groupValueCtorTmpl struct {
	CtorName   string
	GroupKey   string
	GroupType  string
	ParamList  string
	CtorFields []ctorFieldLineTmpl
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
	IDNameCN      string // 主键列在 @Type 中的中文描述（可见字段不含 id，单独填）
	AuxName       string
	Fields        []structFieldTmpl
	ViewAsGroups  []viewAsGroupData
	ViewAsIndexes []viewAsGroupData
}

type tableContainerNoGroupTmpl struct {
	TableName  string
	IDGoType   string
	Fields     []structFieldTmpl
	BinaryData bool
}

type tableGroupSlotTmpl struct {
	MapSuffix  string
	GroupName  string
	GroupType  string
	Comparable bool
	VarName    string
	RowKeyCall string
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
	MapSuffix  string
	IndexName  string
	IndexType  string
	Comparable bool
	VarName    string
	RowKeyCall string
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
	Fields            []structFieldTmpl
	BinaryData        bool
	GroupSlots        []tableGroupSlotTmpl
	IndexSlots        []tableIndexSlotTmpl
	GetRowsMethods    []getRowsMethodTmpl
	GetByIndexMethods []getByIndexMethodTmpl
}

func fieldToStructFieldTmpl(f excelconv.Field, schema *excelconv.Schema) structFieldTmpl {
	got := goFieldTypeTable(f, schema)
	priv := privateFieldIdent(f.Name)
	return structFieldTmpl{
		Priv:     priv,
		GoType:   got,
		JSONName: f.Name,
		Exported: exportedGoIdent(f.Name),
		Getter:   getterMethodName(f.Name),
		NameCN:   excelconv.SanitizeOneLineComment(f.NameCN),
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
	DataFile string // «表名».json 或 «表名».bin（由生成时 binaryExport 决定）
}

type loaderTmpl struct {
	Pkg           string
	SingleTable   bool
	FirstTable    string
	FirstDataFile string
	TableFields   []loaderTableField
	LoadLines     []loaderLoadLine
	DataExt       string // ".json" 或 ".bin"，用于注释
}

func renderLoaderFile(pkg string, tnames []string, binaryData bool) (string, error) {
	t := codegenRoot()
	var buf bytes.Buffer
	ext := ".json"
	dataExtComment := ".json（JSON 行数组）"
	if binaryData {
		ext = ".bin"
		dataExtComment = ".bin（紧凑表二进制）"
	}
	data := loaderTmpl{Pkg: pkg, DataExt: dataExtComment}
	if len(tnames) == 1 {
		data.SingleTable = true
		data.FirstTable = tnames[0]
		data.FirstDataFile = tnames[0] + ext
	} else {
		for _, tn := range tnames {
			data.TableFields = append(data.TableFields, loaderTableField{
				FieldName: gameDataFieldName(tn),
				TypeName:  tn,
			})
			data.LoadLines = append(data.LoadLines, loaderLoadLine{
				Receiver: gameDataFieldName(tn),
				DataFile: tn + ext,
			})
		}
	}
	if err := t.ExecuteTemplate(&buf, "loader", data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
