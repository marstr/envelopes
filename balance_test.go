package envelopes_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/marstr/envelopes"
)

func ExampleParseBalance() {
	fmt.Println(envelopes.ParseBalance([]byte("$99.88")))
	// Output:
	// USD 99.88 <nil>
}

func ExampleBalance_String() {
	fmt.Println(envelopes.Balance{"USD": big.NewRat(9999, 100)})
	// Output:
	// USD 99.99
}

func ExampleBalance_Normalize() {

}

func ExampleBalance_Scale() {
	subject := envelopes.Balance{"USD": big.NewRat(100, 1)}
	subject = subject.Scale(.5)
	fmt.Println(subject)
	// Output:
	// USD 50.00
}

func TestBalance_String(t *testing.T) {
	testCases := []struct {
		envelopes.Balance
		expected string
	}{
		{envelopes.Balance{"USD": big.NewRat(0, 100)}, "USD 0.00"},
		{envelopes.Balance{"USD": big.NewRat(1, 100)}, "USD 0.01"},
		{envelopes.Balance{"USD": big.NewRat(-1, 100)}, "USD -0.01"},
		{envelopes.Balance{"USD": big.NewRat(1000, 1)}, "USD 1000.00"},
	}

	for _, tc := range testCases {
		if got := tc.Balance.String(); got != tc.expected {
			t.Logf("got:\n\t%qwant:\n\t%q", got, tc.expected)
			t.Fail()
		}
	}
}

func Test_ParseBalance(t *testing.T) {
	t.Run("scalar", testParseBalance_scalar)
	t.Run("negative scalar", testParseBalance_scalarNegative)
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
		{"$0.00", envelopes.Balance{"USD": big.NewRat(0, 1)}},
		{"0", envelopes.Balance{"USD": big.NewRat(0, 1)}},
		{"$0.01", envelopes.Balance{"USD": big.NewRat(1, 100)}},
		{"0.01", envelopes.Balance{"USD": big.NewRat(1, 100)}},
		{".01", envelopes.Balance{"USD": big.NewRat(1, 100)}},
		{"$-0.01", envelopes.Balance{"USD": big.NewRat(-1, 100)}},
		{"-0.01", envelopes.Balance{"USD": big.NewRat(-1, 100)}},
		{"$1", envelopes.Balance{"USD": big.NewRat(1, 1)}},
		{"1", envelopes.Balance{"USD": big.NewRat(1, 1)}},
		{"$-1", envelopes.Balance{"USD": big.NewRat(-1, 1)}},
		{"$0.001", envelopes.Balance{"USD": big.NewRat(1, 1000)}},
		{"$0.005", envelopes.Balance{"USD": big.NewRat(5, 1000)}},
		{"$1000000", envelopes.Balance{"USD": big.NewRat(1000000, 1)}},
		{"$1,000,000", envelopes.Balance{"USD": big.NewRat(1000000, 1)}},
		{"$1,000,000.00", envelopes.Balance{"USD": big.NewRat(1000000, 1)}},
		{"1,000", envelopes.Balance{"USD": big.NewRat(1000, 1)}},
		{"988.01", envelopes.Balance{"USD": big.NewRat(98801, 100)}},
		{"988", envelopes.Balance{"USD": big.NewRat(988, 1)}},
		{"$5", envelopes.Balance{"USD": big.NewRat(5, 1)}},
		{"$-5", envelopes.Balance{"USD": big.NewRat(-5, 1)}},
		{"$0800", envelopes.Balance{"USD": big.NewRat(800, 1)}},
		{" $10.98\n", envelopes.Balance{"USD": big.NewRat(1098, 100)}},
		{"$10.98\n", envelopes.Balance{"USD": big.NewRat(1098, 100)}},
	}

	for _, tc := range testCases {
		if got, err := envelopes.ParseBalance([]byte(tc.string)); err != nil {
			t.Errorf("input: %q -> %v", tc.string, err)
			continue
		} else if !got.Equal(tc.expected) {
			t.Logf("input: %q got: %s want: %s", tc.string, got, tc.expected)
			t.Fail()
		}
	}
}
