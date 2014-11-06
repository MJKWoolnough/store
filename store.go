package store

import (
	"fmt"
	"io"
	"reflect"
	"sync"
	"time"

	"code.google.com/p/go-sqlite/go1/sqlite3"
)

const (
	add = iota
	get
	update
	remove
	getPage
)

type TypeMap map[string]interface{}

type Interface interface {
	Get() TypeMap
	Key() string
}

type InterfaceTable interface {
	Interface
	TableName() string
}

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
	s.statements[tableName][update] = statement{Stmt: stmt, vars: append(vars, primary)}

	sql = "DELETE FROM [" + tableName + "] WHERE [" + primary + "] = ?;"
	stmt, err = s.db.Prepare(sql)
	if err != nil {
		return err
	}
	s.statements[tableName][remove] = statement{Stmt: stmt}

	sql = "SELECT " + sqlVars + " FROM [" + tableName + "] ORDER BY [" + primary + "] LIMIT ? OFFSET ?;"
	stmt, err = s.db.Prepare(sql)
	if err != nil {
		return err
	}
	s.statements[tableName][getPage] = statement{Stmt: stmt, vars: vars}

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
			err = fmt.Errorf("could not decide key")
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

func (s *Store) Get(ts ...Interface) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, t := range ts {
		tableName := TableName(t)
		stmt := s.statements[tableName][get]
		vars := t.Get()
		err := stmt.Query(unPointer(vars[t.Key()]))
		if err != nil && err != io.EOF {
			return err
		}
		err = stmt.Scan(stmt.Vars(vars)...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) GetPage(data []Interface, offset int) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if len(data) < 1 {
		return nil
	}
	tableName := TableName(data[0])
	stmt := s.statements[tableName][getPage]
	var (
		err error
		pos int
	)
	for err = stmt.Query(len(data), offset); err == nil; err = stmt.Next() {
		err = stmt.Scan(stmt.Vars(data[pos].Get()))
		if err != nil {
			break
		}
		pos++
	}
	if err != nil && err != io.EOF {
		return err
	}
	return nil
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

func TableName(t Interface) string {
	if it, ok := t.(InterfaceTable); ok {
		return it.TableName()
	}
	name := reflect.TypeOf(t).String()
	if name[0] == '*' {
		name = name[1:]
	}
	return name
}

func getType(i interface{}) string {
	switch i.(type) {
	case *int, *int64, *time.Time:
		return "INTEGER"
	case *float64:
		return "FLOAT"
	case *string:
		return "TEXT"
	case *[]byte:
		return "BLOB"
	}
	return ""
}

func unPointers(is []interface{}) []interface{} {
	for n := range is {
		is[n] = unPointer(is[n])
	}
	return is
}

func unPointer(i interface{}) interface{} {
	switch v := i.(type) {
	case *int:
		return *v
	case *int64:
		return *v
	case *float64:
		return *v
	case *string:
		return *v
	case *[]byte:
		return *v
	default:
		return nil
	}
}
