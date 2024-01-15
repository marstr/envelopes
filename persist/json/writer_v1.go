package json

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/persist"
)

// WriterV1 knows how to navigate the envelopes object model and stash each individual component of an object.
type WriterV1 struct {
	// Writes the serialized form of an object to persistent memory. Must not be nil.
	persist.Stasher

	// Allow recursive calls to Write to invoke the top-level WriterV1. If this is nil, WriterV1 uses itself.
	loopback persist.Writer
}

func NewWriterV1(stasher persist.Stasher) (*WriterV1, error) {
	retval := &WriterV1{
		Stasher: stasher,
	}
	retval.loopback = retval
	return retval, nil
}

func NewWriterV1WithLoopback(stasher persist.Stasher, loopback persist.Writer) (*WriterV1, error) {
	retval := &WriterV1{
		Stasher:  stasher,
		loopback: loopback,
	}
	return retval, nil
}

func (dw WriterV1) WriteTransaction(ctx context.Context, subject envelopes.Transaction) error {
	if subject.State == nil {
		subject.State = &envelopes.State{}
	}

	var parent envelopes.ID
	if parentCount := len(subject.Parents); parentCount > 1 {
		return fmt.Errorf("transaction %s has multiple parents (%d), and cannot be represented in a JSONV1 format", subject.ID(), parentCount)
	} else if parentCount == 1 {
		parent = subject.Parents[0]
	}

	var toMarshal TransactionV1
	toMarshal.Amount = BalanceV1(subject.Amount)
	toMarshal.Parent = parent
	toMarshal.State = subject.State.ID()
	toMarshal.Comment = subject.Comment
	toMarshal.Merchant = subject.Merchant
	toMarshal.ActualTime = subject.ActualTime
	toMarshal.EnteredTime = subject.EnteredTime
	toMarshal.PostedTime = subject.PostedTime
	toMarshal.Committer.FullName = subject.Committer.FullName
	toMarshal.Committer.Email = subject.Committer.Email
	toMarshal.RecordId = BankRecordIDV1(subject.RecordID)

	err := dw.loopback.WriteState(ctx, *subject.State)
	if err != nil {
		return err
	}

	marshaled, err := json.Marshal(toMarshal)
	if err != nil {
		return err
	}

	return dw.Stash(ctx, subject.ID(), marshaled)
}

func (dw WriterV1) WriteState(ctx context.Context, subject envelopes.State) error {
	if subject.Accounts == nil {
		subject.Accounts = make(envelopes.Accounts, 0)
	}
	err := dw.loopback.WriteAccounts(ctx, subject.Accounts)
	if err != nil {
		return err
	}

	if subject.Budget == nil {
		subject.Budget = &envelopes.Budget{}
	}
	err = dw.loopback.WriteBudget(ctx, *subject.Budget)
	if err != nil {
		return err
	}

	var toMarshal StateV1
	toMarshal.Accounts = subject.Accounts.ID()
	toMarshal.Budget = subject.Budget.ID()

	marshaled, err := json.Marshal(toMarshal)
	if err != nil {
		return err
	}

	return dw.Stash(ctx, subject.ID(), marshaled)
}

func (dw WriterV1) WriteBudget(ctx context.Context, subject envelopes.Budget) error {
	if subject.Children == nil {
		subject.Children = make(map[string]*envelopes.Budget, 0)
	}
	for _, child := range subject.Children {
		err := dw.loopback.WriteBudget(ctx, *child)
		if err != nil {
			return err
		}
	}

	var toMarshal BudgetV1
	toMarshal.Balance = BalanceV1(subject.Balance)
	toMarshal.Children = make(map[string]envelopes.ID, len(subject.Children))
	for name, child := range subject.Children {
		toMarshal.Children[name] = child.ID()
	}

	marshaled, err := json.Marshal(toMarshal)
	if err != nil {
		return err
	}

	return dw.Stash(ctx, subject.ID(), marshaled)
}

func (dw WriterV1) WriteAccounts(ctx context.Context, subject envelopes.Accounts) error {
	marshaled, err := json.Marshal(subject)
	if err != nil {
		return err
	}

	return dw.Stash(ctx, subject.ID(), marshaled)
}
