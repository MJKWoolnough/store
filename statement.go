package store

import "code.google.com/p/go-sqlite/go1/sqlite3"

type statement struct {
	*sqlite3.Stmt
	vars []string
}

func (s statement) Vars(t map[string]interface{}) []interface{} {
	r := make([]interface{}, len(s.vars))
	for i, v := range s.vars {
		r[i] = t[v]
	}
	return r
}
