package store

import (
	"database/sql"
	"testing"
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
	s := New(new(sql.DB))
	s.Register(new(TestType1))
}
