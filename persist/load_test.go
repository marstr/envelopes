package persist_test

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/persist"
)

func TestLoadAncestor(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// ctx, cancel := context.Background(), context.CancelFunc(func() {})
	defer cancel()

	test_loc, err := ioutil.TempDir("", "envelopes_TestLoadAncestor_")
	if err != nil {
		t.Error(err)
		return
	}
	defer os.RemoveAll(test_loc)

	fs := persist.FileSystem{
		Root: test_loc,
	}

	t1 := envelopes.Transaction{
		Comment: "1",
	}

	t2 := envelopes.Transaction{
		Comment: "2",
		Parent: []envelopes.ID{
			t1.ID(),
		},
	}

	t3 := envelopes.Transaction{
		Comment: "3",
		Parent: []envelopes.ID{
			t2.ID(),
		},
	}

	writer := persist.DefaultWriter{
		Stasher: fs,
	}

	toWrite := []*envelopes.Transaction{&t1, &t2, &t3}
	for _, entry := range toWrite {
		err = writer.Write(ctx, entry)
		if err != nil {
			t.Error(err)
			return
		}
	}

	subject := persist.DefaultLoader{
		Fetcher: fs,
	}

	testCases := []struct {
		original *envelopes.Transaction
		jumps    uint
		expected *envelopes.Transaction
	}{
		{&t1, 0, &t1},
		{&t3, 0, &t3},
		{&t3, 1, &t2},
		{&t2, 1, &t1},
		{&t3, 2, &t1},
	}

	for _, tc := range testCases {
		result, err := persist.LoadAncestor(ctx, subject, tc.original.ID(), tc.jumps)
		if err != nil {
			t.Errorf("Original Comment: %q jumps: %d\n\tError: %v", tc.original.Comment, tc.jumps, err)
			continue
		}

		if result.Comment != tc.expected.Comment {
			t.Logf(
				"Original Comment: %q jumps: %d\n\tgot:  %q\nwant: %q",
				tc.original.Comment,
				tc.jumps,
				result.Comment,
				tc.original.Comment)
			t.Fail()
		}
	}
}
