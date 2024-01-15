package json

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/marstr/envelopes"
)

func TestWriterV3_writeAccounts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	exampleAccounts := envelopes.Accounts{
		"savings":  envelopes.Balance{"MSFT": big.NewRat(144012, 1000)},
		"checking": envelopes.Balance{"USD": big.NewRat(12313, 100)},
	}

	mockStore := make(mockDisk)
	subject := WriterV3{Stasher: mockStore}
	err := subject.Write(ctx, exampleAccounts)
	if err != nil {
		t.Error(err)
	}

	got := string(mockStore[exampleAccounts.ID()])
	want := `{"checking":{"USD":123.130},"savings":{"MSFT":144.012}}`
	if got != want {
		t.Errorf("\ngot:  %q\nwant: %q", got, want)
	}
}
