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

package evaluate_test

import (
	"fmt"

	"github.com/marstr/envelopes/evaluate"
)

func ExampleBool_Evaluate() {
	fmt.Println(evaluate.Bool(true).Evaluate())
	fmt.Println(evaluate.Bool(false).Evaluate())

	// Output:
	// true
	// false
}

func ExampleAnd_Evaluate() {
	subject := evaluate.And{}

	subject.Left = evaluate.Bool(false)
	subject.Right = evaluate.Bool(false)
	fmt.Println(subject.Evaluate())

	subject.Left = evaluate.Bool(false)
	subject.Right = evaluate.Bool(true)
	fmt.Println(subject.Evaluate())

	subject.Left = evaluate.Bool(true)
	subject.Right = evaluate.Bool(false)
	fmt.Println(subject.Evaluate())

	subject.Left = evaluate.Bool(true)
	subject.Right = evaluate.Bool(true)
	fmt.Println(subject.Evaluate())

	// Output:
	// false
	// false
	// false
	// true
}

func ExampleOr_Evaluate() {
	subject := evaluate.Or{}

	subject.Left = evaluate.Bool(false)
	subject.Right = evaluate.Bool(false)
	fmt.Println(subject.Evaluate())

	subject.Left = evaluate.Bool(false)
	subject.Right = evaluate.Bool(true)
	fmt.Println(subject.Evaluate())

	subject.Left = evaluate.Bool(true)
	subject.Right = evaluate.Bool(false)
	fmt.Println(subject.Evaluate())

	subject.Left = evaluate.Bool(true)
	subject.Right = evaluate.Bool(true)
	fmt.Println(subject.Evaluate())

	// Output:
	// false
	// true
	// true
	// true
}

func ExampleNot_Evaluate() {
	subject := evaluate.Not{}

	subject.Evaluater = evaluate.Bool(false)
	fmt.Println(subject.Evaluate())

	subject.Evaluater = evaluate.Bool(true)
	fmt.Println(subject.Evaluate())

	// Output:
	// true
	// false
}
