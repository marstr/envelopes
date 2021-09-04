// " Money
//   It's a crime
//   Share it fairly
//   But don't take a slice of my pie
//   Money
//   So they say
//   Is the root of all evil today
//   But if you ask for a raise
//   It's no surprise that they're giving none away"
// - Money, Pink Floyd

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

const (
	// DefaultAsset is the label that will be used when parsing a balance given as just a number i.e. with no label.
	DefaultAsset AssetType = "USD"
)

// Exchange represents the known conversion rates from one asset to another. For example an instance of Exchange may
// contain the rates needed to get all types TO United States Dollars, from a host of other types of assets, like Euros,
// shares of stock, etc.
type Exchange map[AssetType]float64

// ErrUnknownAsset indicates that an asset was requested that is not present.
type ErrUnknownAsset AssetType

func (e ErrUnknownAsset) Error() string {
	return fmt.Sprintf("could not find AssetType %s", AssetType(e))
}

// Balance captures an amount of USD pennies.
type Balance map[AssetType]*big.Rat

var zero = big.NewRat(0, 100)

// Copy creates a new Balance that has identical values to b.
func (b Balance) Copy() Balance {

	retval := make(Balance, len(b))

	for k, v := range b {
		retval[k] = &big.Rat{}
		retval[k].Set(v)
	}

	return retval
}

// Add combines two balances, summing any shared components, and including unmatched components without further
// processing.
// Returns a new instance of a Balance, without modifying the two original balances.
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

// Sub combines two balances, however all of the parameters magnitudes are treated as if they are inverted.
// See Add for more behavior details.
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
	bNonZero := b.nonZeroBalances()
	if bNonZero == 0 && other == nil {
		return true
	} else if other == nil {
		return false
	}

	if bNonZero != other.nonZeroBalances() {
		return false
	}

	for id, mag := range b {
		if mag.Cmp(zero) == 0 {
			continue
		}

		if otherMag, ok := other[id]; !ok || mag.Cmp(otherMag) != 0 {
			return false
		}
	}
	return true
}

func (b Balance) nonZeroBalances() uint {
	count := uint(0)
	for _, mag := range b {
		if mag.Cmp(zero) != 0 {
			count++
		}
	}
	return count
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

// Normalize finds the total value of a Balance, but expresses the answer as a scalar instead of a multi-component
// Balance.
func (b Balance) Normalize(rates Exchange) (*big.Rat, error) {
	sum := new(big.Rat)
	var scaled big.Rat

	for k, v := range b {
		if rawRate, ok := rates[k]; ok {
			rate := new(big.Rat).SetFloat64(rawRate)
			scaled.Mul(v, rate)
			sum.Add(sum, &scaled)
		} else {
			return nil, ErrUnknownAsset(k)
		}
	}

	return sum, nil
}

func (b Balance) String() string {
	const defaultResult = "USD 0.00"
	const precision = 3

	if len(b) == 1 { // When there's only a single asset type - we don't want to pare or deal with extra allocations.
		for k := range b {
			return fmt.Sprintf("%s %s", k, b[k].FloatString(precision))
		}
	} else if len(b) > 1 { // When there are multiple asset types, we want to remove unnecessary components
		b.pare()

		if len(b) == 0 {
			// Just like the default case below, if there are multiple asset types that should all fall away, skip
			// all further processing.
			return defaultResult
		}

		keys := make([]string, 0, len(b))
		for key := range b {
			keys = append(keys, string(key))
		}
		sort.Strings(keys)

		buf := &bytes.Buffer{}

		for i := range keys {
			fmt.Fprintf(buf, "%s %s:", keys[i], b[AssetType(keys[i])].FloatString(precision))
		}

		buf.Truncate(buf.Len() - 1)
		return buf.String()
	}

	// In the default case, where balance is zero because there are no assets, we want to continue keeping the same IDs
	// that were previously generated. Because previously a zero balance specifically meant that there were zero USD, to
	// preserve the existing persist package's behavior without any breaking changes, we must assume the value
	// "USD 0.00" here.
	return defaultResult
}

var (
	balancePattern = regexp.MustCompile(`(?m:^\s*(?P<id>[^\s\-\d]+?)??\s*(?P<magnitude>-?(?:[\d]*|(?:\d{1,3}(?:,\d{3})+))(?:\.\d+)?)$)`)
)

// ParseBalanceWithDefault extracts information about a balance from text. Any line items that do not have a label are
// treated as the specified default asset type.
// Lines with the same asset type are summed together.
func ParseBalanceWithDefault(raw []byte, def AssetType) (Balance, error) {
	var created Balance
	const noMatchText = "unable to find balance in text"
	clauses := strings.Split(string(raw), ":")
	for _, clause := range clauses {

		matches := balancePattern.FindAllStringSubmatch(clause, -1)
		if len(matches) == 0 {
			return nil, errors.New(noMatchText)
		}

		for _, match := range matches {
			if len(match[0]) == 0 {
				continue
			}

			id := AssetType(match[1])
			rawMagnitude := match[2]

			rawMagnitude = strings.Replace(string(rawMagnitude), ",", "", -1)

			rehydrated := new(big.Rat)

			if err := rehydrated.UnmarshalText([]byte(rawMagnitude)); err != nil {
				return nil, err
			}

			if id == "" {
				id = def
			}

			if created == nil {
				created = make(Balance)
			}

			if existing, ok := created[id]; ok {
				created[id].Add(existing, rehydrated)
			} else {
				created[id] = rehydrated
			}
		}

		var err error
		if created == nil {
			err = errors.New(noMatchText)
			return nil, err
		}
	}

	return created, nil
}

// ParseBalance converts between a string representation of an amount of dollars
// into an int64 number of cents.
func ParseBalance(raw []byte) (result Balance, err error) {
	return ParseBalanceWithDefault(raw, DefaultAsset)
}

// pare removes components of a balance that are inconsequential - i.e. magnitude of zero.
func (b Balance) pare() {
	for k, v := range b {
		if v.Cmp(zero) == 0 {
			delete(b, k)
		}
	}
}
