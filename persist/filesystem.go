package persist

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/marstr/envelopes"
)

const objectsDir = "objects"

type FileSystem struct {
	Root string
}

func (fs FileSystem) Fetch(id envelopes.ID) ([]byte, error) {
	return ioutil.ReadFile(fs.path(id))
}

func (fs FileSystem) write(target envelopes.IDer) (err error) {
	loc := fs.path(target.ID())
	os.MkdirAll(path.Dir(loc), os.ModePerm)
	handle, err := os.Create(loc)
	if err != nil {
		return
	}
	defer handle.Close()

	marshaled, err := json.Marshal(target)
	if err != nil {
		return
	}

	_, err = handle.Write(marshaled)
	return
}

func (fs FileSystem) WriteBudget(target envelopes.Budget) error {
	return fs.write(target)
}

func (fs FileSystem) WriteState(target envelopes.State) error {
	return fs.write(target)
}

func (fs FileSystem) WriteTransaction(target envelopes.Transaction) error {
	return fs.write(target)
}

func (fs FileSystem) path(id envelopes.ID) string {
	return filepath.Join(fs.Root, objectsDir, id.String()+".json")
}
