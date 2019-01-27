package envelopes

import (
	"fmt"
	"strconv"
	"strings"
)

type Balance int64

func (b Balance) String() string {
	transformed := float64(b) / 100
	return fmt.Sprintf("USD %0.2f", transformed)
}

// ParseBalance converts between a string representation of an amount of dollars
// into an int64 number of cents.
func ParseBalance(raw string) (result Balance, err error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "$")
	raw = strings.TrimPrefix(raw, "USD")
	raw = strings.TrimSpace(raw)
	raw = strings.Replace(raw, ",", "", -1)
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return
	}

	if parsed >= 0 {
		result = Balance(parsed*100 + .5)
	} else {
		result = Balance(parsed*100 - .5)
	}
	return
}
