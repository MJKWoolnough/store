package store

import (
	"testing"

	_ "github.com/mxk/go-sqlite/sqlite3"
)

func newTestStore() (*Store, error) {
	return New("sqlite3", ":memory:")
}

type testType struct {
	ID     int
	Data   string
	Number int64
}

func TestSetGet(t *testing.T) {
	s, err := newTestStore()
	defer s.Close()
	if err != nil {
		t.Errorf("received unexpected error: %s", err)
		return
	}
	err = s.Register(new(testType))
	if err != nil {
		t.Errorf("received unexpected error: %s", err)
		return
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
		err = s.Set(&tests[n].testType)
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

func TestRemove(t *testing.T) {
	s, err := newTestStore()
	defer s.Close()
	if err != nil {
		t.Errorf("received unexpected error: %s", err)
		return
	}
	err = s.Register(new(testType))
	if err != nil {
		t.Errorf("received unexpected error: %s", err)
		return
	}
	tests := []testType{
		{0, "HELLO", 12},
		{0, "WORLD", 14},
		{0, "Barry", 11},
	}
	for n, test := range tests {
		err = s.Set(&test)
		if err != nil {
			t.Errorf("test %d-1: received unexpected error: %s", n+1, err)
		}
		tt := testType{ID: test.ID}
		err = s.Get(&tt)
		if err != nil {
			t.Errorf("test %d-2: received unexpected error: %s", n+1, err)
		} else if tt.ID != test.ID || tt.Data != test.Data || tt.Number != test.Number {
			t.Errorf("test %d-2: returned data does not match expected\nexpecting: %v\ngot: %v", n+1, test, tt)
		}
		err = s.Remove(&tt)
		if err != nil {
			t.Errorf("test %d-3: received unexpected error: %s", n+1, err)
		}
		tt = testType{ID: test.ID}
		err = s.Get(&tt)
		if err != nil {
			t.Errorf("test %d-4: received unexpected error: %s", n+1, err)
		} else if tt.ID != 0 {
			t.Errorf("test %d-4: expecting id 0, got %d", n+1, tt.ID)
		}
	}
}

func TestGetPage(t *testing.T) {
	s, err := newTestStore()
	defer s.Close()
	if err != nil {
		t.Errorf("received unexpected error: %s", err)
		return
	}
	err = s.Register(new(testType))
	if err != nil {
		t.Errorf("received unexpected error: %s", err)
		return
	}
	s.Set(
		&testType{0, "BEEP", 1},
		&testType{0, "3.14159", 42},
		&testType{0, "hats", 420},
		&testType{0, "string", 25},
	)
	tests := []struct {
		toGet, offset int
		result        []testType
	}{
		{1, 0,
			[]testType{
				{1, "BEEP", 1},
			},
		},
		{1, 1,
			[]testType{
				{2, "3.14159", 42},
			},
		},
		{2, 2,
			[]testType{
				{3, "hats", 420},
				{4, "string", 25},
			},
		},
		{2, 3,
			[]testType{
				{4, "string", 25},
			},
		},
	}
	for n, test := range tests {
		data := make([]testType, test.toGet)
		idata := make([]interface{}, test.toGet)
		for i := 0; i < test.toGet; i++ {
			idata[i] = &data[i]
		}
		num, err := s.GetPage(idata, test.offset)
		if err != nil {
			t.Errorf("test %d: received unexpected error: %s", n+1, err)
		} else if num != len(test.result) {
			t.Errorf("test %d: received %d results, expecting %d", n+1, num, len(test.result))
		} else {
			for m, d := range test.result {
				if d.ID != data[m].ID || d.Data != data[m].Data || d.Number != data[m].Number {
					t.Errorf("test %d %d: returned data does not match expected\nexpecting: %v\ngot: %v", n+1, m, d, data[m])
				}
			}
		}
	}
}
