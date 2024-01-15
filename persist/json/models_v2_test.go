package json

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/persist"
)

func TestBalance_MarshalJSON(t *testing.T) {
	testCases := []struct {
		subject BalanceV2
		want    string
	}{
		{
			subject: BalanceV2{
				"USD": big.NewRat(10056, 100),
				"EUR": big.NewRat(40098, 100),
			},
			want: `{"EUR":400.980,"USD":100.560}`,
		},
		{
			subject: BalanceV2{
				"FZROX": big.NewRat(589001, 1000),
			},
			want: `{"FZROX":589.001}`,
		},
	}

	for _, tc := range testCases {
		got, err := tc.subject.MarshalJSON()
		if err != nil {
			t.Error(err)
			continue
		}

		if string(got) != tc.want {
			t.Logf("\ngot:  %s\nwant: %s", string(got), tc.want)
			t.Fail()
		}
	}
}

func Test_parseRatV2(t *testing.T) {
	testCases := map[json.Number]*big.Rat{
		"1":      big.NewRat(1, 1),
		"1.0":    big.NewRat(1, 1),
		"0":      big.NewRat(0, 1),
		"0.0":    big.NewRat(0, 1),
		"197":    big.NewRat(197, 1),
		"197.89": big.NewRat(19789, 100),
		"-198":   big.NewRat(-198, 1),
		"4.2e3":  big.NewRat(4200, 1),
		"4.2e+3": big.NewRat(4200, 1),
		"4.2e-3": big.NewRat(42, 10000),
	}

	for subject, want := range testCases {
		got, err := parseRatV2(subject)

		if err != nil {
			t.Error(err)
			continue
		}

		if got.Cmp(want) != 0 {
			t.Logf("\ninput: %s\ngot:   %s\nwant:  %s\n", subject, got.FloatString(3), want.FloatString(3))
			t.Fail()
		}
	}
}

func Test_StateRoundtrip(t *testing.T) {
	ctx := context.Background()
	testCases := map[string]envelopes.State{
		"empty": {},
		"full": {
			Accounts: envelopes.Accounts{
				"checking": envelopes.Balance{"USD": big.NewRat(4590, 100)},
				"savings":  envelopes.Balance{"USD": big.NewRat(1, 100)},
			},
			Budget: &envelopes.Budget{
				Children: map[string]*envelopes.Budget{
					"foo": {
						Balance: envelopes.Balance{"USD": big.NewRat(2291, 100)},
					},
					"bar": {
						Balance: envelopes.Balance{"USD": big.NewRat(23, 1)},
					},
				},
			},
		},
		"accounts_only": {
			Accounts: envelopes.Accounts{
				"checking": envelopes.Balance{"USD": big.NewRat(4590, 100)},
				"savings":  envelopes.Balance{"USD": big.NewRat(1, 100)},
			},
		},
		"budget_only": {
			Budget: &envelopes.Budget{
				Children: map[string]*envelopes.Budget{
					"foo": {
						Balance: envelopes.Balance{"USD": big.NewRat(2291, 100)},
					},
					"bar": {
						Balance: envelopes.Balance{"USD": big.NewRat(2300, 100)},
					},
				},
			},
		},
	}

	tempLocation, err := os.MkdirTemp("", "envelopes_id_tests")
	if err != nil {
		t.Errorf("unable to create test location")
		return
	}
	defer os.RemoveAll(tempLocation)

	md := make(mockDisk)

	var saver *WriterV2
	saver, err = NewWriterV2(md)
	if err != nil {
		t.Error(err)
		return
	}
	var reader *LoaderV2
	reader, err = NewLoaderV2(md)
	if err != nil {
		t.Error(err)
		return
	}

	for name, subject := range testCases {
		want := subject.ID()

		err := saver.Write(ctx, subject)
		if err != nil {
			t.Errorf("(%s) unable to write subject: %v", name, err)
			continue
		}

		var rehydrated envelopes.State
		err = reader.Load(ctx, want, &rehydrated)
		if err != nil {
			t.Errorf("(%s) unable to read subject: %v", name, err)
			continue
		}

		if got := rehydrated.ID(); !got.Equal(want) {
			t.Logf("\ngot: \t%s\nwant:\t%s", got, want)
			t.Fail()
		}
	}
}

type mockDisk map[envelopes.ID][]byte

func (md mockDisk) Stash(_ context.Context, id envelopes.ID, payload []byte) error {
	md[id] = payload
	return nil
}

func (md mockDisk) Fetch(_ context.Context, id envelopes.ID) ([]byte, error) {
	if val, ok := md[id]; ok {
		return val, nil
	}
	return []byte{}, persist.ErrObjectNotFound(id)
}
