package distribution

import (
	"github.com/marstr/envelopes"
)

// UpperBound is an inverted `Priority` Distribution. It will first allocate
// funds to a target Budget, then send the rest to an overflow Distributer.
//
// This is a good way to replicate a budget each pay-period. For example, if
// you are paid each two weeks like me, and you try to spend roughly $100 a week
// on groceries, you could set an UpperBound on the Grocery budget $200. This
// way, each time you get a paycheck, you'll automatically allocate whatever it
// takes to get back to your goal. If you underspend on groceries because you
// were travelling, money would automatically get allocated to your other
// priorities. If you overspent on groceries, funds would automatically be
// allocated to back you up.
type UpperBound struct {
	Target        *envelopes.Budget
	SoughtBalance int64
	Overflow      Distributer
}

// Distribute brings a Budget up to a particular balance, then allocates any
// remaining funds to the overflow. If the amount to be distributed is less
// than or equal to the amount it would take to bring the target budget up to
// the desired balance, the entire amount will be allocated to that budget.
func (ub UpperBound) Distribute(amount int64) (result envelopes.Effect) {
	remaining := ub.SoughtBalance - ub.Target.Balance

	if remaining <= 0 {
		result = ub.Overflow.Distribute(amount)
	} else if split := amount - remaining; split <= 0 {
		result = (*Identity)(ub.Target).Distribute(amount)
	} else {
		result = ub.Overflow.Distribute(split)
		result = result.Add((*Identity)(ub.Target).Distribute(amount - split))
	}
	return
}
