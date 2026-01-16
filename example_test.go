package store_test

import (
	"fmt"

	"vimagination.zapto.org/store"
)

type User struct {
	ID          int64
	Name, Email string
	Age         int
}

func Example() {
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
