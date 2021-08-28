package distribute

import (
	"context"
	"github.com/marstr/envelopes"
)

type BringToRule struct {
	Desired envelopes.Balance
	Target *envelopes.Budget
	Leftover Distributor
}

func NewBringToRule(target *envelopes.Budget, amount envelopes.Balance, remaining Distributor) *BringToRule {
	return &BringToRule{
		Desired:  amount,
		Target:   target,
		Leftover: remaining,
	}
}

func (btr BringToRule) Distribute(ctx context.Context, balance envelopes.Balance) error {
	original := btr.Target.RecursiveBalance()
	delta := btr.Desired.Sub(original)

	btr.Target.Balance = btr.Target.Balance.Add(delta)

	return btr.Leftover.Distribute(ctx, balance.Sub(delta))
}
