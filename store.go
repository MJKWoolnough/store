// Package store automatically configures a database to store structured information in an sql database
package store

import (
	"database/sql"
	"errors"
	"reflect"
	"strings"
	"sync"
)

const (
	add = iota
	get
	update
	remove
	getPage
	count
)

type field struct {
	name      string
	pos       int
	isPointer bool
}

type typeInfo struct {
	primary    int
	fields     []field
	statements []*sql.Stmt
}

type Store struct {
	db    *sql.DB
	types map[string]typeInfo
	mutex sync.Mutex
}

func New(driverName, dataSourceName string) (*Store, error) {
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	return &Store{
		db:    db,
		types: make(map[string]typeInfo),
	}, nil
}

func (s *Store) Close() error {
	err := s.db.Close()
	s.db = nil
	return err
}

func isPointerStruct(i interface{}) bool {
	t := reflect.TypeOf(i)
	return t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct
}

func (s *Store) Register(i interface{}) error {
	if s.db == nil {
		return DBClosed
	} else if !isPointerStruct(i) {
		return NoPointerStruct
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.defineType(i)
}

func (s *Store) defineType(i interface{}) error {
	name := typeName(i)
	if _, ok := s.types[name]; ok {
		return nil
	}

	s.types[name] = typeInfo{}

	v := reflect.ValueOf(i).Elem()
	numFields := v.Type().NumField()
	fields := make([]field, 0, numFields)
	id := 0
	idType := 0

	for n := 0; n < numFields; n++ {
		f := v.Type().Field(n)
		if f.PkgPath != "" { // not exported
			continue
		}
		fieldName := f.Name
		if fn := f.Tag.Get("store"); fn != "" {
			fieldName = fn
		}
		if fieldName == "-" { // Skip field
			continue
		}
		tmp := strings.ToLower(fieldName)
		for _, tf := range fields {
			if strings.ToLower(tf.name) == tmp {
				return DuplicateColumn
			}
		}
		isPointer := f.Type.Kind() == reflect.Ptr
		var iface interface{}
		if isPointer {
			iface = v.Field(n).Interface()
		} else {
			iface = v.Field(n).Addr().Interface()
		}
		if isPointerStruct(iface) {
			s.defineType(iface)
		} else if !isValidType(iface) {
			continue
		}
		if isValidKeyType(iface) {
			if idType < 3 && f.Tag.Get("key") == "1" {
				idType = 3
				id = len(fields)
			} else if idType < 2 && strings.ToLower(fieldName) == "id" {
				idType = 2
				id = len(fields)
			} else if idType < 1 {
				idType = 1
				id = len(fields)
			}
		}
		fields = append(fields, field{
			fieldName,
			n,
			isPointer,
		})
	}
	if idType == 0 {
		return NoKey
	}
	s.types[name] = typeInfo{
		primary: id,
	}

	// create statements
	var (
		sqlVars, sqlParams, setSQLParams, tableVars string
		doneFirst, doneFirstNonKey                  bool
	)

	for pos, f := range fields {
		if doneFirst {
			tableVars += ", "
		} else {
			doneFirst = true
		}
		if pos != id {
			if doneFirstNonKey {
				sqlVars += ", "
				setSQLParams += ", "
				sqlParams += ", "
			} else {
				doneFirstNonKey = true
			}
		}
		var varType string
		if isPointerStruct(getFieldPointer(i, f.pos)) {
			varType = "INTEGER"
		} else {
			varType = getType(i, f.pos)
		}
		tableVars += "[" + f.name + "] " + varType
		if pos == id {
			tableVars += " PRIMARY KEY AUTOINCREMENT"
		} else {
			sqlVars += "[" + f.name + "]"
			setSQLParams += "[" + f.name + "] = ?"
			sqlParams += "?"
		}
	}

	statements := make([]*sql.Stmt, 6)

	sql := "CREATE TABLE IF NOT EXISTS [" + name + "](" + tableVars + ");"
	_, err := s.db.Exec(sql)
	if err != nil {
		return err
	}

	sql = "INSERT INTO [" + name + "] (" + sqlVars + ") VALUES (" + sqlParams + ");"
	stmt, err := s.db.Prepare(sql)
	if err != nil {
		return err
	}
	statements[add] = stmt

	sql = "SELECT " + sqlVars + " FROM [" + name + "] WHERE [" + fields[id].name + "] = ? LIMIT 1;"
	stmt, err = s.db.Prepare(sql)
	if err != nil {
		return err
	}
	statements[get] = stmt

	sql = "UPDATE [" + name + "] SET " + setSQLParams + " WHERE [" + fields[id].name + "] = ?;"
	stmt, err = s.db.Prepare(sql)
	if err != nil {
		return err
	}
	statements[update] = stmt

	sql = "DELETE FROM [" + name + "] WHERE [" + fields[id].name + "] = ?;"
	stmt, err = s.db.Prepare(sql)
	if err != nil {
		return err
	}
	statements[remove] = stmt

	sql = "SELECT " + sqlVars + ", [" + fields[id].name + "] FROM [" + name + "] ORDER BY [" + fields[id].name + "] LIMIT ? OFFSET ?;"
	stmt, err = s.db.Prepare(sql)
	if err != nil {
		return err
	}
	statements[getPage] = stmt

	sql = "SELECT COUNT(1) FROM [" + name + "];"
	stmt, err = s.db.Prepare(sql)
	if err != nil {
		return err
	}
	statements[count] = stmt

	s.types[name] = typeInfo{
		primary:    id,
		fields:     fields,
		statements: statements,
	}
	return nil
}

func (s *Store) Count(i interface{}) (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if !isPointerStruct(i) {
		return 0, NoPointerStruct
	}
	name := typeName(i)
	stmt := s.types[name].statements[count]
	res, err := stmt.Query()
	if err != nil {
		return err
	}
	num := 0
	err = res.Scan(&num)
	return num, err
}

// Errors

var (
	DBClosed        = errors.New("database already closed")
	NoPointerStruct = errors.New("given variable is not a pointer to a struct")
	NoKey           = errors.New("could not determine key")
	DuplicateColumn = errors.New("duplicate column name found")
)
