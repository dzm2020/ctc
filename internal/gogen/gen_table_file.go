package gogen

import (
	"bytes"
	"strings"
	"text/template"

	"ctc/internal/excelconv"
)

// renderOneTableGoFile 生成单张表对应的 Go 源文件内容（package + import + 该表全部类型与方法）。
func renderOneTableGoFile(pkg, tname string, schema *excelconv.Schema, exportTags []string, binaryData bool) (string, error) {
	tpl := codegenRoot()
	var buf bytes.Buffer
	hdr := computeTableFileHeaderForTable(pkg, tname, schema, exportTags, binaryData)
	if err := tpl.ExecuteTemplate(&buf, "table_file_header", hdr); err != nil {
		return "", err
	}
	if err := appendTableDefinitions(&buf, tpl, tname, schema, exportTags, binaryData); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func computeTableFileHeaderForTable(pkg, tname string, schema *excelconv.Schema, exportTags []string, binaryData bool) tableFileHeaderTmpl {
	vis := visibleTableFields(schema.Tables[tname], exportTags)
	hasLookup := len(excelconv.DistinctFieldGroups(vis)) > 0 || len(excelconv.DistinctFieldIndexes(vis)) > 0
	needStrconv, needFmt := false, false
	if hasLookup {
		needStrconv, needFmt = tableFileGroupColKeyImportsForTable(schema, tname, exportTags)
	}
	needStrings := false
	queryStrconv := false
	for _, g := range excelconv.DistinctFieldGroups(vis) {
		gf := excelconv.FieldsInGroup(vis, g)
		if !groupFieldsComparable(gf, schema) {
			needStrings = true
			qc := buildGroupQuerySwitch(
				g, gf, schema,
				privateFieldIdent(g), tableGroupTypeIdent(tname, g), false,
			)
			if qc.QueryNeedsStrconv() {
				queryStrconv = true
			}
		}
	}
	for _, ix := range excelconv.DistinctFieldIndexes(vis) {
		gf := excelconv.FieldsInIndex(vis, ix)
		if !groupFieldsComparable(gf, schema) {
			needStrings = true
			qc := buildGroupQuerySwitch(
				ix, gf, schema,
				privateFieldIdent(ix), tableIndexTypeIdent(tname, ix), false,
			)
			if qc.QueryNeedsStrconv() {
				queryStrconv = true
			}
		}
	}
	if queryStrconv {
		needStrconv = true
	}
	return tableFileHeaderTmpl{
		Pkg:              pkg,
		NeedOS:           !binaryData,
		NeedFmt:          needFmt,
		NeedStrconv:      needStrconv,
		NeedGroupStrings: hasLookup && needStrings,
		NeedTableBin:     binaryData,
	}
}

func appendTableDefinitions(buf *bytes.Buffer, tpl *template.Template, tname string, schema *excelconv.Schema, exportTags []string, binaryData bool) error {
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
		if err := tpl.ExecuteTemplate(buf, "nested_group", nestedGroupTmpl{
			TypeName:  typ,
			TableName: tname,
			GroupKey:  p.Group,
			IsIndex:   false,
			Fields:    fields,
		}); err != nil {
			return err
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
		if err := tpl.ExecuteTemplate(buf, "nested_group", nestedGroupTmpl{
			TypeName:  typ,
			TableName: tname,
			GroupKey:  ix,
			IsIndex:   true,
			Fields:    fields,
		}); err != nil {
			return err
		}
	}

	idGo := tableRowPrimaryKeyGoType(schema, tname)
	fields := make([]structFieldTmpl, 0, len(visible))
	for _, fld := range visible {
		sf := fieldToStructFieldTmpl(fld, schema)
		if binaryData {
			sf.BinReadLines = binLoadAssignLines(fld, schema, exportTags, "row", sf.Priv, sf.GoType)
		}
		fields = append(fields, sf)
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

	idNameCN := ""
	for _, fld := range schema.Tables[tname] {
		if strings.EqualFold(fld.Name, excelconv.RowJSONIDKey) {
			idNameCN = excelconv.SanitizeOneLineComment(fld.NameCN)
			break
		}
	}

	if err := tpl.ExecuteTemplate(buf, "table_row", tableRowTmpl{
		TableName:     tname,
		IDGoType:      idGo,
		IDJSONKey:     excelconv.RowJSONIDKey,
		IDNameCN:      idNameCN,
		AuxName:       rowJSONAuxTypeName(tname),
		Fields:        fields,
		ViewAsGroups:  viewAs,
		ViewAsIndexes: viewIdx,
	}); err != nil {
		return err
	}

	groups := excelconv.DistinctFieldGroups(visible)
	for _, g := range groups {
		gf := excelconv.FieldsInGroup(visible, g)
		gtyp := tableGroupTypeIdent(tname, g)
		comp := groupFieldsComparable(gf, schema)
		if comp {
			pl, _ := buildGroupParamAndArgLists(gf, schema)
			cf := ctorFieldsFromGroupFields(gf)
			if err := tpl.ExecuteTemplate(buf, "group_value_ctor", groupValueCtorTmpl{
				CtorName:   groupValueCtorName(tname, g),
				GroupKey:   g,
				GroupType:  gtyp,
				ParamList:  pl,
				CtorFields: cf,
				ForIndex:   false,
			}); err != nil {
				return err
			}
		} else {
			fn := rowGroupKeyStrFuncName(tname, g)
			parts := make([]string, 0, len(gf))
			for _, fld := range gf {
				part, _, _ := goGroupKeyPartExpr(fld, schema, "r")
				parts = append(parts, part)
			}
			if err := tpl.ExecuteTemplate(buf, "row_group_key_str", rowGroupKeyStrTmpl{
				FuncName:  fn,
				TableName: tname,
				GroupKey:  g,
				KeyParts:  parts,
				ForIndex:  false,
			}); err != nil {
				return err
			}
		}
	}
	for _, ix := range indexes {
		gf := excelconv.FieldsInIndex(visible, ix)
		ityp := tableIndexTypeIdent(tname, ix)
		comp := groupFieldsComparable(gf, schema)
		if comp {
			pl, _ := buildGroupParamAndArgLists(gf, schema)
			cf := ctorFieldsFromGroupFields(gf)
			if err := tpl.ExecuteTemplate(buf, "group_value_ctor", groupValueCtorTmpl{
				CtorName:   groupValueCtorName(tname, ix),
				GroupKey:   ix,
				GroupType:  ityp,
				ParamList:  pl,
				CtorFields: cf,
				ForIndex:   true,
			}); err != nil {
				return err
			}
		} else {
			fn := rowIndexKeyStrFuncName(tname, ix)
			parts := make([]string, 0, len(gf))
			for _, fld := range gf {
				part, _, _ := goGroupKeyPartExpr(fld, schema, "r")
				parts = append(parts, part)
			}
			if err := tpl.ExecuteTemplate(buf, "row_group_key_str", rowGroupKeyStrTmpl{
				FuncName:  fn,
				TableName: tname,
				GroupKey:  ix,
				KeyParts:  parts,
				ForIndex:  true,
			}); err != nil {
				return err
			}
		}
	}

	if len(groups) == 0 && len(indexes) == 0 {
		if err := tpl.ExecuteTemplate(buf, "table_container_no_group", tableContainerNoGroupTmpl{
			TableName:  tname,
			IDGoType:   idGo,
			Fields:     fields,
			BinaryData: binaryData,
		}); err != nil {
			return err
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
				MapSuffix:  suff,
				GroupName:  g,
				GroupType:  gtyp,
				Comparable: comp,
				VarName:    indexVarName(i),
				RowKeyCall: rkc,
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
		if err := tpl.ExecuteTemplate(buf, "table_container_group", tableContainerGroupTmpl{
			TableName:         tname,
			IDGoType:          idGo,
			Fields:            fields,
			BinaryData:        binaryData,
			GroupSlots:        slots,
			IndexSlots:        idxSlots,
			GetRowsMethods:    getRows,
			GetByIndexMethods: getByIdx,
		}); err != nil {
			return err
		}
	}
	return nil
}

func ctorFieldsFromGroupFields(gf []excelconv.Field) []ctorFieldLineTmpl {
	cf := make([]ctorFieldLineTmpl, 0, len(gf))
	for _, fld := range gf {
		p := privateFieldIdent(fld.Name)
		cf = append(cf, ctorFieldLineTmpl{
			Priv:   p,
			Param:  p,
			NameCN: excelconv.SanitizeOneLineComment(fld.NameCN),
		})
	}
	return cf
}
