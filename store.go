// Package store automatically configures a database to store structured information in an sql database
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
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
	}
	if !isPointerStruct(i) {
		return NoPointerStruct
	}

	name := typeName(i)
	fmt.Println("Processing", name)
	if _, ok := s.types[name]; ok {
		return nil
	}

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
			err := s.Register(iface)
			if err != nil {
				return nil
			}
			pos = append(pos, s.types[typeName(iface)].primary)
		} else if !isValidType(iface) {
			continue
		}
		fields = append(fields, field{
			fieldName,
			pos,
			isPointer,
		})
		fmt.Println(fields[len(fields)-1])
	}
	s.types[name] = typeInfo{
		id,
		fields,
		nil,
	}
	fmt.Println(s.types[name])
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
