package filesystem_test

import (
	"context"
	"encoding/json"
	"math/big"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/persist"
	"github.com/marstr/envelopes/persist/filesystem"
)

func TestOpenRepositoryLayout1(t *testing.T) {
	var ctx context.Context

	if deadline, ok := t.Deadline(); ok {
		const deleteFilesTime = -3 * time.Second
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(context.Background(), deadline.Add(deleteFilesTime))
		defer cancel()
	} else {
		ctx = context.Background()
	}

	repo, err := filesystem.OpenRepository(ctx, "./testdata/test5/.baronial")
	if err != nil {
		t.Error(err)
	}

	encounteredLayout := repo.FileSystem.ObjectLayout
	expectedLayout := uint(1)

	if encounteredLayout != expectedLayout {
		t.Errorf("wrong layout\n\tgot: %v\n\twant: %v", encounteredLayout, expectedLayout)
	}

	current, err := repo.Current(ctx)
	if err != nil {
		t.Error(err)
	}

	headId, err := persist.Resolve(ctx, repo, current)
	if err != nil {
		t.Error(err)
	}

	var head envelopes.Transaction
	err = repo.LoadTransaction(ctx, headId, &head)
	if err != nil {
		t.Error(err)
	}

	encounteredStateId := head.State.ID().String()
	expectedStateId := "960a403e64cca0c8022c8d72e96905991b74f533"
	if encounteredStateId != expectedStateId {
		t.Errorf("wrong transaction:\n\texpected state: %q\n\tgot state %q", expectedStateId, encounteredStateId)
	}
}

func TestCreateRepositoryLayout1(t *testing.T) {
	var ctx context.Context

	if deadline, ok := t.Deadline(); ok {
		const deleteFilesTime = -3 * time.Second
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(context.Background(), deadline.Add(deleteFilesTime))
		defer cancel()
	} else {
		ctx = context.Background()
	}

	testDir, err := os.MkdirTemp("", "envelopes")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(testDir)

	repo, err := filesystem.OpenRepository(ctx, testDir, filesystem.RepositoryObjectLoc(1))
	if err != nil {
		t.Error(err)
	}

	exampleTransaction := envelopes.Transaction{
		State: &envelopes.State{
			Budget: &envelopes.Budget{
				Balance: envelopes.Balance{"": big.NewRat(314, 100)},
			},
		},
	}

	err = repo.WriteTransaction(ctx, exampleTransaction)
	if err != nil {
		t.Error(err)
	}

	id := exampleTransaction.ID().String()

	handle, err := os.Open(filepath.Join(testDir, filesystem.ObjectsDir, id[:2], id[2:]+".json"))
	if err != nil {
		t.Error(err)
	}
	defer handle.Close()
}

func TestRemoteConfig_UnmarshalJSON(t *testing.T) {
	raw := []byte("{\"url\":\"https://go.dev/play\"}")

	var hydrated filesystem.RemoteConfig
	err := json.Unmarshal(raw, &hydrated)
	if err != nil {
		t.Error(err)
		return
	}

	if got := hydrated.Url.Scheme; got != "https" {
		t.Errorf("want: \"https\" got: %q", hydrated.Url.Scheme)
	}

	if got := hydrated.Url.Hostname(); got != "go.dev" {
		t.Errorf("want: \"go.dev\" got: %q", got)
	}

	if got := hydrated.Url.Path; got != "/play" {
		t.Errorf("want: \"/play\" got: %q", got)
	}
}

func TestRemoteConfig_UnmarshalJSON_IgnoreUnknownProperties(t *testing.T) {
	raw := []byte("{\"foo\":\"bar\",\"url\":\"https://marstr.dev\"}")
	var hydrated filesystem.RemoteConfig
	err := json.Unmarshal(raw, &hydrated)
	if err != nil {
		t.Error(err)
		return
	}

	if got := hydrated.Url.Scheme; got != "https" {
		t.Errorf("want: \"https\" got: %q", got)
	}

	if got := hydrated.Url.Hostname(); got != "marstr.dev" {
		t.Errorf("want: \"marstr.dev\" got: %q", got)
	}
}

func TestRemoteConfig_MarshalingRoundTripFromUnmarshaled(t *testing.T) {
	remote, err := url.Parse("//scotty.iot.strohomish:9043")
	if err != nil {
		t.Error(err)
		return
	}
	original := filesystem.RemoteConfig{
		Url: remote,
	}

	marshaled, err := json.Marshal(original)
	if err != nil {
		t.Error(err)
		return
	}

	var unmarshaled filesystem.RemoteConfig
	err = json.Unmarshal(marshaled, &unmarshaled)
	if err != nil {
		t.Error(err)
		return
	}

	if got, want := unmarshaled.Url.String(), remote.String(); got != want {
		t.Errorf("got: %q want: %q", got, want)
	}
}

func TestRemoteConfig_MarshalingRoundTripFromMarshaled(t *testing.T) {
	original := `{"url":"//scotty.iot.strohomish:9043"}`

	var unmarshaled filesystem.RemoteConfig
	err := json.Unmarshal([]byte(original), &unmarshaled)
	if err != nil {
		t.Error(err)
		return
	}

	marshaled, err := json.Marshal(unmarshaled)
	if err != nil {
		t.Error(err)
		return
	}

	if got := string(marshaled); got != original {
		t.Errorf("want: %q got: %q", original, got)
	}
}
