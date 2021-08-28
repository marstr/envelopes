// Package distribute seeks to enable people to transfer money between their budgets with complex, rich rules. For
// instance, people may aggregate paychecks throughout the month, then transfer their cash into the appropriate budgets
// to prepare for monthly expenses, save for the future, or just allocate funds to have fun.
package distribute

import (
	"context"
	"github.com/marstr/envelopes"
)

// Distributor is the fundamental building block that allows for composable distribution rules.
type Distributor interface {
	// Distribute takes funds and divides them into a budget.
	Distribute(ctx context.Context, balance envelopes.Balance) error
}