package store

import (
	"reflect"
	"time"

	"github.com/mxk/go-sqlite/sqlite3"
)

// TypeMap is a convenience type to reduce typing.
type TypeMap map[string]interface{}

// Interface is the data interface for the database.
type Interface interface {
	// Get returns a TypeMap of column name -> *type.
	Get() TypeMap
	// Key returns the column name of the primary key.
	Key() string
}

// InterfaceTable is an extension of Interface which allows a custom table name.
type InterfaceTable interface {
	Interface
	// TableName returns the custom table name for the type.
	TableName() string
}

type Sort struct {
	Interface
	SortBy string
	Asc    bool
}

func NewSort(data []Interface, sortBy string, asc bool) []Interface {
	if len(data) > 0 {
		data[0] = Sort{data[0], sortBy, asc}
	}
	return data
}

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

func tableName(t Interface) string {
	if s, ok := t.(Sort); ok {
		t = s.Interface
	}
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
	case *bool, *int, *int64, *time.Time:
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
	case *bool:
		return *v
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
