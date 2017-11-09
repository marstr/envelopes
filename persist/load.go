package persist

import (
	"encoding/json"
	"io/ioutil"

	"github.com/marstr/envelopes"
)

// Loader can instantiate core envelopes objects given just an ID.
type Loader interface {
	LoadBudget(id envelopes.ID) (envelopes.Budget, error)
	LoadState(id envelopes.ID) (envelopes.State, error)
	LoadTransaction(id envelopes.ID) (envelopes.Transaction, error)
}

// DefaultLoader wraps a Fetcher and does just the unmarshaling portion.
type DefaultLoader struct {
	Fetcher
}

func (dl DefaultLoader) load(id envelopes.ID, target interface{}) (err error) {
	reader, err := dl.Fetch(id)
	if err != nil {
		return
	}

	contents, err := ioutil.ReadAll(reader)
	if err != nil {
		return
	}

	err = json.Unmarshal(contents, target)
	return
}

// LoadBudget fetches a Budget in its marshaled form, then unmarshals it into a Budget object.
func (dl DefaultLoader) LoadBudget(id envelopes.ID) (loaded envelopes.Budget, err error) {
	err = dl.load(id, &loaded)
	return
}

// LoadState fetches a State in its marshaled form, then unmarshals it into a State object.
func (dl DefaultLoader) LoadState(id envelopes.ID) (loaded envelopes.State, err error) {
	err = dl.load(id, &loaded)
	return
}

// LoadTransaction fetches a Transaction in its marshaled form, then unmarshals it into a Transaction object.
func (dl DefaultLoader) LoadTransaction(id envelopes.ID) (loaded envelopes.Transaction, err error) {
	err = dl.load(id, &loaded)
	return
}
