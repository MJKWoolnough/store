package store

import (
	"reflect"
	"time"
)

func getFieldPointer(i interface{}, fieldNum int) interface{} {
	v := reflect.ValueOf(i).Elem()
	if v.NumField() < fieldNum {
		return nil
	}
	if v.Type().Field(fieldNum).PkgPath != "" {
		return nil
	}
	f := v.Field(fieldNum)
	if f.Kind() == reflect.Ptr {
		return f.Interface()
	}
	return f.Addr().Interface()
}

func getField(i interface{}, fieldNum int) interface{} {
	v := reflect.ValueOf(i)
	if v.NumField() < fieldNum {
		return nil
	}
	if v.Type().Field(fieldNum).PkgPath != "" {
		return nil
	}
	f := v.Field(fieldNum)
	if f.Kind() == reflect.Ptr {
		return f.Elem().Interface()
	}
	return f.Interface()
}

func getType(i interface{}, fieldNum int) string {
	v := getFieldPointer(i, fieldNum)
	if v == nil {
		return ""
	}
	switch v.(type) {
	case *int, *int64, *time.Time:
		return "INTEGER"
	case *float32, *float64:
		return "FLOAT"
	case *string:
		return "TEXT"
	case *[]byte:
		return "BLOB"
	}
	return ""
}

func isValidType(i interface{}) bool {
	switch i.(type) {
	case *int, *int64,
		*string, *float32, *float64, *bool, *time.Time:
		return true
	}
	return false
}

func isValidKeyType(i interface{}) bool {
	switch i.(type) {
	case *int, *int64:
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

func (t *typeInfo) GetID(i interface{}) int64 {
	switch v := getField(i, t.fields[t.primary].pos).(type) {
	case int:
		return int64(v)
	case int64:
		return v
	}
	return 0
}

func (t *typeInfo) SetID(i interface{}, id int64) {
	switch v := getFieldPointer(i, t.fields[t.primary].pos).(type) {
	case *int:
		*v = int(id)
	case *int64:
		*v = id
	}
}
