package distribution_test

import (
	"fmt"

	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/distribution"
)

func ExampleUpperBound_Distribute() {
	grocery, transit := &envelopes.Budget{
		Name:    "Groceries",
		Balance: 8420,
	}, &envelopes.Budget{
		Name:    "Transit",
		Balance: 2278,
	}

	subject := distribution.UpperBound{
		Target:        grocery,
		SoughtBalance: 10000,
		Overflow:      (*distribution.Identity)(transit),
	}

	fmt.Println(subject.Distribute(4000))
	fmt.Println(subject.Distribute(1000))
	// Output:
	// ["Transit":$24.20 "Groceries":$15.80]
	// ["Groceries":$10.00]
}
