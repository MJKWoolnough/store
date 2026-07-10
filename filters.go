package store

import "strings"

type Filter interface {
	SQL() string
	Vars() []any
}

type And []Filter

func (a And) SQL() string {
	var sql strings.Builder
	sql.WriteString("(")

	for n, f := range a {
		if n > 0 {
			sql.WriteString(" AND ")
		}

		sql.WriteString(f.SQL())
	}

	sql.WriteString(")")

	return sql.String()
}

func (a And) Vars() []any {
	var vars []any

	for _, f := range a {
		vars = append(vars, f.Vars()...)
	}

	return vars
}

type Or []Filter

func (o Or) SQL() string {
	var sql strings.Builder
	sql.WriteString("(")

	for n, f := range o {
		if n > 0 {
			sql.WriteString(" OR ")
		}

		sql.WriteString(f.SQL())
	}

	sql.WriteString(")")

	return sql.String()
}

func (o Or) Vars() []any {
	var vars []any

	for _, f := range o {
		vars = append(vars, f.Vars()...)
	}

	return vars
}
