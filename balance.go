package envelopes

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strings"
)

// AssetType is a character code the uniquely identifies a type of asset. For currencies, that is a three-letter code.
// For securities like stocks, it will be the set of characters that are used to trade shares of those stocks.
// For instance:
// United States Dollar -> USD
// Microsoft Stock Shares -> MSFT
type AssetType string

// Balance captures an amount of USD pennies.
type Balance map[AssetType]*big.Rat

func (b Balance) Add(other Balance) Balance {
	sum := make(Balance, len(b))

	unseen := make(map[AssetType]struct{}, len(other))
	for key := range other {
		unseen[key] = struct{}{}
	}

	for key, bMag := range b {
		if otherMag, ok := other[key]; ok {
			delete(unseen, key)

			newMag := &big.Rat{}
			sum[key] = newMag.Add(bMag, otherMag)
		} else {
			clone := *bMag
			sum[key] = &clone
		}
	}

	for key := range unseen {
		clone := *other[key]
		sum[key] = &clone
	}

	return sum
}

func (b Balance) Sub(other Balance) Balance {
	sum := make(Balance, len(b))

	unseen := make(map[AssetType]struct{}, len(other))
	for key := range other {
		unseen[key] = struct{}{}
	}

	for key, bMag := range b {
		if otherMag, ok := other[key]; ok {
			delete(unseen, key)

			newMag := &big.Rat{}
			sum[key] = newMag.Sub(bMag, otherMag)
		} else {
			clone := *bMag
			sum[key] = &clone
		}
	}

	for key := range unseen {
		var negated big.Rat
		sum[key] = negated.Neg(other[key])
	}

	return sum
}

// Equal determines whether two balances are compromised of the same mix of assets with the same magnitude assigned to
// each.
func (b Balance) Equal(other Balance) bool {
	if len(b) != len(other) {
		return false
	}

	for id, mag := range b {
		if otherMag, ok := other[id]; !ok || mag.Cmp(otherMag) != 0 {
			return false
		}
	}
	return true
}

// Negate inverts the sign of each entry in a balance.
func (b Balance) Negate() Balance {
	retval := make(Balance, len(b))

	for key, value := range b {
		var negated big.Rat
		retval[key] = negated.Neg(value)
	}

	return retval
}

// Scale multiplies each entry in a Balance by a constant amount. This may be useful for diving
func (b Balance) Scale(s float64) Balance {
	retval := make(Balance, len(b))

	t := new(big.Rat).SetFloat64(s)
	for key, value := range b {
		var scaled big.Rat
		retval[key] = scaled.Mul(value, t)
	}

	return retval
}

func (b Balance) String() string {
	if len(b) > 0 {
		keys := make([]string, 0, len(b))
		for key := range b {
			keys = append(keys, string(key))
		}
		sort.Strings(keys)

		buf := &bytes.Buffer{}

		for i := range keys {
			fmt.Fprintf(buf, "%s %s, ", keys[i], b[AssetType(keys[i])].FloatString(3))
		}

		buf.Truncate(buf.Len() - 2)
		return buf.String()
	}

	// In the default case, where balance is zero because there are no assets, we want to continue keeping the same IDs
	// that were previously generated. Because previously a zero balance specifically meant that there were zero USD, to
	// preserve the existing persist package's behavior without any breaking changes, we must assume the value
	// "USD 0.00" here.
	return "USD 0.00"
}

var (
	balancePattern = regexp.MustCompile(`(?m:^\s*(?P<id>[^\s\-\d]+?)??\s*(?P<magnitude>-?(?:[\d]*|(?:\d{1,3}(?:,\d{3})+))(?:\.\d+)?)$)`)
)

func ParseBalanceWithDefault(raw []byte, def AssetType) (Balance, error) {
	var created Balance

	const noMatchText = "unable to find balance in text"

	matches := balancePattern.FindAllSubmatch(raw, -1)
	if len(matches) == 0 {
		return nil, errors.New(noMatchText)
	}

	for _, match := range matches {
		if len(match[0]) == 0 {
			continue
		}

		id := AssetType(match[1])
		rawMagnitude := match[2]

		rawMagnitude = []byte(strings.Replace(string(rawMagnitude), ",", "", -1))

		var rehydrated big.Rat

		if err := rehydrated.UnmarshalText(rawMagnitude); err != nil {
			return nil, err
		}

		if id == "" {
			id = def
		} else if expanded, ok := CurrencyShorthands[id]; ok {
			id = expanded
		}

		if created == nil {
			created = make(Balance)
		}

		created[id] = &rehydrated
	}

	var err error
	if created == nil {
		err = errors.New(noMatchText)
	}

	return created, err
}

// ParseBalance converts between a string representation of an amount of dollars
// into an int64 number of cents.
func ParseBalance(raw []byte) (result Balance, err error) {
	return ParseBalanceWithDefault(raw, DefaultAsset)
}
