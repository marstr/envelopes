package json_test

import (
	"context"
	stdjson "encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/persist"
	"github.com/marstr/envelopes/persist/json"
)

func newVersionedLoader(t *testing.T, version uint, fetcher persist.Fetcher) persist.Loader {
	t.Helper()
	var loader persist.Loader
	var err error
	switch version {
	case 1:
		loader, err = json.NewLoaderV1(fetcher)
	case 2:
		loader, err = json.NewLoaderV2(fetcher)
	case 3:
		loader, err = json.NewLoaderV3(fetcher)
	default:
		t.Fatalf("unsupported test version %d", version)
	}
	if err != nil {
		t.Fatal(err)
	}
	return loader
}

func TestLoader_DecodingErrorContext(t *testing.T) {
	ctx := context.Background()
	for version := uint(1); version <= 3; version++ {
		files := NewMockFilesystem()
		loader := newVersionedLoader(t, version, files)
		id := envelopes.ID{1}
		type loadCase struct {
			objectType string
			load       func(context.Context, envelopes.ID) error
		}
		methods := []loadCase{
			{"transaction", func(ctx context.Context, id envelopes.ID) error {
				return loader.LoadTransaction(ctx, id, &envelopes.Transaction{})
			}},
			{"state", func(ctx context.Context, id envelopes.ID) error {
				return loader.LoadState(ctx, id, &envelopes.State{})
			}},
			{"budget", func(ctx context.Context, id envelopes.ID) error {
				return loader.LoadBudget(ctx, id, &envelopes.Budget{})
			}},
			{"accounts", func(ctx context.Context, id envelopes.ID) error {
				return loader.LoadAccounts(ctx, id, &envelopes.Accounts{})
			}},
		}
		if version == 3 {
			methods = append(methods, loadCase{"attachment", func(ctx context.Context, id envelopes.ID) error {
				return loader.LoadAttachment(ctx, id, &envelopes.Attachment{})
			}})
		}
		for _, method := range methods {
			for _, payload := range []string{`{"private-payload-sentinel":`, `42`} {
				t.Run(fmt.Sprintf("v%d/%s/%s", version, method.objectType, payload), func(t *testing.T) {
					if err := files.Stash(ctx, id, []byte(payload)); err != nil {
						t.Fatal(err)
					}
					err := method.load(ctx, id)
					if err == nil {
						t.Fatal("expected decode failure")
					}
					wantContext := fmt.Sprintf("decode %s object %s (JSONV%d)", method.objectType, id, version)
					if !strings.Contains(err.Error(), wantContext) {
						t.Fatalf("missing object context: %v", err)
					}
					if payload == `42` {
						var typeErr *stdjson.UnmarshalTypeError
						if !errors.As(err, &typeErr) {
							t.Fatalf("type error was not preserved: %v", err)
						}
					} else {
						var syntaxErr *stdjson.SyntaxError
						if !errors.As(err, &syntaxErr) {
							t.Fatalf("syntax error was not preserved: %v", err)
						}
					}
					if strings.Contains(err.Error(), "private-payload-sentinel") {
						t.Fatal("error context included the object payload")
					}
				})
			}
		}
	}
}

type failingFetcher struct {
	persist.Fetcher
	err error
}

func (fetcher failingFetcher) Fetch(context.Context, envelopes.ID) ([]byte, error) {
	return nil, fetcher.err
}

func TestLoader_FetchErrorsUnchanged(t *testing.T) {
	sentinel := errors.New("fetch failed")
	for version := uint(1); version <= 3; version++ {
		loader := newVersionedLoader(t, version, failingFetcher{err: sentinel})
		err := loader.LoadTransaction(context.Background(), envelopes.ID{1}, &envelopes.Transaction{})
		if err != sentinel {
			t.Fatalf("v%d fetch error was wrapped: %v", version, err)
		}
	}
}

func TestLoader_NestedDecodingErrorContext(t *testing.T) {
	ctx := context.Background()
	files := NewMockFilesystem()
	transactionID, stateID := envelopes.ID{1}, envelopes.ID{2}
	payload := fmt.Sprintf(`{"state":"%s"}`, stateID)
	if err := files.Stash(ctx, transactionID, []byte(payload)); err != nil {
		t.Fatal(err)
	}
	if err := files.Stash(ctx, stateID, []byte("{")); err != nil {
		t.Fatal(err)
	}
	loader := newVersionedLoader(t, 3, files)
	err := loader.LoadTransaction(ctx, transactionID, &envelopes.Transaction{})
	if err == nil || !strings.Contains(err.Error(), "decode state object "+stateID.String()+" (JSONV3)") {
		t.Fatalf("missing failing sub-object context: %v", err)
	}
	var syntaxErr *stdjson.SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("nested decode cause was not preserved: %v", err)
	}
}

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
