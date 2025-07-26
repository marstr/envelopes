package json_test

import (
	"context"
	"testing"

	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/persist/json"
)

func TestLoaderV3_LoadTransaction_notNullingReverts(t *testing.T) {
	ctx := context.Background()

	var err error
	mockFiles := NewMockFilesystem()
	var writer *json.Writer

	writer, err = json.NewWriterV3(mockFiles)
	if err != nil {
		t.Error(err)
		return
	}

	desired := envelopes.Transaction{
		Comment: "This transaction needs to not have the Reverts field",
	}

	err = writer.WriteTransaction(ctx, desired)
	if err != nil {
		t.Error(err)
		return
	}

	bogus := envelopes.Transaction{
		Comment: "Some nonsense",
	}

	var poisoned = envelopes.Transaction{
		Reverts: []envelopes.ID{bogus.ID()},
		Comment: "This transaction needs to have the Reverts field, and it needs to be not set to the default ID",
	}

	var specimen *json.LoaderV3
	specimen, err = json.NewLoaderV3(mockFiles)
	if err != nil {
		t.Error(err)
		return
	}

	err = specimen.LoadTransaction(ctx, desired.ID(), &poisoned)
	if err != nil {
		t.Errorf("it should've been able to load this: %v", err)
		return
	}

	if len(poisoned.Reverts) != 0 {
		t.Errorf("The poison value %q was discovered after a deemed successful load that should've cleared it.", poisoned.Reverts)
	}
}
