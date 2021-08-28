package distribute

import (
	"context"
	"github.com/marstr/envelopes"
)

type PercentageRule struct {
	/* The primary mechanism used to distribute funds. */
	Targets map[Distributor]float64

	/* The Distributor that will be debited or credited to correct for any rounding errors that remain after distribution. */
	Err Distributor
}

func NewPercentageRule(capacity uint, err Distributor) *PercentageRule {
	return &PercentageRule{
		Targets: make(map[Distributor]float64, capacity),
		Err: err,
	}
}

func (pr PercentageRule) AddRule(equity float64, distributor Distributor) {
	pr.Targets[distributor] = equity
}

func (pr PercentageRule) Distribute(ctx context.Context, balance envelopes.Balance) error {
	var remaining envelopes.Balance
	remaining.Set(balance)

	for target, scale := range pr.Targets {
		var current envelopes.Balance
		current.Set(balance)
		current = current.Scale(scale)

		if err := target.Distribute(ctx, current); err != nil {
			return err
		}

		remaining = remaining.Sub(current)
	}

	return pr.Err.Distribute(ctx, remaining);
}
