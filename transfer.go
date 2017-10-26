package envelopes

import (
	"time"
)

// Transfer contains metadata about a change in the state of your budgets.
// It is differentiated from a Transaction because it may happen internally
// for bookkeeping reasons, with no Merchant or Account being targeted.
type Transfer struct {
	time.Time
	Comments string
	Effect
}
