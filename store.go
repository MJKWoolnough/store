// Package store automatically configures a database to store structured information in an sql database
package store

import (
	"database/sql"
	"errors"
	"reflect"
	"strings"
	"sync"
)

type field struct {
	name      string
	pos       []int
	isPointer bool
}

type statement struct {
	*sql.Stmt
	in  []int
	out []int
}

type typeInfo struct {
	primary    int
	fields     []field
	statements []statement
}

type Store struct {
	db    *sql.DB
	types map[string]typeInfo
	mutex sync.Mutex
}

func New(db *sql.DB) *Store {
	return &Store{
		db:    db,
		types: make(map[string]typeInfo),
	}
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

	v := reflect.ValueOf(i).Elem()
	numFields := v.Type().NumField()
	fields := make([]field, 0, numFields)
	id := 0
	idType := 0

	type tR struct {
		i interface{}
		n int
	}
	toRegister := make([]tR, 0)
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
		isPointer := f.Type.Kind() == reflect.Ptr
		var iface interface{}
		if isPointer {
			iface = v.Field(n).Interface()
		} else {
			iface = v.Field(n).Addr().Interface()
		}
		if isValidKeyType(iface) {
			if idType < 3 && f.Tag.Get("key") == "1" {
				idType = 3
				id = n
			} else if idType < 2 && strings.ToLower(fieldName) == "id" {
				idType = 2
				id = n
			} else if idType < 1 {
				idType = 1
				id = n
			}
		}
		pos := make([]int, 1, 2)
		pos[0] = n
		if isPointerStruct(iface) {
			toRegister = append(toRegister, tR{
				iface,
				len(fields),
			})
		} else if !isValidType(iface) {
			continue
		}
		fields = append(fields, field{
			fieldName,
			pos,
			isPointer,
		})
	}
	s.types[name] = typeInfo{
		primary: id,
	}
	for _, t := range toRegister {
		if err := s.defineType(t.i); err != nil {
			return err
		}

		fields[t.n].pos = append(fields[t.n].pos, s.types[typeName(t.i)].primary)
	}
	s.types[name] = typeInfo{
		primary: id,
		fields:  fields,
	}

	// create statements

	return nil
}

func isValidType(i interface{}) bool {
	switch i.(type) {
	case *int, *int8, *int16, *int32, *int64,
		*uint, *uint8, *uint16, *uint32, *uint64,
		*string, *float32, *float64, *bool:
		return true
	}
	return false
}

func isValidKeyType(i interface{}) bool {
	switch i.(type) {
	case *int, *int8, *int16, *int32, *int64,
		*uint, *uint8, *uint16, *uint32, *uint64:
		return true
	}
	return false
}

func typeName(i interface{}) string {
	name := reflect.TypeOf(i).String()
	if name[0] == '*' {
		name = name[1:]
	}
	return name
}

// Errors

var (
	DBClosed        = errors.New("database already closed")
	NoPointerStruct = errors.New("given variable is not a pointer to a struct")
)
