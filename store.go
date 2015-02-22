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
	isStruct  bool
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
		isStruct := false
		if isPointerStruct(iface) {
			s.defineType(iface)
			isStruct = true
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
			isStruct,
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
		if f.isStruct {
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

func (s *Store) Set(is ...interface{}) error {
	var toSet []interface{}
	for _, i := range is {
		t, ok := s.types[typeName(i)]
		if !ok {
			return UnregisteredType
		}
		toSet = toSet[:0]
		err := s.Set(i, &t, &toSet)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) set(i interface{}, t *typeInfo, toSet *[]interface{}) error {
	for _, oi := range *toSet {
		if oi == i {
			return nil
		}
	}
	(*toSet) = append(*toSet, i)
	id := t.GetID(i)
	isUpdate := id != 0
	vars := make([]interface{}, 0, len(t.fields))
	for pos, f := range t.fields {
		if pos == t.primary {
			continue
		}
		if f.isStruct {
			ni := getFieldPointer(i, f.pos)
			nt := s.types[typeName(ni)]
			err := s.Set(ni, &nt, toSet)
			if err != nil {
				return err
			}
			vars = append(vars, getField(ni, nt.fields[nt.primary].pos))
		} else {
			vars = append(vars, getField(i, f.pos))
		}
	}
	if isUpdate {
		vars = append(vars, id)
		_, err := t.statements[update].Exec(vars...)
		return err
	}
	r, err := t.statements[add].Exec(vars...)
	if err != nil {
		return err
	}
	lid, err := r.LastInsertId()
	if err != nil {
		return err
	}
	t.SetID(i, lid)
	return nil
}

func (s *Store) Get(is ...interface{}) error {
	for _, i := range is {
		t, ok := s.types[typeName(i)]
		if !ok {
			return UnregisteredType
		}
		id := t.GetID(i)
		if id == 0 {
			continue
		}
		vars := make([]interface{}, 0, len(t.fields))
		var toGet []interface{}
		for pos, f := range t.fields {
			if pos == t.primary {
				continue
			}
			if f.isStruct {
				ni := getFieldPointer(i, f.pos)
				nt := s.types[typeName(ni)]
				toGet = append(toGet, ni)
				vars = append(vars, getFieldPointer(ni, nt.fields[nt.primary].pos))
			} else {
				vars = append(vars, getFieldPointer(i, f.pos))
			}
		}
		row := t.statements[get].QueryRow(id)
		err := row.Scan(vars...)
		if err != nil {
			return err
		}
		if len(toGet) > 0 {
			if err = s.Get(toGet...); err != nil {
				return err
			}
		}
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
	DBClosed         = errors.New("database already closed")
	NoPointerStruct  = errors.New("given variable is not a pointer to a struct")
	NoKey            = errors.New("could not determine key")
	DuplicateColumn  = errors.New("duplicate column name found")
	UnregisteredType = errors.New("type not registered")
)
