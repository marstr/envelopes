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
	underlyer map[string]int64
}

func (accs Accounts) deepCopy() (updated Accounts) {
	updated.underlyer = make(map[string]int64, len(accs.underlyer))

	for name, balance := range accs.underlyer {
		updated.underlyer[name] = balance
	}
	return
}

func NewAccounts(accs map[string]int64) (created Accounts) {
	created.underlyer = accs
	created = created.deepCopy()
	return
}

func (accs Accounts) ID() (id ID) {
	id, _ = NewID(accs)
	return
}

func (accs Accounts) AddAccount(name string, balance int64) (updated Accounts, added bool) {
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
