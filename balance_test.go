package envelopes_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/marstr/envelopes"
)

func ExampleParseBalance() {
	fmt.Println(envelopes.ParseBalance([]byte("USD 99.88")))
	// Output:
	// USD 99.880 <nil>
}

func ExampleBalance_String() {
	fmt.Println(envelopes.Balance{"USD": big.NewRat(9999, 100)})
	// Output:
	// USD 99.990
}

func ExampleBalance_Equal() {
	a := envelopes.Balance{"USD": big.NewRat(0, 100)}
	b := envelopes.Balance{"USD": big.NewRat(0, 1000)}
	c := envelopes.Balance{}
	d := envelopes.Balance{"$": big.NewRat(0, 100)}
	e := envelopes.Balance{"EUR": big.NewRat(1095, 100)}
	f := envelopes.Balance{"USD": big.NewRat(1095, 100)}
	g := envelopes.Balance{"$": big.NewRat(1095, 100)}

	fmt.Println(a.Equal(b)) // True
	fmt.Println(a.Equal(c)) // True
	fmt.Println(a.Equal(d)) // True
	fmt.Println(a.Equal(e)) // False
	fmt.Println(a.Equal(f)) // False
	fmt.Println(a.Equal(g)) // False
	fmt.Println(b.Equal(c)) // True
	fmt.Println(b.Equal(d)) // True
	fmt.Println(b.Equal(e)) // False
	fmt.Println(b.Equal(f)) // False
	fmt.Println(b.Equal(g)) // False
	fmt.Println(c.Equal(d)) // True
	fmt.Println(c.Equal(e)) // False
	fmt.Println(d.Equal(e)) // False
	fmt.Println(d.Equal(f)) // False
	fmt.Println(d.Equal(g)) // False
	fmt.Println(e.Equal(f)) // False
	fmt.Println(e.Equal(g)) // False
	fmt.Println(f.Equal(g)) // False

	// Output:
	// true
	// true
	// true
	// false
	// false
	// false
	// true
	// true
	// false
	// false
	// false
	// true
	// false
	// false
	// false
	// false
	// false
	// false
	// false
}

func ExampleBalance_Scale() {
	subject := envelopes.Balance{"USD": big.NewRat(100, 1)}
	subject = subject.Scale(.5)
	fmt.Println(subject)
	// Output:
	// USD 50.000
}

func TestBalance_String(t *testing.T) {
	zero := big.NewRat(0, 1)
	testCases := []struct {
		envelopes.Balance
		expected string
	}{
		{envelopes.Balance{"USD": zero}, "USD 0.000"},
		{envelopes.Balance{"USD": big.NewRat(1, 100)}, "USD 0.010"},
		{envelopes.Balance{"USD": big.NewRat(-1, 100)}, "USD -0.010"},
		{envelopes.Balance{"USD": big.NewRat(1000, 1)}, "USD 1000.000"},
		{
			envelopes.Balance{
				"FZROX": zero,
				"MSFT": zero,
				"TMUS": zero,
				"USD": zero,
			},
			"USD 0.00",
		},
		{envelopes.Balance{}, "USD 0.00"},
		{envelopes.Balance{"MSFT": zero	}, "MSFT 0.000"},
	}

	for _, tc := range testCases {
		if got := tc.Balance.String(); got != tc.expected {
			t.Logf("\ngot:  %q\nwant: %q", got, tc.expected)
			t.Fail()
		}
	}
}

func Test_ParseBalance(t *testing.T) {
	t.Run("scalar", testParseBalance_scalar)
	t.Run("negative scalar", testParseBalance_scalarNegative)
	t.Run("complex", testParseBalance_complex)
}

func testParseBalance_scalarNegative(t *testing.T) {
	testCases := []string{
		"",
		"$1,00,000.00",
	}

	for _, tc := range testCases {
		got, err := envelopes.ParseBalance([]byte(tc))
		if err == nil {
			t.Logf("malformed %q was able to be parsed into: %v", tc, got)
			t.Fail()
		} else if got != nil {
			t.Logf("result should always be nil when an error is present")
			t.Fail()
		}
	}
}

