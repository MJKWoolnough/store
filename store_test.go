package store

import (
	"database/sql"
	"testing"

	_ "github.com/mxk/go-sqlite/sqlite3"
)

type TestType2 struct {
	a      string
	Person int
	Pid    int    `key:"1"`
	DOB    string `store:"DateOfBirth"`
}

type TestType1 struct {
	ID        int
	Name      string
	hidden    bool
	NotHidden float32
	PtrInt    *int64
	DontShow  string `store:"-"`
	TestType2
}

func TestRegister(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	defer db.Close()
	s := New(db)
	s.Register(new(TestType1))
}
