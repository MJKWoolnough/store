package store

import "io"

// Searcher
type Searcher interface {
	// Expr returns a sqlite snippet used in the WHERE clause.
	Expr(string) string
	//Params returns the required parameters for the sqlite snippet.
	Params() []interface{}
}

// Between is a searcher which searches for values between (inclusive) the
// given values.
type Between struct {
	from, to int
}

func (Between) Expr(col string) string {
	return "[" + col + "] BETWEEN ? AND ?"
}

func (b Between) Params() []interface{} {
	return []interface{}{b.from, b.to}
}

// Like implements a search which uses the LIKE syntax in a WHERE clause.
type Like string

func (Like) Expr(col string) string {
	return "[" + col + "] LIKE ?"
}

func (l Like) Params() []interface{} {
	return []interface{}{string(l)}
}

// Search is used for a custom (non primary key) search on a table.
//
// Returns the number of items found and an error if any occurred.
func (s *Store) Search(params map[string]Searcher, offset int, data ...Interface) (int, error) {
	if len(params) == 0 {
		return 0, NoParams{}
	}
	if len(data) == 0 {
		return 0, nil
	}
	table := tableName(data[0])
	cols := data[0].Get()
	vars := make([]string, len(cols))
	first := true
	columns := ""
	for col := range cols {
		if first {
			first = false
		} else {
			columns += ", "
		}
		columns += "[" + col + "]"
		vars = append(vars, col)
	}
	clause := ""
	first = true
	paramVars := make([]interface{}, 0, len(params)+2)
	for col, param := range params {
		if _, ok := cols[col]; !ok {
			return 0, UnknownColumn(col)
		}
		if first {
			first = false
		} else {
			clause += " AND "
		}
		clause += param.Expr(col)
		paramVars = append(paramVars, param.Params()...)
	}
	sql := "SELECT " + columns + " FROM [" + table + "] WHERE " + clause + " LIMIT ? OFFSET ?;"
	stmt, err := s.db.Query(sql, append(paramVars, len(data), offset)...)
	stmtVars := statement{Stmt: stmt, vars: vars}
	pos := 0
	for ; err == nil; err = stmt.Next() {
		if typeName := tableName(data[pos]); typeName != table {
			err = UnmatchedType{table, typeName}
		} else {
			err = stmtVars.Scan(stmtVars.Vars(data[pos].Get())...)
			pos++
		}
	}
	if err != nil && err != io.EOF {
		return pos, err
	}

	return pos, nil
}

//Errors

// NoParams is an error that occurs when no search parameters are given to
// Search.
type NoParams struct{}

func (NoParams) Error() string {
	return "no search parameters given"
}

// UnknownColumn is an error that occurrs when a search parameter requires a
// column which does not exist for the given type.
type UnknownColumn string

func (u UnknownColumn) Error() string {
	return "unknown column: " + string(u)
}
