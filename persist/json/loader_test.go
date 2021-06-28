package json

import (
	"context"
	"github.com/marstr/envelopes/persist/filesystem"
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

	fs := filesystem.FileSystem{
		Root: test_loc,
	}

	t1 := envelopes.Transaction{
		Comment: "1",
	}

	t2 := envelopes.Transaction{
		Comment: "2",
		Parents: []envelopes.ID{
			t1.ID(),
		},
	}

	t3 := envelopes.Transaction{
		Comment: "3",
		Parents: []envelopes.ID{
			t2.ID(),
		},
	}

	writer := Writer{
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

	subject := Loader{
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


func TestCache_Load_reuseHits(t *testing.T) {
	var err error
	ctx := context.Background()
	const rawTargetId = "07e72edcf913fd3ef5eababf60852216d68dbb90"
	var targetId envelopes.ID
	targetId.UnmarshalText([]byte(rawTargetId))

	passThrough := filesystem.FileSystem{
		Root: "../filesystem/testdata/test3/.baronial",
	}

	subject := persist.NewCache(10)
	subject.Loader = &Loader{
		Fetcher: passThrough,
		Loopback: subject,
	}

	var want envelopes.State
	err = subject.Load(ctx, targetId, &want)
	if err != nil {
		t.Error(err)
		return
	}

	var got envelopes.State
	err = subject.Load(ctx, targetId, &got)
	if err != nil {
		t.Error(err)
		return
	}

	if got.Budget != want.Budget {
		t.Logf("When encountering a cache hit, the SAME Budget object should be reused")
		t.Fail()
	}
}
