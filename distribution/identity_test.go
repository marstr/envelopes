package distribution_test

import (
	"github.com/marstr/envelopes/distribution"
	"github.com/marstr/envelopes"
)

func ExampleIdentity_Distribute() {
	subject := envelopes.Budget{}.WithBalance(2).WithChildren(map[string]envelopes.Budget{
		"transit":envelopes.Budget{}.WithBalance(4),
		"housing":envelopes.Budget{}.WithBalance(8),
	})

	eff := (*distribution.Identity)(&subject).Distribute(44)

}