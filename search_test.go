package store

import "testing"

func TestSearch(t *testing.T) {
	t.Parallel()
	s, err := newTestStore("testSearch")
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
		&testType{0, "hello world", 4},
		&testType{0, "hello dave", 7},
		&testType{0, "foo bar baz", 12},
		&testType{0, "fubar", 765},
		&testType{0, "abc easy as", 123},
	)
	tests := []struct {
		params        []Searcher
		limit, offset int
		result        []testType
	}{
		{
			[]Searcher{
				Between("number", 5, 15),
			},
			10, 0,
			[]testType{
				{2, "hello dave", 7},
				{3, "foo bar baz", 12},
			},
		},
		{
			[]Searcher{
				Between("number", 5, 150),
			},
			2, 1,
			[]testType{
				{3, "foo bar baz", 12},
				{5, "abc easy as", 123},
			},
		},
		{
			[]Searcher{
				Between("number", 5, 150),
			},
			1, 1,
			[]testType{
				{3, "foo bar baz", 12},
			},
		},
		{
			[]Searcher{
				Like("data", "hello %"),
			},
			10, 0,
			[]testType{
				{1, "hello world", 4},
				{2, "hello dave", 7},
			},
		},
	}

	for n, test := range tests {
		data := make([]testType, test.limit)
		idata := make([]Interface, test.limit)
		for i := 0; i < test.limit; i++ {
			idata[i] = &data[i]
		}
		num, err := s.Search(idata, test.offset, test.params...)
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
