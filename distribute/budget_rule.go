package distribute

import (
	"context"
	"github.com/marstr/envelopes"
)

// BudgetRule is a Distributor which applies a balance to a specified envelopes.Budget.
type BudgetRule envelopes.Budget

func (b *BudgetRule) Distribute(_ context.Context, balance envelopes.Balance) error {
	b.Balance = b.Balance.Add(balance)
	return nil
}
