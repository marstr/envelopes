package distribute

import (
	"context"
	"github.com/marstr/envelopes"
)

// PercentageRule allows a balance to be split proportionately between multiple distributors. Generally, the values
// assigned to each Distributor should be in the range [0, 1], and collectively should sum to 1.
type PercentageRule struct {
	/* The primary mechanism used to distribute funds. */
	Targets map[Distributor]float64

	/* The Distributor that will be debited or credited to correct for any rounding errors that remain after distribution. */
	Err Distributor
}

// NewPercentageRule creates a new empty PercentageRule that will put any error in the given Distributor.
func NewPercentageRule(capacity uint, err Distributor) *PercentageRule {
	return &PercentageRule{
		Targets: make(map[Distributor]float64, capacity),
		Err: err,
	}
}

// AddRule appends a new Distributor that will receive a portion of funds when Distribute is called.
func (pr PercentageRule) AddRule(equity float64, distributor Distributor) {
	pr.Targets[distributor] = equity
}

// Distribute takes funds from balance and splits it up equitably among the rules that were previously added using
// AddRule.
func (pr PercentageRule) Distribute(ctx context.Context, balance envelopes.Balance) error {
	remaining := balance.Copy()

	for target, scale := range pr.Targets {
		current := balance.Scale(scale)

		if err := target.Distribute(ctx, current); err != nil {
			return err
		}

		remaining = remaining.Sub(current)
	}

	return pr.Err.Distribute(ctx, remaining);
}
