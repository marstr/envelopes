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

// Package evaluate allows a means of programatically evaluating a `State`. This
// is especially useful when used in tandem with `distribution.Branch` or when
// building tools for giving suggestions for improving financial help.
package evaluate

// Evaluater allows for polymorphic determiniation of whether or not to apply
// an `envelopes.Effect`
type Evaluater interface {
	Evaluate() bool
}
