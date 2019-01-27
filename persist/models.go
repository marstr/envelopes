package persist

import (
	"github.com/marstr/envelopes"
	"time"
)

type (
	// Budget is copy of envelopes.Budget for ORM purposes.
	Budget struct {
		Balance  envelopes.Balance
		Children map[string]envelopes.ID
	}

	// Transaction is a copy of envelopes.Transaction for ORM purposes.
	Transaction struct {
		State    envelopes.ID
		Time     time.Time
		Amount   envelopes.Balance
		Merchant string
		Comment  string
		Parent   envelopes.ID
	}

	// State is a copy of envelopes.State for ORM purposes.
	State struct {
		Budget   envelopes.ID
		Accounts envelopes.ID
	}
)
