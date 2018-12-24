// Copyright 2017 Martin Strobel
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package envelopes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

type Accounts struct {
	underlyer map[string]Balance
}

func (accs Accounts) deepCopy() (updated Accounts) {
	updated.underlyer = make(map[string]Balance, len(accs.underlyer))

	for name, balance := range accs.underlyer {
		updated.underlyer[name] = balance
	}
	return
}

func NewAccounts(accs map[string]Balance) (created Accounts) {
	created.underlyer = accs
	created = created.deepCopy()
	return
}

func (accs Accounts) ID() (id ID) {
	id, _ = NewID(accs)
	return
}

func (accs Accounts) AddAccount(name string, balance Balance) (updated Accounts, added bool) {
	if _, ok := accs.underlyer[name]; ok {
		updated = accs
		added = false
		return
	}

	updated = accs.deepCopy()
	updated.underlyer[name] = balance
	added = true
	return
}

func (accs Accounts) Balance(name string) (bal Balance, ok bool) {
	bal, ok = accs.underlyer[name]
	return
}

func (accs Accounts) WithBalance(name string, balance Balance) (updated Accounts, ok bool) {
	if _, ok = accs.underlyer[name]; !ok {
		updated = accs
		return
	}

	updated = accs.deepCopy()
	updated.underlyer[name] = balance

	return
}

func (accs Accounts) AdjustBalance(name string, impact Balance) (updated Accounts, ok bool) {
	var previousBalance Balance
	if previousBalance, ok = accs.underlyer[name]; !ok {
		updated = accs
		return
	}

	updated = accs.deepCopy()
	updated.underlyer[name] = previousBalance + impact
	return
}

func (accs Accounts) Size() int {
	return len(accs.underlyer)
}

func (accs Accounts) Names() (names []string) {
	names = make([]string, 0, len(accs.underlyer))

	for name := range accs.underlyer {
		names = append(names, name)
	}

	return
}

// RenameAccount changes the name associated with an account. If the new name already exists in this collection of
// accounts, nothing happens and this function return the original object and false.
func (accs Accounts) RenameAccount(old, new string) (updated Accounts, ok bool) {
	_, ok = accs.underlyer[new]
	if !ok {
		updated = accs
		return
	}

	updated = accs.deepCopy()
	updated.underlyer[new] = updated.underlyer[old]
	delete(accs.underlyer, old)
	return
}

func (accs Accounts) HasAccount(name string) (ok bool) {
	_, ok = accs.underlyer[name]
	return
}

func (accs Accounts) RemoveAccount(name string) (updated Accounts, removed bool) {
	if _, ok := accs.underlyer[name]; !ok {
		updated = accs
		removed = false
		return
	}

	updated = accs.deepCopy()
	delete(updated.underlyer, name)
	removed = true
	return
}

func (accs Accounts) AsMap() map[string]Balance {
	return accs.deepCopy().underlyer
}

func (accs Accounts) MarshalJSON() ([]byte, error) {
	names := make([]string, 0, len(accs.underlyer))

	for name := range accs.underlyer {
		names = append(names, name)
	}

	sort.Strings(names)

	builder := new(bytes.Buffer)

	builder.WriteRune('{')
	if len(names) > 0 {
		for _, name := range names {
			fmt.Fprintf(builder, "%q:%d,", name, accs.underlyer[name])
		}
		builder.Truncate(builder.Len() - 1)
	}
	builder.WriteRune('}')

	return builder.Bytes(), nil
}

func (accs *Accounts) UnmarshalJSON(contents []byte) (err error) {
	return json.Unmarshal(contents, &accs.underlyer)
}
