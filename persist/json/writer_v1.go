package json

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

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

// Write uses the persist.Stasher it is composed of to write the given Envelopes object to a persistent storage space.
func (dw WriterV1) Write(ctx context.Context, subject envelopes.IDer) error {
	// In recursive methods, it is easy to detect that a context has been cancelled between calls to itself.
	// Must have default clause to prevent blocking.
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Intentionally Left Blank
	}

	switch subject.(type) {
	case envelopes.Transaction:
		return dw.writeTransaction(ctx, subject.(envelopes.Transaction))
	case *envelopes.Transaction:
		return dw.writeTransaction(ctx, *subject.(*envelopes.Transaction))
	case envelopes.State:
		return dw.writeState(ctx, subject.(envelopes.State))
	case *envelopes.State:
		return dw.writeState(ctx, *subject.(*envelopes.State))
	case envelopes.Budget:
		return dw.writeBudget(ctx, subject.(envelopes.Budget))
	case *envelopes.Budget:
		return dw.writeBudget(ctx, *subject.(*envelopes.Budget))
	case envelopes.Accounts:
		return dw.writeAccounts(ctx, subject.(envelopes.Accounts))
	case *envelopes.Accounts:
		return dw.writeAccounts(ctx, *subject.(*envelopes.Accounts))
	default:
		return fmt.Errorf("unknown type: %s", reflect.TypeOf(subject).Name())
	}
}

func (dw WriterV1) writeTransaction(ctx context.Context, subject envelopes.Transaction) error {
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

	err := dw.loopback.Write(ctx, subject.State)
	if err != nil {
		return err
	}

	marshaled, err := json.Marshal(toMarshal)
	if err != nil {
		return err
	}

	return dw.Stash(ctx, subject.ID(), marshaled)
}

func (dw WriterV1) writeState(ctx context.Context, subject envelopes.State) error {
	if subject.Accounts == nil {
		subject.Accounts = make(envelopes.Accounts, 0)
	}
	err := dw.loopback.Write(ctx, subject.Accounts)
	if err != nil {
		return err
	}

	if subject.Budget == nil {
		subject.Budget = &envelopes.Budget{}
	}
	err = dw.loopback.Write(ctx, subject.Budget)
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

func (dw WriterV1) writeBudget(ctx context.Context, subject envelopes.Budget) error {
	if subject.Children == nil {
		subject.Children = make(map[string]*envelopes.Budget, 0)
	}
	for _, child := range subject.Children {
		err := dw.loopback.Write(ctx, child)
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

func (dw WriterV1) writeAccounts(ctx context.Context, subject envelopes.Accounts) error {
	marshaled, err := json.Marshal(subject)
	if err != nil {
		return err
	}

	return dw.Stash(ctx, subject.ID(), marshaled)
}