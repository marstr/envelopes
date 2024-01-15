package json

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/marstr/envelopes"
)

type (
	BankRecordIDV3 string

	Budget = BudgetV3
	// BudgetV3 is copy of envelopes.Budget for ORM purposes.
	BudgetV3 struct {
		Balance  BalanceV3               `json:"balance"`
		Children map[string]envelopes.ID `json:"children"`
	}

	Transaction = TransactionV3
	// TransactionV3 is a copy of envelopes.Transaction for ORM purposes.
	TransactionV3 struct {
		State       envelopes.ID   `json:"state"`
		PostedTime  time.Time      `json:"postedTime"`
		ActualTime  time.Time      `json:"actualTime,omitempty"`
		EnteredTime time.Time      `json:"enteredTime,omitempty"`
		Amount      BalanceV3      `json:"amount"`
		Merchant    string         `json:"merchant"`
		Comment     string         `json:"comment"`
		Committer   UserV3         `json:"committer,omitempty"`
		RecordId    BankRecordIDV3 `json:"recordId,omitempty"`
		Parent      []envelopes.ID `json:"parent"`
	}

	State = StateV3
	// StateV3 is a copy of envelopes.State for ORM purposes.
	StateV3 struct {
		Budget   envelopes.ID `json:"budget"`
		Accounts envelopes.ID `json:"accounts"`
	}

	User = UserV3
	// UserV3 is a copy of envelopes.User for ORM purposes.
	UserV3 struct {
		FullName string `json:"fullName"`
		Email    string `json:"email"`
	}

	Balance = BalanceV3
	// BalanceV3 is a copy of envelopes.Balance for ORM purposes.
	BalanceV3 map[envelopes.AssetType]*big.Rat

	Accounts = AccountsV3
	// AccountsV3 is a copy of envelopes.Accounts for ORM purposes.
	AccountsV3 map[string]BalanceV3
)

var jsonNumberPatternV3 = regexp.MustCompile(`^(?P<sign>-?)(?P<number>\d+)(?:\.(?P<fraction>\d+))?(?:[eE](?P<exponent>[\-+]?\d+))?$`)

// MarshalJSON converts a BalanceV3 into a serialized JSON object which can be round-tripped back to a BalanceV3.
func (b BalanceV3) MarshalJSON() ([]byte, error) {
	assetTypes := make([]string, 0, len(b))
	for k := range b {
		assetTypes = append(assetTypes, string(k))
	}
	sort.Strings(assetTypes)

	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)

	var err error
	_, err = fmt.Fprint(buf, "{")
	if err != nil {
		return nil, err
	}
	for i := range assetTypes {
		err = encoder.Encode(assetTypes[i])
		if err != nil {
			return nil, err
		}
		buf.Truncate(buf.Len() - 1)
		_, err = fmt.Fprint(buf, ":")
		if err != nil {
			return nil, err
		}
		err = encoder.Encode(formatRatV3(b[envelopes.AssetType(assetTypes[i])]))
		if err != nil {
			return nil, err
		}
		buf.Truncate(buf.Len() - 1)
		_, err = fmt.Fprint(buf, ",")
		if err != nil {
			return nil, err
		}
	}
	if buf.Len() > 1 {
		buf.Truncate(buf.Len() - 1)
	}
	_, err = fmt.Fprint(buf, "}")
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalJSON reconstitutes a BalanceV3 that had previously been converted into a JSON object by MarshalJSON.
func (b *BalanceV3) UnmarshalJSON(text []byte) error {
	type RawBalance map[envelopes.AssetType]json.Number

	unmarshaled := RawBalance{}

	dec := json.NewDecoder(bytes.NewReader(text))
	err := dec.Decode(&unmarshaled)
	if err != nil {
		return err
	}

	retval := make(BalanceV3, len(unmarshaled))
	*b = retval

	for k, v := range unmarshaled {
		var parsed *big.Rat
		parsed, err = parseRatV3(v)
		if err != nil {
			return err
		}
		retval[k] = parsed
	}
	return nil
}

func formatRatV3(rat *big.Rat) json.Number {
	return json.Number(rat.FloatString(3))
}

func parseRatV3(input json.Number) (*big.Rat, error) {
	var err error
	numerator := int64(0)
	denominator := int64(1)
	var sign int64 = 1

	match := jsonNumberPatternV3.FindStringSubmatch(string(input))
	if len(match) == 0 {
		return nil, fmt.Errorf("did not recognize %q as a JSON Number", input)
	}

	if match[1] == "-" {
		sign = -1
	}

	numerator, err = strconv.ParseInt(match[2], 10, 64)
	if err != nil {
		return nil, err
	}

	for _, c := range match[3] {
		numerator *= 10
		numerator += int64(c - '0')
		denominator *= 10
	}

	if match[4] != "" {
		var exponent int64
		if match[4][0] == '+' {
			match[4] = match[4][1:]
		}

		exponent, err = strconv.ParseInt(match[4], 10, 32)
		if err != nil {
			return nil, err
		}
		if exponent >= 0 {
			numerator = int64(float64(numerator) * math.Pow10(int(exponent)))
		} else {
			denominator = int64(float64(denominator) * math.Pow10(int(-exponent)))
		}
	}

	return big.NewRat(sign*numerator, denominator), nil
}
