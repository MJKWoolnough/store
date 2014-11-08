package store

import (
	"io"
	"reflect"
	"sync"
	"time"

	"github.com/mxk/go-sqlite/sqlite3"
)

const (
	add = iota
	get
	update
	remove
	getPage
)

type Store struct {
	db         *sqlite3.Conn
	mutex      sync.Mutex
	statements map[string][]statement
}

func NewStore(filename string) (*Store, error) {
	s, err := sqlite3.Open(filename)
	if err != nil {
		return nil, err
	}
	return &Store{
		db:         s,
		statements: make(map[string][]statement),
	}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Register(t Interface) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tableName := TableName(t)
	tVars := t.Get()

	var sqlVars, sqlParams, setSQLParams, tableVars string
	vars := make([]string, 0, len(tVars))

	first := true
	oFirst := true
	primary := t.Key()

	for typeName, typeVal := range tVars {
		if first {
			first = false
		} else {
			tableVars += ", "
		}
		if primary != typeName {
			if oFirst {
				oFirst = false
			} else {
				sqlVars += ", "
				setSQLParams += ", "
				sqlParams += ", "
			}
		}
		varType := getType(typeVal)
		tableVars += "[" + typeName + "] " + varType
		if primary == typeName {
			tableVars += " PRIMARY KEY AUTOINCREMENT"
		} else {
			sqlVars += "[" + typeName + "]"
			setSQLParams += "[" + typeName + "] = ?"
			sqlParams += "?"
			vars = append(vars, typeName)
		}
	}
	pVars := append(vars, primary)
	sql := "CREATE TABLE IF NOT EXISTS [" + tableName + "](" + tableVars + ");"
	err := s.db.Exec(sql)
	if err != nil {
		return err
	}
	s.statements[tableName] = make([]statement, 5)

	sql = "INSERT INTO [" + tableName + "] (" + sqlVars + ") VALUES (" + sqlParams + ");"
	stmt, err := s.db.Prepare(sql)
	if err != nil {
		return err
	}
	s.statements[tableName][add] = statement{Stmt: stmt, vars: vars}

	sql = "SELECT " + sqlVars + " FROM [" + tableName + "] WHERE [" + primary + "] = ? LIMIT 1;"
	stmt, err = s.db.Prepare(sql)
	if err != nil {
		return err
	}
	s.statements[tableName][get] = statement{Stmt: stmt, vars: vars}

	sql = "UPDATE [" + tableName + "] SET " + setSQLParams + " WHERE [" + primary + "] = ?;"
	stmt, err = s.db.Prepare(sql)
	if err != nil {
		return err
	}
	s.statements[tableName][update] = statement{Stmt: stmt, vars: pVars}

	sql = "DELETE FROM [" + tableName + "] WHERE [" + primary + "] = ?;"
	stmt, err = s.db.Prepare(sql)
	if err != nil {
		return err
	}
	s.statements[tableName][remove] = statement{Stmt: stmt}

	sql = "SELECT " + sqlVars + ", [" + primary + "] FROM [" + tableName + "] ORDER BY [" + primary + "] LIMIT ? OFFSET ?;"
	stmt, err = s.db.Prepare(sql)
	if err != nil {
		return err
	}
	s.statements[tableName][getPage] = statement{Stmt: stmt, vars: pVars}

	return nil
}

func (s *Store) Set(ts ...Interface) (id int, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, t := range ts {
		tableName := TableName(t)
		vars := t.Get()
		p := vars[t.Key()]
		var primary int
		if v, ok := p.(*int); ok {
			primary = *v
		} else {
			err = WrongKeyType{}
			return
		}
		if primary == 0 {
			stmt := s.statements[tableName][add]
			err = stmt.Exec(unPointers(stmt.Vars(vars))...)
			if err != nil {
				return
			}
			id = int(s.db.LastInsertId())
		} else {
			stmt := s.statements[tableName][update]
			err = stmt.Exec(unPointers(stmt.Vars(vars))...)
			if err != nil {
				return
			}
			id = primary
		}
	}
	return
}

func (s *Store) Get(data ...Interface) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, t := range data {
		tableName := TableName(t)
		stmt := s.statements[tableName][get]
		vars := t.Get()
		err := stmt.Query(unPointer(vars[t.Key()]))
		if err != nil && err != io.EOF {
			return err
		}
		err = stmt.Scan(stmt.Vars(vars)...)
		if err == io.EOF {
			if v, ok := vars[t.Key()].(*int); ok {
				*v = 0
			}
		} else if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) GetPage(offset int, data ...Interface) (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if len(data) < 1 {
		return 0, nil
	}
	tableName := TableName(data[0])
	stmt := s.statements[tableName][getPage]
	var (
		err error
		pos int
	)
	for err = stmt.Query(len(data), offset); err == nil; err = stmt.Next() {
		if typeName := TableName(data[pos]); typeName != tableName {
			err = UnmatchedType{tableName, typeName}
		} else {
			err = stmt.Scan(stmt.Vars(data[pos].Get())...)
			pos++
		}
	}
	if err != nil && err != io.EOF {
		return pos, err
	}
	return pos, nil
}

func (s *Store) Delete(ts ...Interface) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, t := range ts {
		tableName := TableName(t)
		err := s.statements[tableName][remove].Exec(unPointer(t.Get()[t.Key()]))
		if err != nil {
			return err
		}
	}
	return nil
}

//Errors

type WrongKeyType struct{}

func (WrongKeyType) Error() string {
	return "primary key needs to be int"
}

type UnmatchedType struct {
	MainType, ThisType string
}

func (u UnmatchedType) Error() string {
	return "expecting type " + u.MainType + ", got type " + u.ThisType
}
