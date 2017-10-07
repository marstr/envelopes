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

package condition

type binary struct {
	Left  Conditioner
	Right Conditioner
}

type And binary

func (a And) Apply() bool {
	return a.Left.Apply() && a.Right.Apply()
}

type Or binary

func (o Or) Apply() bool {
	return o.Left.Apply() || o.Right.Apply()
}

type Not struct {
	Conditioner
}

func (n Not) Apply() bool {
	return !n.Conditioner.Apply()
}

type Bool bool

func (b Bool) Apply() bool {
	return bool(b)
}
