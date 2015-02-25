package store

import (
	"database/sql"
	"reflect"
)

type SortBy struct {
	Column string
	Asc    bool
}

type Search struct {
	store  *Store
	i      interface{}
	Sort   []SortBy
	Filter Filter
}

func (s *Store) NewSearch(i interface{}) *Search {
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
		doneFirst    bool
		sql, sqlVars string
		vars         []interface{}
		name         = typeName(s.i)
	)
	if s.Filter != nil {
		sql += "WHERE " + s.Filter.SQL() + " "
		for _, i := range s.Filter.Vars() {
			if v := reflect.ValueOf(i); v.Kind() != reflect.Ptr {
				i = v.Addr().Interface()
			}
			if !isValidType(i) {
				return nil, ErrInvalidType
			}
			vars = append(vars, i)
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
		sql += "ORDER BY "
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
	get, err := s.store.db.Prepare("SELECT [" + t.fields[t.primary].name + "] FROM [" + name + "] " + sql + "LIMIT ? OFFSET ?;")
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
	rows, err := p.getStmt.Query(append(p.getVars(), len(is), offset)...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	return p.store.getPage(is, rows)
}

func (p *PreparedSearch) getVars() []interface{} {
	vars := make([]interface{}, len(p.vars), len(p.vars)+2)
	for n, v := range p.vars {
		vars[n] = reflect.ValueOf(v).Elem().Interface()
	}
	return vars
}
