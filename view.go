package main

import (
	"fmt"
	"strings"
	"text/template"
)

// ListTablesData ListTables テンプレートに渡すデータ構造
type ListTablesData struct {
	DBName string
	Tables []TableSummary
}

// listTablesTemplate ListTables の出力フォーマット
const listTablesTemplate = `データベース「{{.DBName}}」のテーブル一覧 (全{{len .Tables}}件)
フォーマット: テーブル名 - テーブルコメント [PK: 主キー] [UK: 一意キー1; 一意キー2...] [FK: 外部キー -> 参照先テーブル.カラム; ...]
※ 複合キー（複数カラムで構成されるキー）は括弧でグループ化: (col1, col2)
※ 複数の異なるキー制約はセミコロンで区切り: key1; key2

{{range .Tables -}}
- {{.Name}} - {{.Comment}}{{if len .PK}} [PK: {{formatPK .PK}}]{{end}}{{if len .UK}} [UK: {{formatUK .UK}}]{{end}}{{if len .FK}} [FK: {{formatFK .FK}}]{{end}}
{{end -}}
`

var funcMap = template.FuncMap{
	"formatPK": formatPK,
	"formatUK": formatUK,
	"formatFK": formatFK,
}

// formatPK は主キー情報をフォーマットします
func formatPK(pk []string) string {
	if len(pk) == 0 {
		return ""
	}
	pkStr := strings.Join(pk, ", ")
	if len(pk) > 1 {
		pkStr = fmt.Sprintf("(%s)", pkStr)
	}
	return pkStr
}

// formatUK は一意キー情報をフォーマットします
func formatUK(uk []UniqueKey) string {
	if len(uk) == 0 {
		return ""
	}
	var ukInfo []string
	for _, k := range uk {
		if len(k.Columns) > 1 {
			ukInfo = append(ukInfo, fmt.Sprintf("(%s)", strings.Join(k.Columns, ", ")))
		} else {
			ukInfo = append(ukInfo, strings.Join(k.Columns, ", "))
		}
	}
	return strings.Join(ukInfo, "; ")
}

// formatFK は外部キー情報をフォーマットします
func formatFK(fk []ForeignKey) string {
	if len(fk) == 0 {
		return ""
	}
	var fkInfo []string
	for _, k := range fk {
		colStr := strings.Join(k.Columns, ", ")
		refColStr := strings.Join(k.RefColumns, ", ")

		if len(k.Columns) > 1 {
			colStr = fmt.Sprintf("(%s)", colStr)
		}

		if len(k.RefColumns) > 1 {
			refColStr = fmt.Sprintf("(%s)", refColStr)
		}

		fkInfo = append(fkInfo, fmt.Sprintf("%s -> %s.%s",
			colStr,
			k.RefTable,
			refColStr))
	}
	return strings.Join(fkInfo, "; ")
}
