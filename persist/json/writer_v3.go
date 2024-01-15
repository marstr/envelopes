package json

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/persist"
)

type Writer = WriterV3

// WriterV3 knows how to navigate the envelopes object model and stash each individual component of an object.
type WriterV3 struct {
	// Writes the serialized form of an object to persistent memory. Must not be nil.
	persist.Stasher

	// Allow recursive calls to Write to invoke the top-level WriterV3. If this is nil, WriterV3 uses itself.
	loopback persist.Writer
}

func NewWriterV3(stasher persist.Stasher) (*WriterV3, error) {
	retval := &WriterV3{
		Stasher: stasher,
	}
	retval.loopback = retval
	return retval, nil
}

func NewWriterV3WithLoopback(stasher persist.Stasher, loopback persist.Writer) (*WriterV3, error) {
	retval := &WriterV3{
		Stasher:  stasher,
		loopback: loopback,
	}
	return retval, nil
}

// Write uses the persist.Stasher it is composed of to write the given Envelopes object to a persistent storage space.
func (dw WriterV3) Write(ctx context.Context, subject envelopes.IDer) error {
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

func (dw WriterV3) writeTransaction(ctx context.Context, subject envelopes.Transaction) error {
	if subject.State == nil {
		subject.State = &envelopes.State{}
	}

	var toMarshal TransactionV2
	toMarshal.Amount = BalanceV2(subject.Amount)
	toMarshal.Parent = subject.Parents
	toMarshal.State = subject.State.ID()
	toMarshal.Comment = subject.Comment
	toMarshal.Merchant = subject.Merchant
	toMarshal.ActualTime = subject.ActualTime
	toMarshal.EnteredTime = subject.EnteredTime
	toMarshal.PostedTime = subject.PostedTime
	toMarshal.Committer.FullName = subject.Committer.FullName
	toMarshal.Committer.Email = subject.Committer.Email
	toMarshal.RecordId = BankRecordIDV2(subject.RecordID)

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

func (dw WriterV3) writeState(ctx context.Context, subject envelopes.State) error {
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

	var toMarshal StateV2
	toMarshal.Accounts = subject.Accounts.ID()
	toMarshal.Budget = subject.Budget.ID()

	marshaled, err := json.Marshal(toMarshal)
	if err != nil {
		return err
	}

	return dw.Stash(ctx, subject.ID(), marshaled)
}

func (dw WriterV3) writeBudget(ctx context.Context, subject envelopes.Budget) error {
	if subject.Children == nil {
		subject.Children = make(map[string]*envelopes.Budget, 0)
	}
	for _, child := range subject.Children {
		err := dw.loopback.Write(ctx, child)
		if err != nil {
			return err
		}
	}

	var toMarshal BudgetV2
	toMarshal.Balance = BalanceV2(subject.Balance)
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

func (dw WriterV3) writeAccounts(ctx context.Context, subject envelopes.Accounts) error {
	accountNames := make([]string, 0, len(subject))
	for k := range subject {
		accountNames = append(accountNames, k)
	}
	sort.Strings(accountNames)

	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)

	var err error
	_, err = fmt.Fprint(buf, "{")
	if err != nil {
		return err
	}

	for i := range accountNames {
		err = encoder.Encode(accountNames[i])
		if err != nil {
			return err
		}
		buf.Truncate(buf.Len() - 1)
		_, err = fmt.Fprint(buf, ":")
		if err != nil {
			return err
		}

		err = encoder.Encode(BalanceV3(subject[accountNames[i]]))
		if err != nil {
			return err
		}
		buf.Truncate(buf.Len() - 1)
		_, err = fmt.Fprint(buf, ",")
		if err != nil {
			return err
		}
	}
	if buf.Len() > 1 {
		buf.Truncate(buf.Len() - 1)
	}
	_, err = fmt.Fprint(buf, "}")
	if err != nil {
		return err
	}

	return dw.Stash(ctx, subject.ID(), buf.Bytes())
}
