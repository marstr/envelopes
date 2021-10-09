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
	BankRecordIDV1 string

	// BudgetV1 is copy of envelopes.Budget for ORM purposes.
	BudgetV1 struct {
		Balance  BalanceV1               `json:"balance"`
		Children map[string]envelopes.ID `json:"children"`
	}

	// TransactionV1 is a copy of envelopes.Transaction for ORM purposes.
	TransactionV1 struct {
		State       envelopes.ID   `json:"state"`
		PostedTime  time.Time      `json:"postedTime"`
		ActualTime  time.Time      `json:"actualTime,omitempty"`
		EnteredTime time.Time `json:"enteredTime,omitempty"`
		Amount      BalanceV1 `json:"amount"`
		Merchant    string    `json:"merchant"`
		Comment   string       `json:"comment"`
		Committer UserV1         `json:"committer,omitempty"`
		RecordId  BankRecordIDV1 `json:"recordId,omitempty"`
		Parent    envelopes.ID `json:"parent"`
	}

	// StateV1 is a copy of envelopes.State for ORM purposes.
	StateV1 struct {
		Budget   envelopes.ID `json:"budget"`
		Accounts envelopes.ID `json:"accounts"`
	}

	// UserV1 is a copy of envelopes.User for ORM purposes.
	UserV1 struct {
		FullName string `json:"fullName"`
		Email    string `json:"email"`
	}

	// BalanceV1 is a copy of envelopes.Balance for ORM purposes.
	BalanceV1 map[envelopes.AssetType]*big.Rat
)

var jsonNumberPatternV1 = regexp.MustCompile(`^(?P<sign>-?)(?P<number>\d+)(?:\.(?P<fraction>\d+))?(?:[eE](?P<exponent>[\-+]?\d+))?$`)

// MarshalJSON converts a BalanceV1 into a serialized JSON object which can be round-tripped back to a BalanceV1.
func (b BalanceV1) MarshalJSON() ([]byte, error) {
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
		err = encoder.Encode(formatRatV1(b[envelopes.AssetType(assetTypes[i])]))
		if err != nil {
			return nil, err
		}
		buf.Truncate(buf.Len() - 1)
		_, err = fmt.Fprint(buf, ",")
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

// UnmarshalJSON reconstitutes a BalanceV1 that had previously been converted into a JSON object by MarshalJSON.
func (b *BalanceV1) UnmarshalJSON(text []byte) error {
	type RawBalance map[envelopes.AssetType]json.Number

	unmarshaled := RawBalance{}

	dec := json.NewDecoder(bytes.NewReader(text))
	err := dec.Decode(&unmarshaled)
	if err != nil {
		return err
	}

	retval := make(BalanceV1, len(unmarshaled))
	*b = retval

	for k, v := range unmarshaled {
		var parsed *big.Rat
		parsed, err = parseRatV1(v)
		if err != nil {
			return err
		}
		retval[k] = parsed
	}
	return nil
}

func formatRatV1(rat *big.Rat) json.Number {
	return json.Number(rat.FloatString(3))
}

func parseRatV1(input json.Number) (*big.Rat, error) {
	var err error
	numerator := int64(0)
	denominator := int64(1)
	var sign int64 = 1

	match := jsonNumberPatternV1.FindStringSubmatch(string(input))
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
