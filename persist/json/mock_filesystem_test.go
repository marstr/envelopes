package json_test

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/marstr/collection/v2"
	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/persist"
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

func (mf MockFilesystem) StashReadCloser(ctx context.Context, id envelopes.ID, payload io.ReadCloser) error {
	buffer, err := io.ReadAll(payload)
	if err != nil {
		return err
	}

	err = payload.Close()
	if err != nil {
		return err
	}

	mf.Put(id, buffer)
	return nil
}

func (mf MockFilesystem) Fetch(ctx context.Context, id envelopes.ID) ([]byte, error) {
	retval, ok := mf.Get(id)
	if !ok {
		return nil, fmt.Errorf("did not find a stashed objected with ID: %s", id)
	}
	return retval, nil
}

func (mf MockFilesystem) FetchReadCloser(ctx context.Context, id envelopes.ID) (io.ReadCloser, error) {
	retval, ok := mf.Get(id)
	if !ok {
		return nil, persist.ErrObjectNotFound(id)
	}
	return io.NopCloser(bytes.NewReader(retval)), nil
}
