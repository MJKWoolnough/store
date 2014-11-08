package store

type TypeMap map[string]interface{}

type Interface interface {
	Get() TypeMap
	Key() string
}

type InterfaceTable interface {
	Interface
	TableName() string
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
