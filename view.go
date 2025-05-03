package main

import (
	"fmt"
	"strings"
	"text/template"
)

// ListTablesData is the data structure passed to the ListTables template
type ListTablesData struct {
	DBName string
	Tables []TableSummary
}

// listTablesTemplate is the output format for ListTables
const listTablesTemplate = `Tables in database "{{.DBName}}" (Total: {{len .Tables}})
Format: Table Name - Table Comment [PK: Primary Key] [UK: Unique Key 1; Unique Key 2...] [FK: Foreign Key -> Referenced Table.Column; ...]
* Composite keys (keys composed of multiple columns) are grouped in parentheses: (col1, col2)
* Multiple different key constraints are separated by semicolons: key1; key2

{{range .Tables -}}
- {{.Name}} - {{.Comment}}{{if len .PK}} [PK: {{formatPK .PK}}]{{end}}{{if len .UK}} [UK: {{formatUK .UK}}]{{end}}{{if len .FK}} [FK: {{formatFK .FK}}]{{end}}
{{end -}}
`

// TableDetail holds detailed information for individual tables (uses types from db.go)
type TableDetail struct {
	Name        string
	Comment     string
	Columns     []ColumnInfo
	PrimaryKeys []string
	UniqueKeys  []UniqueKey
	ForeignKeys []ForeignKey
	Indexes     []IndexInfo
}

// describeTableDetailTemplate is the output format for describe_tables
const describeTableDetailTemplate = `# Table: {{.Name}}{{if .Comment}} - {{.Comment}}{{end}}

## Columns{{range .Columns}}
{{formatColumn .}}{{end}}

## Key Information{{if .PrimaryKeys}}
[PK: {{formatPK .PrimaryKeys}}]{{end}}{{if .UniqueKeys}}
[UK: {{formatUK .UniqueKeys}}]{{end}}{{if .ForeignKeys}}
[FK: {{formatFK .ForeignKeys}}]{{end}}{{if .Indexes}}
[INDEX: {{formatIndex .Indexes}}]{{end}}
`

var funcMap = template.FuncMap{
	"formatPK":     formatPK,
	"formatUK":     formatUK,
	"formatFK":     formatFK,
	"formatColumn": formatColumn,
	"formatIndex":  formatIndex,
}

// formatPK formats primary key information
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

// formatUK formats unique key information
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

// formatFK formats foreign key information
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

// formatColumn formats column information
func formatColumn(col ColumnInfo) string {
	nullable := "NOT NULL"
	if col.IsNullable == "YES" {
		nullable = "NULL"
	}

	defaultValue := ""
	if col.Default.Valid {
		defaultValue = fmt.Sprintf(" DEFAULT %s", col.Default.String)
	}

	comment := ""
	if col.Comment != "" {
		comment = fmt.Sprintf(" [%s]", col.Comment)
	}

	return fmt.Sprintf("- %s: %s %s%s%s",
		col.Name, col.Type, nullable, defaultValue, comment)
}

func formatIndex(idx []IndexInfo) string {
	if len(idx) == 0 {
		return ""
	}
	var idxInfo []string
	for _, i := range idx {
		if len(i.Columns) > 1 {
			idxInfo = append(idxInfo, fmt.Sprintf("(%s)", strings.Join(i.Columns, ", ")))
		} else {
			idxInfo = append(idxInfo, strings.Join(i.Columns, ", "))
		}
	}
	return strings.Join(idxInfo, "; ")
}