func testParseBalance_scalar(t *testing.T) {
	testCases := []struct {
		string
		expected envelopes.Balance
	}{
		{"USD 0.00", envelopes.Balance{"USD": big.NewRat(0, 1)}},
		{"0", envelopes.Balance{"USD": big.NewRat(0, 1)}},
		{"USD 0.01", envelopes.Balance{"USD": big.NewRat(1, 100)}},
		{"0.01", envelopes.Balance{"USD": big.NewRat(1, 100)}},
		{".01", envelopes.Balance{"USD": big.NewRat(1, 100)}},
		{"USD -0.01", envelopes.Balance{"USD": big.NewRat(-1, 100)}},
		{"-0.01", envelopes.Balance{"USD": big.NewRat(-1, 100)}},
		{"USD 1", envelopes.Balance{"USD": big.NewRat(1, 1)}},
		{"1", envelopes.Balance{"USD": big.NewRat(1, 1)}},
		{"USD -1", envelopes.Balance{"USD": big.NewRat(-1, 1)}},
		{"USD 0.001", envelopes.Balance{"USD": big.NewRat(1, 1000)}},
		{"USD 0.005", envelopes.Balance{"USD": big.NewRat(5, 1000)}},
		{"USD 1000000", envelopes.Balance{"USD": big.NewRat(1000000, 1)}},
		{"USD 1,000,000", envelopes.Balance{"USD": big.NewRat(1000000, 1)}},
		{"USD 1,000,000.00", envelopes.Balance{"USD": big.NewRat(1000000, 1)}},
		{"1,000", envelopes.Balance{"USD": big.NewRat(1000, 1)}},
		{"988.01", envelopes.Balance{"USD": big.NewRat(98801, 100)}},
		{"988", envelopes.Balance{"USD": big.NewRat(988, 1)}},
		{"USD 5", envelopes.Balance{"USD": big.NewRat(5, 1)}},
		{"USD -5", envelopes.Balance{"USD": big.NewRat(-5, 1)}},
		{"USD 0800", envelopes.Balance{"USD": big.NewRat(800, 1)}},
		{" USD 10.98\n", envelopes.Balance{"USD": big.NewRat(1098, 100)}},
		{"USD 10.98\n", envelopes.Balance{"USD": big.NewRat(1098, 100)}},
		{"FZROX 590.319", envelopes.Balance{"FZROX": big.NewRat(590319, 1000)}},
	}

	for _, tc := range testCases {
		if got, err := envelopes.ParseBalance([]byte(tc.string)); err != nil {
			t.Errorf("input: %q -> %v", tc.string, err)
			continue
		} else if !got.Equal(tc.expected) {
			t.Logf("\ninput: %q\ngot:   %s\nwant:  %s", tc.string, got, tc.expected)
			t.Fail()
		}
	}
}

func testParseBalance_complex(t *testing.T) {
	testCases := []struct {
		string
		expected envelopes.Balance
	}{
		{
			"USD 10.00: FZROX 193.468",
			envelopes.Balance{
				"USD": big.NewRat(10, 1),
				"FZROX": big.NewRat(193468, 1000),
			},
		},
		{
			"USD10.00: FZROX193.468",
			envelopes.Balance{
				"USD": big.NewRat(10, 1),
				"FZROX": big.NewRat(193468, 1000),
			},
		},
		{
			"USD10.00:FZROX193.468",
			envelopes.Balance{
				"USD": big.NewRat(10, 1),
				"FZROX": big.NewRat(193468, 1000),
			},
		},
		{
			"USD10.00:FZROX193.468:MSFT198.890",
			envelopes.Balance{
				"USD": big.NewRat(10, 1),
				"FZROX": big.NewRat(193468, 1000),
				"MSFT": big.NewRat(19889, 100),
			},
		},
		{
			"USD10.00\nFZROX193.468\nMSFT198.890",
			envelopes.Balance{
				"USD": big.NewRat(10, 1),
				"FZROX": big.NewRat(193468, 1000),
				"MSFT": big.NewRat(19889, 100),
			},
		},
		{
			"USD10:USD5.6",
			envelopes.Balance{
				"USD": big.NewRat(156, 10),
			},
		},
	}

	for _, tc := range testCases {
		if got, err := envelopes.ParseBalance([]byte(tc.string)); err != nil {
			t.Errorf("input: %q -> %v", tc.string, err)
			continue
		} else if !got.Equal(tc.expected) {
			t.Logf("\ninput: %q\ngot:   %s\nwant:  %s", tc.string, got, tc.expected)
			t.Fail()
		}
	}
}
