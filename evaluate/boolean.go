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

package evaluate

type binary struct {
	Left  Evaluater
	Right Evaluater
}

// And is an aggregate `Evaluater` which fulfills Boolean AND behavior.
type And binary

// Evaluate performs a Boolean AND operation.
func (a And) Evaluate() bool {
	return a.Left.Evaluate() && a.Right.Evaluate()
}

// Or is an aggregate `Evaluater` which fulfills Boolean OR behavior.
type Or binary

// Evaluate performs a Boolean OR operation.
func (o Or) Evaluate() bool {
	return o.Left.Evaluate() || o.Right.Evaluate()
}

// Not decorates an `Evaluater` fulfilling Boolean NOT behavior.
type Not struct {
	Evaluater
}

// Evaluate performs a Boolean NOT operation.
func (n Not) Evaluate() bool {
	return !n.Evaluater.Evaluate()
}

// Bool is a primitive `Evaluater` which always evaluates to `true` or `false`.
type Bool bool

// Evaluate returns a Boolean primitive.
func (b Bool) Evaluate() bool {
	return bool(b)
}
