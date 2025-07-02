package json_test

import (
	"context"
	"fmt"

	"github.com/marstr/collection/v2"
	"github.com/marstr/envelopes"
)

type MockFilesystem struct {
	*collection.LRUCache[envelopes.ID, []byte]
}

func NewMockFilesystem() *MockFilesystem {
	return NewMockFilesystemWithCapacity(10000) // Some arbitrary large number that feels like tests are unlikely to hit.
}

func NewMockFilesystemWithCapacity(cap uint) *MockFilesystem {
	return &MockFilesystem{
		LRUCache: collection.NewLRUCache[envelopes.ID, []byte](cap),
	}
}

func (mf MockFilesystem) Stash(ctx context.Context, id envelopes.ID, payload []byte) error {
	mf.Put(id, payload)
	return nil
}

func (mf MockFilesystem) Fetch(ctx context.Context, id envelopes.ID) ([]byte, error) {
	retval, ok := mf.Get(id)
	if !ok {
		return nil, fmt.Errorf("did not find a stashed objected with ID: %s", id)
	}
	return retval, nil
}
