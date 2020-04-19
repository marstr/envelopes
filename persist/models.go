package persist

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
	// Budget is copy of envelopes.Budget for ORM purposes.
	Budget struct {
		Balance  Balance
		Children map[string]envelopes.ID
	}

	// Transaction is a copy of envelopes.Transaction for ORM purposes.
	Transaction struct {
		State       envelopes.ID
		PostedTime  time.Time
		ActualTime  time.Time
		EnteredTime time.Time
		Amount      Balance
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

	// Balance is a copy of envelopes.Balance for ORM purposes.
	Balance map[envelopes.AssetType]*big.Rat
)

var jsonNumberPattern = regexp.MustCompile(`^(?P<sign>-?)(?P<number>\d+)(?:\.(?P<fraction>\d+))?(?:[eE](?P<exponent>[\-+]?\d+))?$`)

func (b Balance) MarshalJSON() ([]byte, error) {
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
		err = encoder.Encode(formatRat(b[envelopes.AssetType(assetTypes[i])]))
		if err != nil {
			return nil, err
		}
		buf.Truncate(buf.Len() - 1)
		_, err = fmt.Fprint(buf, ",")
	}
	buf.Truncate(buf.Len() - 1)
	_, err = fmt.Fprint(buf, "}")
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (b *Balance) UnmarshalJSON(text []byte) error {
	type RawBalance map[envelopes.AssetType]json.Number

	unmarshaled := RawBalance{}

	dec := json.NewDecoder(bytes.NewReader(text))
	err := dec.Decode(&unmarshaled)
	if err != nil {
		return err
	}

	retval := make(Balance, len(unmarshaled))
	*b = retval

	for k, v := range unmarshaled {
		var parsed *big.Rat
		parsed, err = parseRat(v)
		if err != nil {
			return err
		}
		retval[k] = parsed
	}
	return nil
}

func formatRat(rat *big.Rat) json.Number {
	return json.Number(rat.FloatString(3))
}

func parseRat(input json.Number) (*big.Rat, error) {
	var err error
	numerator := int64(0)
	denominator := int64(1)
	var sign int64 = 1

	match := jsonNumberPattern.FindStringSubmatch(string(input))
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
