package distribute

import (
	"context"
	"github.com/marstr/collection"
	"github.com/marstr/envelopes"
)

// PriorityRule walks a list of target distributions giving a specified amount of funds and removing those funds from the
// given balance. It will stop its walk when the balance is exhausted, or there is only one remaining priority. The last
// priority will receive all remaining funds.
type PriorityRule struct {
	priority *collection.LinkedList
	Leftover Distributor
}

type priorityEntry struct {
	Amount envelopes.Balance
	Distributor
}

// NewPriorityRule creates a new empty PriorityRule that will send unallocated funds to leftover.
func NewPriorityRule(leftover Distributor) *PriorityRule {
	return &PriorityRule{
		priority: collection.NewLinkedList(),
		Leftover: leftover,
	}
}

// AddRule will put a new entry at the back of the line for receiving a given amount of funds.
func (pr PriorityRule) AddRule(rule Distributor, amount envelopes.Balance) {
	pr.priority.AddBack(priorityEntry{
		Amount:      amount,
		Distributor: rule,
	})
}

// Distribute will step through each Distributor that was added in previous calls to AddRule. Surplus or shortfall will
// be passed on to Leftover.
func (pr PriorityRule) Distribute(ctx context.Context, balance envelopes.Balance) error {
	for item := range pr.priority.Enumerate(ctx.Done()) {
		cast := item.(priorityEntry)
		if err := cast.Distribute(ctx, cast.Amount); err != nil {
			return err
		}

		balance = balance.Sub(cast.Amount)
	}

	return pr.Leftover.Distribute(ctx, balance)
}
