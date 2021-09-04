package distribute

import (
	"context"
	"github.com/marstr/envelopes"
)

// BringToRule will evaluate the balance of an envelopes.Budget and distribute available funds necessary to set its
// balance a given amount.
type BringToRule struct {
	Desired envelopes.Balance
	Target *envelopes.Budget
	Leftover Distributor
}

// NewBringToRule creates a new fully instantiated BringToRule.
func NewBringToRule(target *envelopes.Budget, amount envelopes.Balance, remaining Distributor) *BringToRule {
	return &BringToRule{
		Desired:  amount,
		Target:   target,
		Leftover: remaining,
	}
}

// Distribute considers the Target's RecursiveBalance and allocates funds necessary to have that balance end at the
// given Desired amount. If balance is insufficient, Leftover will have a negative balance distributed to make up for
// it.
func (btr BringToRule) Distribute(ctx context.Context, balance envelopes.Balance) error {
	original := btr.Target.RecursiveBalance()
	delta := btr.Desired.Sub(original)

	btr.Target.Balance = btr.Target.Balance.Add(delta)

	return btr.Leftover.Distribute(ctx, balance.Sub(delta))
}
