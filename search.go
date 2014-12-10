package store

import "io"

// Searcher is an interface for the different searcher params
type Searcher interface {
	// expr returns a sqlite snippet used in the WHERE clause.
	Expr() string
	// params returns the required parameters for the sqlite snippet.
	Params() []interface{}
	// column returns the column name in the table
	Column() string
}

// between is a searcher which searches for values between (inclusive) the
// given values.
type between struct {
	col      string
	from, to int
}

// Between returns a Searcher that looks for an integer between two values
func Between(column string, from, to int) Searcher {
	return between{column, from, to}
}

func (b between) Expr() string {
	return "[" + b.col + "] BETWEEN ? AND ?"
}

func (b between) Params() []interface{} {
	return []interface{}{b.from, b.to}
}

func (b between) Column() string {
	return b.col
}

// like implements a search which uses the LIKE syntax in a WHERE clause.
type like struct {
	col, likeStr string
}

// Like returns a Searcher that looks for similar strings
func Like(column, likeStr string) Searcher {
	return like{column, likeStr}
}

func (l like) Expr() string {
	return "[" + l.col + "] LIKE ?"
}

func (l like) Params() []interface{} {
	return []interface{}{l.likeStr}
}

func (l like) Column() string {
	return l.col
}

type matchString struct {
	col, match string
}

// MatchString searches for an exact string match on the given column
func MatchString(column, match string) Searcher {
	return matchString{column, match}
}

func (m matchString) Expr() string {
	return "[" + m.col + "] = ?"
}

func (m matchString) Params() []interface{} {
	return []interface{}{m.match}
}

func (m matchString) Column() string {
	return m.col
}

//

// Search is used for a custom (non primary key) search on a table.
//
// Returns the number of items found and an error if any occurred.
func (s *Store) Search(data []Interface, offset int, params ...Searcher) (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if len(params) == 0 {
		return 0, NoParams{}
	}
	if len(data) == 0 {
		return 0, nil
	}
	table := tableName(data[0])
	cols := data[0].Get()
	vars := make([]string, 0, len(cols))
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
	for _, param := range params {
		col := param.Column()
		if _, ok := cols[col]; !ok {
			return 0, UnknownColumn(col)
		}
		if first {
			first = false
		} else {
			clause += " AND "
		}
		clause += param.Expr()
		paramVars = append(paramVars, param.Params()...)
	}
	sql := "SELECT " + columns + " FROM [" + table + "] WHERE " + clause + " ORDER BY [" + data[0].Key() + "] LIMIT ? OFFSET ?;"
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

// SearchCount performs the search and returns the number of results
func (s *Store) SearchCount(data Interface, params ...Searcher) (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if len(params) == 0 {
		return 0, NoParams{}
	}
	table := tableName(data)
	cols := data.Get()
	first := true
	clause := ""
	paramVars := make([]interface{}, 0, len(params))
	for _, param := range params {
		col := param.Column()
		if _, ok := cols[col]; !ok {
			return 0, UnknownColumn(col)
		}
		if first {
			first = false
		} else {
			clause += " AND "
		}
		clause += param.Expr()
		paramVars = append(paramVars, param.Params()...)
	}
	sql := "SELECT count(1) FROM [" + table + "] WHERE " + clause + ";"
	stmt, err := s.db.Query(sql, paramVars...)
	if err != nil {
		return 0, err
	}
	count := 0
	err = stmt.Scan(&count)
	if err != nil && err != io.EOF {
		return 0, err
	}
	return count, nil
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
