# store

[![CI](https://github.com/MJKWoolnough/store/actions/workflows/go-checks.yml/badge.svg)](https://github.com/MJKWoolnough/store/actions)
[![Go Reference](https://pkg.go.dev/badge/vimagination.zapto.org/store.svg)](https://pkg.go.dev/vimagination.zapto.org/store)
[![Go Report Card](https://goreportcard.com/badge/vimagination.zapto.org/store)](https://goreportcard.com/report/vimagination.zapto.org/store)

--
    import "vimagination.zapto.org/store"

A lightweight Go package for automatically configuring SQL databases to store structured information.

It inspects Go structs, creates the necessary tables, and provides a simple API for inserting, retrieving, updating, deleting, and searching records.

## Highlights

 - Automatic table creation based on struct layout.
 - Flexible filtering functions.
 - Sorting and pagination.

## Usage

```go
package main

import (
	"fmt"

	"vimagination.zapto.org/store"
)

type User struct {
	ID          int64
	Name, Email string
	Age         int
}

func main() {
	s, err := store.New(":memory:")
	if err != nil {
		fmt.Println(err)

		return
	}

	defer s.Close()

	if err := s.Register(new(User)); err != nil {
		fmt.Println(err)

		return
	}

	users := [...]User{
		{Name: "Alice", Email: "alice@email.com", Age: 20},
		{Name: "Bob", Email: "robert@email.com", Age: 25},
		{Name: "Charlie", Email: "charlie@email.com", Age: 20},
		{Name: "Dana", Email: "dana@email.com", Age: 40},
	}

	if err := s.Set(&users[0], &users[1], &users[2], &users[3]); err != nil {
		fmt.Println(err)

		return
	}

	get := User{ID: users[1].ID}

	if err := s.Get(&get); err != nil {
		fmt.Println(err)
	}

	fmt.Println(get.Name)

	// output:
	// Bob
}
```

## Documentation

Full API docs can be found at:

https://pkg.go.dev/vimagination.zapto.org/store
