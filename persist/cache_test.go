package persist

import (
	"context"
	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/persist/filesystem"
	"testing"
)

func TestCache_Load_passThroughMiss(t *testing.T) {
	ctx := context.Background()
	const rawTargetId = "07e72edcf913fd3ef5eababf60852216d68dbb90"
	var targetId envelopes.ID
	targetId.UnmarshalText([]byte(rawTargetId))

	passThrough := filesystem.FileSystem{
		Root: "./testdata/test3/.baronial",
	}

	subject := NewCache(10)
	subject.Loader = &DefaultLoader{
		Fetcher: passThrough,
		Loopback: subject,
	}

	var got envelopes.State
	err := subject.Load(ctx, targetId, &got)
	if err != nil {
		t.Error(err)
		return
	}

	var want envelopes.State
	err = subject.Loader.Load(ctx, targetId, &want)
	if err != nil {
		t.Error(err)
		return
	}

	if !got.Equal(want) {
		t.Logf("got: %s, want: %s", got.String(), want.String())
		t.Fail()
	}
}

func TestCache_Load_reuseHits(t *testing.T) {
	var err error
	ctx := context.Background()
	const rawTargetId = "07e72edcf913fd3ef5eababf60852216d68dbb90"
	var targetId envelopes.ID
	targetId.UnmarshalText([]byte(rawTargetId))

	passThrough := filesystem.FileSystem{
		Root: "./testdata/test3/.baronial",
	}

	subject := NewCache(10)
	subject.Loader = &DefaultLoader{
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
