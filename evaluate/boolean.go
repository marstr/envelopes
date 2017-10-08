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
	Left  Conditioner
	Right Conditioner
}

type And binary

func (a And) Evaluate() bool {
	return a.Left.Evaluate() && a.Right.Evaluate()
}

type Or binary

func (o Or) Evaluate() bool {
	return o.Left.Evaluate() || o.Right.Evaluate()
}

type Not struct {
	Conditioner
}

func (n Not) Evaluate() bool {
	return !n.Conditioner.Evaluate()
}

type Bool bool

func (b Bool) Evaluate() bool {
	return bool(b)
}
