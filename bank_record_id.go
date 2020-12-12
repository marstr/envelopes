// Copyright 2020 Martin Strobel
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

// "But The banks are made of marble,
//  with a guard at every door
//  And the vaults are made of silver
//  That the workers sweated for"
// - Banks of Marble, The Weavers

package envelopes

// A unique identifier provided by a financial institution that is associated with precisely one transfer of funds.
//
// There may be semantics embedded in this string, or it may be a randomly generated UUID. Because the details will be
// specific to the institution that provided the data, we will treat this as an opaque string for this type. Wrappers
// of this type that are specific to the institution to extract semantics are encouraged.
type BankRecordID string

// Equal determines whether or not two BankRecordIDs are identical to each other.
func (bri BankRecordID) Equal(other BankRecordID) bool {
	return string(bri) == string(other)
}

// String presents a BankRecordID as a string.
func (bri BankRecordID) String() string {
	return string(bri)
}
