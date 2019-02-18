package envelopes_test

import (
	"fmt"
	"testing"

	"github.com/marstr/envelopes"
)

func ExampleParseBalance() {
	fmt.Println(envelopes.ParseBalance("$99.88"))
	// Output:
	// USD 99.88 <nil>
}

func ExampleBalance_String() {
	fmt.Println(envelopes.Balance(9999))
	// Output:
	// USD 99.99
}

func TestBalance_String(t *testing.T) {
	testCases := []struct {
		envelopes.Balance
		expected string
	}{
		{0, "USD 0.00"},
		{1, "USD 0.01"},
		{-1, "USD -0.01"},
		{100000, "USD 1000.00"},
	}

	for _, tc := range testCases {
		if got := tc.Balance.String(); got != tc.expected {
			t.Logf("got:\n\t%qwant:\n\t%q", got, tc.expected)
			t.Fail()
		}
	}
}

func Test_ParseBalance(t *testing.T) {
	testCases := []struct {
		string
		expected envelopes.Balance
	}{
		{"$0.00", 0},
		{"0", 0},
		{"$0.01", 1},
		{"0.01", 1},
		{".01", 1},
		{"$-0.01", -1},
		{"-0.01", -1},
		{"$1", 100},
		{"1", 100},
		{"$-1", -100},
		{"$0.001", 0},
		{"$0.005", 1},
		{"$1000000", 100000000},
		{"$1,000,000", 100000000},
		{"$5", 500},
		{"$-5", -500},
		{"$0800", 80000},
		{" $10.98\n", 1098},
	}

	for _, tc := range testCases {
		if got, err := envelopes.ParseBalance(tc.string); err != nil {
			t.Error(err)
			continue
		} else if got != tc.expected {
			t.Logf("got: %d want: %d", got, tc.expected)
			t.Fail()
		}
	}
}
