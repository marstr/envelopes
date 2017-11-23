package envelopes_test

import (
	"fmt"
	"testing"

	"github.com/marstr/envelopes"
)

func ExampleParseAmount() {
	fmt.Println(envelopes.ParseAmount("$99.88"))
	// Output:
	// 9988 <nil>
}

func ExampleFormatAmount() {
	fmt.Println(envelopes.FormatAmount(9999))
	// Output:
	// $99.99
}

func TestFormatAmount(t *testing.T) {
	testCases := []struct {
		int64
		expected string
	}{
		{0, "$0.00"},
		{1, "$0.01"},
		{-1, "$-0.01"},
		{100000, "$1000.00"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprint(tc.int64), func(t *testing.T) {
			if got := envelopes.FormatAmount(tc.int64); got != tc.expected {
				t.Logf("got:\n\t%qwant:\n\t%q", got, tc.expected)
				t.Fail()
			}
		})
	}
}

func Test_ParseAmount(t *testing.T) {
	testCases := []struct {
		string
		expected int64
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
	}

	for _, tc := range testCases {
		t.Run(tc.string, func(t *testing.T) {
			if got, err := envelopes.ParseAmount(tc.string); err != nil {
				t.Error(err)
				return
			} else if got != tc.expected {
				t.Logf("got: %d want: %d", got, tc.expected)
				t.Fail()
			}

		})
	}
}
