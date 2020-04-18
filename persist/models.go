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
		State       envelopes.ID
		PostedTime  time.Time
		ActualTime  time.Time
		EnteredTime time.Time
		Amount      envelopes.Balance
		Merchant    string
		Comment     string
		Committer   User
		Parent      envelopes.ID
	}

	// State is a copy of envelopes.State for ORM purposes.
	State struct {
		Budget   envelopes.ID
		Accounts envelopes.ID
	}

	// User is a copy of envelopes.User for ORM purposes.
	User struct {
		FullName string
		Email    string
	}
)
