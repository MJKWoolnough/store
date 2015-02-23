package store

import (
	"database/sql"
	"reflect"
)

type SortBy struct {
	Column string
	Asc    bool
}

type Filter interface {
	SQL() string
	Vars() []interface{}
}

type Search struct {
	store   *Store
	i       interface{}
	Sort    []SortBy
	Filters []Filter
}

func (s *Store) Search(i interface{}) *Search {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if _, ok := s.types[typeName(i)]; !ok {
		return nil
	}
	return &Search{
		store: s,
		i:     i,
	}
}

type PreparedSearch struct {
	countStmt *sql.Stmt
	getStmt   *sql.Stmt
	vars      []interface{}
	store     *Store
}

func (s *Search) Prepare() (*PreparedSearch, error) {
	var (
		doneFirst bool
		sqlVars   string
	)
	name := typeName(s.i)
	var sql string
	var vars []interface{}
	if len(s.Filters) > 0 {
		sql += "WHERE "
		for n, f := range s.Filters {
			if n > 0 {
				sql += "AND "
			}
			sql += f.SQL() + " "
			for _, i := range f.Vars() {
				if v := reflect.ValueOf(i); v.Kind() != reflect.Ptr {
					i = v.Addr().Interface()
				}
				if !isValidType(i) {
					return nil, ErrInvalidType
				}
				vars = append(vars, i)
			}
		}
	}
	s.store.mutex.Lock()
	defer s.store.mutex.Unlock()
	count, err := s.store.db.Prepare("SELECT COUNT(1) FROM [" + name + "] " + sql)
	if err != nil {
		return nil, err
	}
	t := s.store.types[name]
	for pos, f := range t.fields {
		if pos == t.primary {
			continue
		}
		if doneFirst {
			sqlVars += ", "
		} else {
			doneFirst = true
		}
		sqlVars += "[" + f.name + "]"
	}
	if len(s.Sort) > 0 {
		sql += "SORT BY "
		for n, f := range s.Sort {
			if n > 0 {
				sql += ", "
			}
			sql += "[" + f.Column + "]"
			if f.Asc {
				sql += " ASC "
			} else {
				sql += " DESC "
			}
		}
	}
	get, err := s.store.db.Prepare("SELECT [" + t.fields[t.primary].name + "] FROM [" + name + "] " + sql)
	if err != nil {
		return nil, err
	}
	return &PreparedSearch{
		count,
		get,
		vars,
		s.store,
	}, nil
}

func (p *PreparedSearch) Count() (int, error) {
	p.store.mutex.Lock()
	defer p.store.mutex.Unlock()
	row := p.countStmt.QueryRow(p.getVars()...)
	var count int
	err := row.Scan(&count)
	return count, err
}

func (p *PreparedSearch) GetPage(is []interface{}, offset int) (int, error) {
	if len(is) == 0 {
		return 0, nil
	}
	p.store.mutex.Lock()
	defer p.store.mutex.Unlock()
	rows, err := p.getStmt.Query(p.getVars()...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	return p.store.getPage(is, rows)
}

func (p *PreparedSearch) getVars() []interface{} {
	vars := make([]interface{}, len(p.vars))
	for n, v := range p.vars {
		vars[n] = reflect.ValueOf(v).Elem().Interface()
	}
	return vars
}
