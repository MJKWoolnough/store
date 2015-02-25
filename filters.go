package store

type Filter interface {
	SQL() string
	Vars() []interface{}
}

type And []Filter

func (a And) SQL() string {
	sql := "("
	for n, f := range a {
		if n > 0 {
			sql += " AND "
		}
		sql += f.SQL()
	}
	sql += ")"
	return sql
}

func (a And) Vars() []interface{} {
	var vars []interface{}
	for _, f := range a {
		vars = append(vars, f.Vars()...)
	}
	return vars
}

type Or []Filter

func (o Or) SQL() string {
	sql := "("
	for n, f := range o {
		if n > 0 {
			sql += " OR "
		}
		sql += f.SQL()
	}
	sql += ")"
	return sql
}

func (o Or) Vars() []interface{} {
	var vars []interface{}
	for _, f := range o {
		vars = append(vars, f.Vars()...)
	}
	return vars
}
