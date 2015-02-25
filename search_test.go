package store

import (
	"reflect"
	"testing"
)

type match struct {
	col string
	num *int64
}

func (m match) SQL() string {
	return "[" + m.col + "] = ?"
}

func (m match) Vars() []interface{} {
	return []interface{}{m.num}
}

func TestSearch(t *testing.T) {
	s, err := newTestStore()
	defer s.Close()
	if err != nil {
		t.Error(err)
		return
	}
	err = s.Register(new(testType))
	if err != nil {
		t.Error(err)
		return
	}
	values := []*testType{
		&testType{0, "One", 45},
		&testType{0, "Two", 657},
		&testType{0, "Three", 43},
		&testType{0, "Four", 3},
		&testType{0, "Five", 83},
		&testType{0, "Six", 214},
		&testType{0, "Seven", 1784},
		&testType{0, "Eight", 90},
		&testType{0, "Nine", 214},
		&testType{0, "Ten", 45},
		&testType{0, "Eleven", 45},
		&testType{0, "Twelve", 77},
	}
	for _, value := range values {
		if err = s.Set(value); err != nil {
			t.Error(err)
			return
		}
	}

	var number int64

	matchNumber := s.NewSearch(new(testType))
	matchNumber.Sort = []SortBy{{"ID", true}}
	matchNumber.Filter = match{"Number", &number}
	matchNumberPrepared, err := matchNumber.Prepare()
	if err != nil {
		t.Error(err)
		return
	}

	tests := []struct {
		number  int64
		results []*testType
	}{
		{657, []*testType{values[1]}},
		{45, []*testType{values[0], values[9], values[10]}},
		{123, []*testType{}},
	}

	for n, test := range tests {
		number = test.number
		tts := make([]testType, 10)
		vars := make([]interface{}, 10)
		for i := 0; i < 10; i++ {
			vars[i] = &tts[i]
		}
		found, err := matchNumberPrepared.GetPage(vars, 0)
		if err != nil {
			t.Error(err)
			return
		}
		vars = vars[:found]
		if len(vars) != len(test.results) {
			t.Errorf("test 1-%d: expecting %d results, got %d", n+1, len(test.results), len(vars))
		} else {
			for num, tt := range test.results {
				if !reflect.DeepEqual(tt, &tts[num]) {
					t.Errorf("test 1-%d-%d: expecting %v, got %v", n+1, num+1, tt, &tts[num])
				}
			}
		}
	}
}
