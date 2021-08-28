package distribute_test

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/distribute"
)

func ExampleBringToRule_Distribute() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 30)
	defer cancel()

	grocery := envelopes.Budget{Balance: envelopes.Balance{
		"USD": big.NewRat(4833, 100), // $48.33
	}}

	spending := envelopes.Budget{Balance: envelopes.Balance{
		"USD": big.NewRat(2218, 100), // $22.18
	}}

	subject := distribute.NewBringToRule(&grocery, envelopes.Balance{"USD": big.NewRat(100, 1)}, (*distribute.BudgetRule)(&spending))

	amountToCredit := envelopes.Balance{
		"USD": big.NewRat(200, 1),
	}

	if err := subject.Distribute(ctx, amountToCredit); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "couldn't distribute %s: %v\n", amountToCredit, err)
		return
	}

	fmt.Println("Grocery:", grocery.RecursiveBalance())
	fmt.Println("Spending:", spending.RecursiveBalance())

	// Output:
	// Grocery: USD 100.000
	// Spending: USD 170.510
}

func ExampleBudgetRule() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 30)
	defer cancel()

	recipient := envelopes.Budget{Balance: envelopes.Balance{
		"USD": big.NewRat(10, 1),
	}}
	amountToCredit := envelopes.Balance{
		"USD": big.NewRat(5, 1),
	}

	err := (*distribute.BudgetRule)(&recipient).Distribute(ctx, amountToCredit)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "couldn't distribute %s: %v\n", amountToCredit, err)
		return
	}

	fmt.Println(recipient.Balance)

	// Output:
	// USD 15.000
}

func ExamplePercentageRule() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 30)
	defer cancel()

	fed := envelopes.Budget{Balance: envelopes.Balance{
		"USD": big.NewRat(100, 1),
	}}

	starved := envelopes.Budget{Balance: envelopes.Balance{
		"USD": big.NewRat(10, 1),
	}}

	socialism := distribute.NewPercentageRule(2, (*distribute.BudgetRule)(&starved))
	socialism.AddRule(.6, (*distribute.BudgetRule)(&fed))
	socialism.AddRule(.4, (*distribute.BudgetRule)(&starved))

	amountToCredit := envelopes.Balance{
		"USD": big.NewRat(5, 1),
	}

	if err := socialism.Distribute(ctx, amountToCredit); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "couldn't distribute %s: %v\n", amountToCredit, err)
		return
	}

	fmt.Println("Fed:", fed.Balance)
	fmt.Println("Starved:", starved.Balance)

	// Output:
	// Fed: USD 103.000
	// Starved: USD 12.000
}

func ExamplePriorityRule() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 30)
	defer cancel()

	fed := envelopes.Budget{Balance: envelopes.Balance{
		"USD": big.NewRat(100, 1),
	}}

	starved := envelopes.Budget{Balance: envelopes.Balance{
		"USD": big.NewRat(10, 1),
	}}

	capitalism := distribute.NewPriorityRule((*distribute.BudgetRule)(&starved))
	capitalism.AddRule((*distribute.BudgetRule)(&fed), envelopes.Balance{"USD": big.NewRat(5, 1)})

	amountToCredit := envelopes.Balance{
		"USD": big.NewRat(7, 1),
	}

	if err := capitalism.Distribute(ctx, amountToCredit); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "couldn't distribute %s: %v\n", amountToCredit, err)
		return
	}

	fmt.Println("Fed:", fed.Balance)
	fmt.Println("Starved:", starved.Balance)

	// Output:
	// Fed: USD 105.000
	// Starved: USD 12.000
}

func ExamplePriorityRule_Distribute_insufficientFunds() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 30)
	defer cancel()

	fed := envelopes.Budget{Balance: envelopes.Balance{
		"USD": big.NewRat(100, 1),
	}}

	starved := envelopes.Budget{Balance: envelopes.Balance{
		"USD": big.NewRat(10, 1),
	}}

	capitalism := distribute.NewPriorityRule((*distribute.BudgetRule)(&starved))
	capitalism.AddRule((*distribute.BudgetRule)(&fed), envelopes.Balance{"USD": big.NewRat(10, 1)})

	amountToCredit := envelopes.Balance{
		"USD": big.NewRat(6, 1),
	}

	if err := capitalism.Distribute(ctx, amountToCredit); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "couldn't distribute %s: %v\n", amountToCredit, err)
		return
	}

	fmt.Println("Fed:", fed.Balance)
	fmt.Println("Starved:", starved.Balance)

	// Output:
	// Fed: USD 110.000
	// Starved: USD 6.000
}

