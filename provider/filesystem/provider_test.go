package filesystem_test

import (
	"path"
	"runtime"
	"testing"

	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/provider/filesystem"
)

func TestProvider_LoadState(t *testing.T) {
	_, currentFile, _, _ := runtime.Caller(0)

	subject := filesystem.Provider{
		RootLocation: path.Join(path.Dir(currentFile), "testdata", "store1"),
	}

	_, err := subject.LoadState(envelopes.State{}.ID())
	if err != nil {
		t.Error(err)
	}
}
