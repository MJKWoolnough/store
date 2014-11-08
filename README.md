# store
--
    import "github.com/MJKWoolnough/store"

Store uses an Interface to automatically store information in a sqlite database

## Usage

#### type Between

```go
type Between struct {
}
```

Between is a searcher which searches for values between (inclusive) the given
values.

#### func (Between) Expr

```go
func (Between) Expr(col string) string
```

#### func (Between) Params

```go
func (b Between) Params() []interface{}
```

#### type Interface

```go
type Interface interface {
	// Get returns a TypeMap of column name -> *type.
	Get() TypeMap
	// Key returns the column name of the primary key.
	Key() string
}
```

Interface is the data interface for the database.

#### type InterfaceTable

```go
type InterfaceTable interface {
	Interface
	// TableName returns the custom table name for the type.
	TableName() string
}
```

InterfaceTable is an extension of Interface which allows a custom table name.

#### type Like

```go
type Like string
```

Like implements a search which uses the LIKE syntax in a WHERE clause.

#### func (Like) Expr

```go
func (Like) Expr(col string) string
```

#### func (Like) Params

```go
func (l Like) Params() []interface{}
```

#### type NoParams

```go
type NoParams struct{}
```

NoParams is an error that occurs when no search parameters are given to Search.

#### func (NoParams) Error

```go
func (NoParams) Error() string
```

#### type Searcher

```go
type Searcher interface {
	// Expr returns a sqlite snippet used in the WHERE clause.
	Expr(string) string
	//Params returns the required parameters for the sqlite snippet.
	Params() []interface{}
}
```

Searcher

#### type Store

```go
type Store struct {
}
```


#### func  NewStore

```go
func NewStore(filename string) (*Store, error)
```
NewStore takes the filename of a new or existing sqlite3 database

#### func (*Store) Close

```go
func (s *Store) Close() error
```
Close closes the sqlite3 database

#### func (*Store) Delete

```go
func (s *Store) Delete(ts ...Interface) error
```
Delete removes data from the database. The instances of Interface do not need to
be of the same type.

#### func (*Store) Get

```go
func (s *Store) Get(data ...Interface) error
```
Get will retrieve the data from the database. The instance os Interface do not
need to be of the same type.

#### func (*Store) GetPage

```go
func (s *Store) GetPage(offset int, data ...Interface) (int, error)
```
GetPage will get data of a single type from the database. The offset is the
number of items that is to be skipped before filling the data.

The types of data all need to be of the same concrete type.

Returns the number of items retrieved and an error if any occurred.

#### func (*Store) Register

```go
func (s *Store) Register(t Interface) error
```
Register allows a type to be registered with the store, creating the table if it
does not already exists and prepares the common statements.

#### func (*Store) Search

```go
func (s *Store) Search(params map[string]Searcher, offset int, data ...Interface) (int, error)
```
Search is used for a custom (non primary key) search on a table.

Returns the number of items found and an error if any occurred.

#### func (*Store) Set

```go
func (s *Store) Set(ts ...Interface) (id int, err error)
```
Set will store the given data into the database. The instances of Interface do
not need to be of the same type.

#### type TypeMap

```go
type TypeMap map[string]interface{}
```

TypeMap is a convenience type to reduce typing.

#### type UnknownColumn

```go
type UnknownColumn string
```

UnknownColumn is an error that occurrs when a search parameter requires a column
which does not exist for the given type.

#### func (UnknownColumn) Error

```go
func (u UnknownColumn) Error() string
```

#### type UnmatchedType

```go
type UnmatchedType struct {
	MainType, ThisType string
}
```

UnmatchedType is an error given when an instance of Interface does not match a
previous instance.

#### func (UnmatchedType) Error

```go
func (u UnmatchedType) Error() string
```

#### type WrongKeyType

```go
type WrongKeyType struct{}
```

WrongKeyType is an error given when the primary key is not of type int.

#### func (WrongKeyType) Error

```go
func (WrongKeyType) Error() string
```
