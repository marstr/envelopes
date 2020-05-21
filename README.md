<p align="center"><a href="./README.md#logo"><img src="https://github.com/ashleymcnamara/gophers/raw/4ddd92f3f0830f5d9a9eab50c410878249fe6515/NERDY.png" width="360"></a></p>

# envelopes
[![Build Status](https://travis-ci.org/marstr/envelopes.svg?branch=master)](https://travis-ci.org/marstr/envelopes)

Been scouring the internet looking for a personal finance library written in Go flexible enough to model your budget? 
You have arrived at a library that aims to give you the building blocks to represent your finances no matter how you 
look at them.

## include
Whether you're using Go modules or $GOPATH, you can acquire this library by simply running the following command from 
the root directory of your project:

``` bash
$ go get github.com/marstr/envelopes
```

## model overview

### balances

The [`Balance`](https://pkg.go.dev/github.com/marstr/envelopes?tab=doc#Balance) type represents an amount. You'll notice
it isn't a scalar type. This is so accounts which hold multiple types of assets (like brokerage accounts) can have their
contents fully represented.

Let's say you're working in Euros, you could initialize a balance of €10,76 the following way:

``` Go
mySimpleBalance := envelopes.Balance{
    "EUR": big.NewRat(1076, 100),
}
```

How about the balance of a brokerage account that has 90 shares of T-Mobile (TMUS), and 89.143 shares of Tesla (TSLA)?
 
``` Go
myComplexBalance := envelopes.Balance{
    "TSLA": big.NewRat(89143, 1000),
    "TMUS": big.NewRat(90, 1),
}
```

When you do arithmetic with balances, each term will be combined with like terms. That is to say, when you [`Add`](https://pkg.go.dev/github.com/marstr/envelopes?tab=doc#Balance.Add)
or [`Sub`](https://pkg.go.dev/github.com/marstr/envelopes?tab=doc#Balance.Sub) balances, the EUR component of one 
balance will be summed against the other balance's EUR component, etc. Notably, because different stocks and currencies 
have different values, greater than and less than operations can't be done directly on instances of `Balance`. You'll 
need to [`Normalize`](https://pkg.go.dev/github.com/marstr/envelopes?tab=doc#Balance.Normalize) them first.

While not all modern currencies in the world are decimalized, even the exceptions are subdivided only once by a factor 
of five. The two exceptions are the [Malagasy ariary](https://en.wikipedia.org/wiki/Malagasy_ariary) and the 
[Mauritanian ouguiya](https://en.wikipedia.org/wiki/Mauritanian_ouguiya)). So at the time of writing, this library 
should work reasonably well for any currency.

### accounts

The [`Accounts`](https://pkg.go.dev/github.com/marstr/envelopes?tab=doc#Accounts) type seeks to help answer the question
"Where is my money?" It is a collection of account names to the [Balance](#balances) in that account. Accounts are flat,
and do not contain other accounts. For accounts like bank accounts and brokerage accounts, which hold assets, the
magnitudes should be positive. For accounts like credit cards, which hold liabilities, the intention is to have balances
be negative. In this regard, the sum of all of the balances of your accounts should be the total of your tracked value.
As you spend money (be that on a credit card, or with a debit card) the total goes down. This is a little simpler than
traditional doubly-entry accounting, which has credits/debits get swapped based on the type of the account. 

### budgets

The [`Budget`](https://pkg.go.dev/github.com/marstr/envelopes?tab=doc#Budget) type seeks to help answer the question
"How do I want to spend my money?" Budgets can be nested, and have both a value of just that budget, and a recursive
that is the sum of it and all its children. The system was designed to use both accounts and budgets at the same time,
as we'll discuss in [#states](#states). 

### states

A [`State`](https://pkg.go.dev/github.com/marstr/envelopes?tab=doc#State) combines all your account balances with a root
budget. This reinforces the way the fundamental abstraction that is alluded to in the [#accounts](#accounts) and 
[#balances](#balances) sections above. A state allows you to separate the "where" and the "for what" of you money. The
idea here is that you may want to have more control than "all of my stocks are being saved for a down payment", or
"the money in this savings account is my emergency budget."

### transactions

A [`Transaction`](https://pkg.go.dev/github.com/marstr/envelopes?tab=doc#Transaction) captures metadata around a change 
in the current `State`. It could be associated with a financial institution, or it may just capture funds that are 
transferred between two budgets. This library was inspired by [Git](https://git-scm.com), and borrows a lot its 
architecture and ideas. One of the most notable consequences is that instead of directly using the amount of a
transaction as a delta, instead a transaction points to the `State` that your accounts and budget are in immediately 
after the transaction was completed. If you're familiar with Git's object model, this is the same behavior Git uses when
creating a commit.

### immutability

Conceptually, each of the models above are immutable. A SHA1 uniquely identifies each one, so that it can be stored to
disk and referred to later unambiguously. This immutability has not made it's way into the code, in order to allow
objects to be built up and modified more easily between being committed to disk.

## related projects

### baronial
[baronial](https://github.com/marstr/baronial/tree/master/README.md) is a command line interface that exposes the 
concepts enumerated above in a way that is ready to be scripted against, or just interacted with by a person comfortable
working in a terminal.

### envelopes-azure

[envelopes-azure](https://github.com/marstr/envelopes-azure/tree/master/README.md) implements a client for storing
serialized `IDer`s in Azure, instead of just using filesystem constructs.

## logo

The nerdy gopher is artwork by the amazing Ashley McNamara, and is based on the gopher drawn by Renée French. It is 
licensed under the [
Attribution-NonCommercial-ShareAlike 4.0 International](https://github.com/ashleymcnamara/gophers/blob/4ddd92f3f0830f5d9a9eab50c410878249fe6515/LICENSE)
License. You can find this gopher, and many more like it, at: https://github.com/ashleymcnamara/gophers 