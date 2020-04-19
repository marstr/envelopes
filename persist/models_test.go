package persist

import (
	"encoding/json"
	"math/big"
	"testing"
)

func TestBalance_MarshalJSON(t *testing.T) {
	testCases := []struct {
		subject Balance
		want    string
	}{
		{
			subject: Balance{
				"USD": big.NewRat(10056, 100),
				"EUR": big.NewRat(40098, 100),
			},
			want: `{"EUR":400.980,"USD":100.560}`,
		},
		{
			subject: Balance{
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

func Test_parseRat(t *testing.T) {
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
		got, err := parseRat(subject)

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
