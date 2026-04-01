package csharpgen

import (
	"bytes"
	"embed"
	"sync"
	"text/template"
)

//go:embed templates/*.tmpl
var csharpTmplFS embed.FS

var (
	csharpParseOnce sync.Once
	csharpRootTmpl  *template.Template
)

func csharpTemplateRoot() *template.Template {
	csharpParseOnce.Do(func() {
		root := template.New("csharpgen").Option("missingkey=error")
		var err error
		csharpRootTmpl, err = root.ParseFS(csharpTmplFS, "templates/table_bin_decoder.tmpl", "templates/enums.tmpl", "templates/structs.tmpl", "templates/table.tmpl", "templates/gamedata.tmpl", "templates/csproj.tmpl")
		if err != nil {
			panic(err)
		}
	})
	return csharpRootTmpl
}

func executeCSharpTemplate(name string, data any) (string, error) {
	var buf bytes.Buffer
	if err := csharpTemplateRoot().ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
