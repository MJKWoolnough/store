// Package store uses an Interface to automatically store information in a sqlite database
package store

import (
	"io"
	"sync"

	"github.com/mxk/go-sqlite/sqlite3"
)

const (
	add = iota
	get
	update
	remove
	getPage
	count
)

// Store is a instance of a sqlite3 connection and numerous prepared statements
type Store struct {
	db         *sqlite3.Conn
	mutex      sync.Mutex
	statements map[string][]statement
}

// NewStore takes the filename of a new or existing sqlite3 database
func NewStore(filename string) (*Store, error) {
	s, err := sqlite3.Open(filename)
	if err != nil {
		return nil, err
	}
	return &Store{
		db:         s,
		statements: make(map[string][]statement),
	}, nil
}

// Close closes the sqlite3 database
func (s *Store) Close() error {
	return s.db.Close()
}

// Register allows a type to be registered with the store, creating the table if
// it does not already exists and prepares the common statements.
func (s *Store) Register(t Interface) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	table := tableName(t)
	tVars := t.Get()

	var sqlVars, sqlParams, setSQLParams, tableVars string
	vars := make([]string, 0, len(tVars))

	first := true
	oFirst := true
	primary := t.Key()

	for typeName, typeVal := range tVars {
		if first {
			first = false
		} else {
			tableVars += ", "
		}
		if primary != typeName {
			if oFirst {
				oFirst = false
			} else {
				sqlVars += ", "
				setSQLParams += ", "
				sqlParams += ", "
			}
		}
		varType := getType(typeVal)
		tableVars += "[" + typeName + "] " + varType
		if primary == typeName {
			tableVars += " PRIMARY KEY AUTOINCREMENT"
		} else {
			sqlVars += "[" + typeName + "]"
			setSQLParams += "[" + typeName + "] = ?"
			sqlParams += "?"
			vars = append(vars, typeName)
		}
	}
	pVars := append(vars, primary)
	sql := "CREATE TABLE IF NOT EXISTS [" + table + "](" + tableVars + ");"
	err := s.db.Exec(sql)
	if err != nil {
		return err
	}
	s.statements[table] = make([]statement, 6)

	sql = "INSERT INTO [" + table + "] (" + sqlVars + ") VALUES (" + sqlParams + ");"
	stmt, err := s.db.Prepare(sql)
	if err != nil {
		return err
	}
	s.statements[table][add] = statement{Stmt: stmt, vars: vars}

	sql = "SELECT " + sqlVars + " FROM [" + table + "] WHERE [" + primary + "] = ? LIMIT 1;"
	stmt, err = s.db.Prepare(sql)
	if err != nil {
		return err
	}
	s.statements[table][get] = statement{Stmt: stmt, vars: vars}

	sql = "UPDATE [" + table + "] SET " + setSQLParams + " WHERE [" + primary + "] = ?;"
	stmt, err = s.db.Prepare(sql)
	if err != nil {
		return err
	}
	s.statements[table][update] = statement{Stmt: stmt, vars: pVars}

	sql = "DELETE FROM [" + table + "] WHERE [" + primary + "] = ?;"
	stmt, err = s.db.Prepare(sql)
	if err != nil {
		return err
	}
	s.statements[table][remove] = statement{Stmt: stmt}

	sql = "SELECT " + sqlVars + ", [" + primary + "] FROM [" + table + "] ORDER BY [" + primary + "] LIMIT ? OFFSET ?;"
	stmt, err = s.db.Prepare(sql)
	if err != nil {
		return err
	}
	s.statements[table][getPage] = statement{Stmt: stmt, vars: pVars}

	sql = "SELECT COUNT(1) FROM [" + table + "];"
	stmt, err = s.db.Prepare(sql)
	if err != nil {
		return err
	}
	s.statements[table][count] = statement{Stmt: stmt}

	return nil
}

// Set will store the given data into the database. The instances of Interface
// do not need to be of the same type.
func (s *Store) Set(ts ...Interface) (id int, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, t := range ts {
		table := tableName(t)
		vars := t.Get()
		p := vars[t.Key()]
		var primary int
		if v, ok := p.(*int); ok {
			primary = *v
		} else {
			err = WrongKeyType{}
			return id, err
		}
		if primary == 0 {
			stmt := s.statements[table][add]
			err = stmt.Exec(unPointers(stmt.Vars(vars))...)
			if err != nil {
				return id, err
			}
			id = int(s.db.LastInsertId())
		} else {
			stmt := s.statements[table][update]
			err = stmt.Exec(unPointers(stmt.Vars(vars))...)
			if err != nil {
				return id, err
			}
			id = primary
		}
	}
	return id, err
}

// Get will retrieve the data from the database. The instances of Interface do
// not need to be of the same type.
func (s *Store) Get(data ...Interface) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, t := range data {
		table := tableName(t)
		stmt := s.statements[table][get]
		vars := t.Get()
		err := stmt.Query(unPointer(vars[t.Key()]))
		if err != nil && err != io.EOF {
			return err
		}
		err = stmt.Scan(stmt.Vars(vars)...)
		if err == io.EOF {
			if v, ok := vars[t.Key()].(*int); ok {
				*v = 0
			}
		} else if err != nil {
			return err
		}
	}
	return nil
}

// GetPage will get data of a single type from the database. The offset is the
// number of items that is to be skipped before filling the data.
//
// The types of data all need to be of the same concrete type.
//
// Returns the number of items retrieved and an error if any occurred.
func (s *Store) GetPage(data []Interface, offset int) (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if len(data) < 1 {
		return 0, nil
	}
	table := tableName(data[0])
	stmt := s.statements[table][getPage]
	var (
		err error
		pos int
	)
	for err = stmt.Query(len(data), offset); err == nil; err = stmt.Next() {
		if typeName := tableName(data[pos]); typeName != table {
			err = UnmatchedType{table, typeName}
		} else {
			err = stmt.Scan(stmt.Vars(data[pos].Get())...)
			pos++
		}
	}
	if err != nil && err != io.EOF {
		return pos, err
	}
	return pos, nil
}

// Delete removes data from the database. The instances of Interface do not
// need to be of the same type.
func (s *Store) Delete(ts ...Interface) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, t := range ts {
		table := tableName(t)
		err := s.statements[table][remove].Exec(unPointer(t.Get()[t.Key()]))
		if err != nil {
			return err
		}
	}
	return nil
}

// Count returns the number of entries for the given type
func (s *Store) Count(t Interface) (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	table := tableName(t)
	stmt := s.statements[table][count]
	err := stmt.Query()
	if err != nil {
		return 0, err
	}
	num := 0
	err = stmt.Scan(&num)
	if err != nil {
		return 0, err
	}
	return num, nil
}

//Errors

// WrongKeyType is an error given when the primary key is not of type int.
type WrongKeyType struct{}

func (WrongKeyType) Error() string {
	return "primary key needs to be int"
}

// UnmatchedType is an error given when an instance of Interface does not match
// a previous instance.
type UnmatchedType struct {
	MainType, ThisType string
}

func (u UnmatchedType) Error() string {
	return "expecting type " + u.MainType + ", got type " + u.ThisType
}
