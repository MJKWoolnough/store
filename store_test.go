package store

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
)

type testStore struct {
	*Store
	path string
}

func (t testStore) Close() {
	t.Store.Close()
	os.RemoveAll(t.path)
}

func newTestStore(dirname string) (*testStore, error) {
	p, err := ioutil.TempDir(os.TempDir(), dirname)
	if err != nil {
		return nil, err
	}
	s, err := NewStore(path.Join(p, "test.db"))
	if err != nil {
		return nil, err
	}
	return &testStore{s, p}, nil
}

type testType struct {
	ID     int
	Data   string
	Number int64
}

func (t *testType) Get() TypeMap {
	return TypeMap{
		"id":     &t.ID,
		"data":   &t.Data,
		"number": &t.Number,
	}
}

func (t *testType) Key() string {
	return "id"
}

func TestSetGet(t *testing.T) {
	s, err := newTestStore("test")
	defer s.Close()
	if err != nil {
		t.Errorf("received unexpected error: %s", err)
	}
	err = s.Register(new(testType))
	if err != nil {
		t.Errorf("received unexpected error: %s", err)
	}
	tests := []struct {
		testType
		redo bool
	}{
		{testType{0, "HELLO", 12}, true},
		{testType{0, "WORLD", 14}, false},
		{testType{2, "Barry", 11}, true},
	}
	for n := range tests {
		tests[n].ID, err = s.Set(&tests[n].testType)
		test := tests[n]
		if err != nil {
			t.Errorf("test 1 %d: received unexpected error: %s", n+1, err)
			break
		}
		toRet := new(testType)
		toRet.ID = test.ID
		err = s.Get(toRet)
		if err != nil {
			t.Errorf("test 1 %d: received unexpected error: %s", n+1, err)
		} else if toRet.ID != test.ID || toRet.Data != test.Data || toRet.Number != test.Number {
			t.Errorf("test 1 %d: returned data does not match expected\nexpecting: %v\ngot: %v", n+1, test.testType, toRet)
		}
	}
	for n, test := range tests {
		if !test.redo {
			continue
		}
		toRet := new(testType)
		toRet.ID = test.ID
		err = s.Get(toRet)
		if err != nil {
			t.Errorf("test 2 %d: received unexpected error: %s", n+1, err)
		} else if toRet.ID != test.ID || toRet.Data != test.Data || toRet.Number != test.Number {
			t.Errorf("test 2 %d: returned data does not match expected\nexpecting: %v\ngot: %v", n+1, test.testType, toRet)
		}
	}
}
