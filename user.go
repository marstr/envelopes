package envelopes

import (
	"bytes"
	"fmt"
	"regexp"
)

const (
	fullNameTag = "full name:"
	emailTag    = "email:"
)

var (
	userTextPattern = regexp.MustCompile(`^\s*` + fullNameTag + `\s*(.+), ` + emailTag + `\s*([a-zA-Z0-9_\-\.]+@[a-zA-Z0-9_\-\.]+\.[a-zA-Z]{2,5})\s*$`)
)

// User captures metadata about the people who are creating transactions.
type User struct {
	FullName string
	Email    string
}

// MarshalText creates a serialized form of all information defining a User.
func (u User) MarshalText() ([]byte, error) {
	retval := &bytes.Buffer{}
	const formatStr = fullNameTag + " %s, " + emailTag + " %s"
	_, err := fmt.Fprintf(retval, formatStr, u.FullName, u.Email)
	if err != nil {
		return nil, err
	}
	return retval.Bytes(), nil
}

// UnmarshalText reconstitutes a User from the text provided.
func (u *User) UnmarshalText(text []byte) error {
	match := userTextPattern.FindStringSubmatch(string(text))
	if len(match) == 0 {
		return fmt.Errorf("%q is not recognized as a marshaled user", string(text))
	}

	u.FullName = match[1]
	u.Email = match[2]
	return nil
}

func (u User) String() string {
	result, err := u.MarshalText()
	if err != nil {
		panic(err)
	}
	return string(result)
}
