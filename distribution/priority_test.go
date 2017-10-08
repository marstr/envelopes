package distribution_test

import (
	"fmt"

	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/distribution"
)

func ExamplePriority_Distribute() {
	groceries, transit := &envelopes.Budget{Name: "Groceries"},
		&envelopes.Budget{Name: "Transit"}

	subject := distribution.Priority{
		Target:    (*distribution.Identity)(groceries),
		Magnitude: 1000,
		Overflow:  (*distribution.Identity)(transit),
	}

	fmt.Println(subject.Distribute(900))
	fmt.Println(subject.Distribute(1100))
	fmt.Println(subject.Distribute(-900))
	fmt.Println(subject.Distribute(-1100))
	// Output:
	// ["Groceries":$9.00]
	// ["Groceries":$10.00 "Transit":$1.00]
	// ["Groceries":$-9.00]
	// ["Groceries":$-10.00 "Transit":$-1.00]
}
