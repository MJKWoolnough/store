# store
--
    import "vimagination.zapto.org/store"

Package store automatically configures a database to store structured
information in an sql database

## Usage

```go
var (
	ErrDBClosed         = errors.New("database already closed")
	ErrNoPointerStruct  = errors.New("given variable is not a pointer to a struct")
	ErrNoKey            = errors.New("could not determine key")
	ErrDuplicateColumn  = errors.New("duplicate column name found")
	ErrUnregisteredType = errors.New("type not registered")
	ErrInvalidType      = errors.New("invalid type")
)
```

#### type And

```go
type And []Filter
```


#### func (And) SQL

```go
func (a And) SQL() string
```

#### func (And) Vars

```go
func (a And) Vars() []interface{}
```

#### type Filter

```go
type Filter interface {
	SQL() string
	Vars() []interface{}
}
```


#### type Or

```go
type Or []Filter
```


#### func (Or) SQL

```go
func (o Or) SQL() string
```

#### func (Or) Vars

```go
func (o Or) Vars() []interface{}
```

#### type PreparedSearch

```go
type PreparedSearch struct {
}
```


#### func (*PreparedSearch) Count

```go
func (p *PreparedSearch) Count() (int, error)
```

#### func (*PreparedSearch) GetPage

```go
func (p *PreparedSearch) GetPage(is []interface{}, offset int) (int, error)
```

#### type Search

```go
type Search struct {
	Sort   []SortBy
	Filter Filter
}
```


#### func (*Search) Prepare

```go
func (s *Search) Prepare() (*PreparedSearch, error)
```

#### type SortBy

```go
type SortBy struct {
	Column string
	Asc    bool
}
```


#### type Store

```go
type Store struct {
}
```


#### func  New

```go
func New(dataSourceName string) (*Store, error)
```

#### func (*Store) Close

```go
func (s *Store) Close() error
```

#### func (*Store) Count

```go
func (s *Store) Count(i interface{}) (int, error)
```

#### func (*Store) Get

```go
func (s *Store) Get(is ...interface{}) error
```

#### func (*Store) GetPage

```go
func (s *Store) GetPage(is []interface{}, offset int) (int, error)
```

#### func (*Store) NewSearch

```go
func (s *Store) NewSearch(i interface{}) *Search
```

#### func (*Store) Register

```go
func (s *Store) Register(is ...interface{}) error
```

#### func (*Store) Remove

```go
func (s *Store) Remove(is ...interface{}) error
```

#### func (*Store) Set

```go
func (s *Store) Set(is ...interface{}) error
```
