package store

import "io"

// Searcher is an interface for the different searcher params
type Searcher interface {
	// expr returns a sqlite snippet used in the WHERE clause.
	Expr() string
	// params returns the required parameters for the sqlite snippet.
	Params() []interface{}
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

type notLike struct {
	col, notLikeStr string
}

// NotLike returns a Searcher that excludes results with similar strings
func NotLike(column, notLikeStr string) Searcher {
	return notLike{column, notLikeStr}
}

func (n notLike) Expr() string {
	return "[" + n.col + "] NOT LIKE ?"
}

func (n notLike) Params() []interface{} {
	return []interface{}{n.notLikeStr}
}

type matchString struct {
	col, match string
}

// MatchString searches for an exact (case-insensitive) string match on the given column
func MatchString(column, match string) Searcher {
	return matchString{column, match}
}

func (m matchString) Expr() string {
	return "[" + m.col + "] = ?"
}

func (m matchString) Params() []interface{} {
	return []interface{}{m.match}
}

type matchInt struct {
	col   string
	match int
}

// MatchInt searches for an exact integer match on the given column
func MatchInt(column string, match int) Searcher {
	return matchInt{column, match}
}

func (m matchInt) Expr() string {
	return "[" + m.col + "] = ?"
}

func (m matchInt) Params() []interface{} {
	return []interface{}{m.match}
}

type or []Searcher

// Or returns a Searcher that matches for any of the given searchers
func Or(searchers ...Searcher) Searcher {
	return or(searchers)
}

func (o or) Expr() string {
	expr := "( "
	for n, searcher := range o {
		if n > 0 {
			expr += " OR "
		}
		expr += searcher.Expr()
	}
	expr += " )"
	return expr
}

func (o or) Params() []interface{} {
	p := make([]interface{}, 0, len(o))
	for _, searcher := range o {
		p = append(p, searcher.Params())
	}
	return p
}

type and []Searcher

// And returns a Search that matches all of the given searches. Useful for
// within an Or Searcher
func And(searchers ...Searcher) Searcher {
	return and(searchers)
}

func (a and) Expr() string {
	expr := "( "
	for n, searcher := range a {
		if n > 0 {
			expr += " AND "
		}
		expr += searcher.Expr()
	}
	expr += " )"
	return expr
}

func (a and) Params() []interface{} {
	p := make([]interface{}, 0, len(a))
	for _, searcher := range a {
		p = append(p, searcher.Params())
	}
	return p
}

// Search is used for a custom (non primary key) search on a table.
//
// Returns the number of items found and an error if any occurred.
func (s *Store) Search(data []Interface, offset int, params ...Searcher) (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
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
	var clause string
	paramVars := make([]interface{}, 0, len(params)+2)
	if len(params) > 0 {
		clause = "WHERE "
		first = true
		for _, param := range params {
			if first {
				first = false
			} else {
				clause += " AND "
			}
			clause += param.Expr()
			paramVars = append(paramVars, param.Params()...)
		}
	}
	var sortBy, dir string
	if s, ok := data[0].(sort); ok {
		sortBy = s.sortBy
		if s.asc {
			dir = "ASC"
		} else {
			dir = "DESC"
		}
	} else {
		sortBy = data[0].Key()
		dir = "ASC"
	}
	sql := "SELECT " + columns + " FROM [" + table + "] " + clause + " ORDER BY [" + sortBy + "] " + dir + " LIMIT ? OFFSET ?;"
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
	table := tableName(data)
	first := true
	var clause string
	paramVars := make([]interface{}, 0, len(params))
	if len(params) > 0 {
		clause = "WHERE "
		for _, param := range params {
			if first {
				first = false
			} else {
				clause += " AND "
			}
			clause += param.Expr()
			paramVars = append(paramVars, param.Params()...)
		}
	}
	sql := "SELECT count(1) FROM [" + table + "] " + clause + ";"
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

// UnknownColumn is an error that occurrs when a search parameter requires a
// column which does not exist for the given type.
type UnknownColumn string

func (u UnknownColumn) Error() string {
	return "unknown column: " + string(u)
}
