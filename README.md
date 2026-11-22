<p align="center"><a href="./README.md#logo"><img src="https://github.com/ashleymcnamara/gophers/raw/4ddd92f3f0830f5d9a9eab50c410878249fe6515/NERDY.png" width="360"></a></p>

# envelopes
[![PkgGoDev](https://pkg.go.dev/badge/github.com/marstr/envelopes)](https://pkg.go.dev/github.com/marstr/envelopes)
[![Build](https://github.com/marstr/envelopes/actions/workflows/build.yml/badge.svg)](https://github.com/marstr/envelopes/actions/workflows/build.yml)
[![CodeQL](https://github.com/marstr/envelopes/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/marstr/envelopes/actions/workflows/codeql-analysis.yml)

Been scouring the internet looking for a personal finance library written in Go? Having trouble finding one flexible 
enough to model your budget? You have arrived at a library that aims to give you the building blocks to represent your 
finances no matter how you look at them.

## include
Whether you're using Go modules or $GOPATH, you can acquire this library by simply running the following command from 
the root directory of your project:

``` bash
$ go get github.com/marstr/envelopes
```

Go 1.21 or later is required.

## persistence diagnostics

Filesystem persistence is silent by default, including when the executable configures
`slog.Default()`. Supply your application's `*slog.Logger` to opt in:

```go
logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelDebug,
})).With("component", "envelopes")

repo, err := filesystem.OpenRepository(ctx, root,
    filesystem.RepositoryLogger(logger),
)
```

The same option works with `OpenRepositoryWithCache`. For a standalone `FileSystem`
or `FilesystemBankRecordIDIndex`, set its `Logger` field in a keyed struct literal.
Configure it before concurrent use; nil disables logging. The application owns the
handler, destination, level, and any additional attributes or redaction.

Debug records describe filesystem I/O, reference operations, repository configuration,
bank-index operations, and commit/clone progress. Returned failures are also Debug: the executable should
handle and report its final error rather than receive duplicate Error logs from each
library layer. Warn records report failures that existing APIs do not return, such as
ignored close errors or asynchronous index read/parse failures. Arithmetic, cache hits,
JSON codecs, and per-node graph traversal do not emit their own logs.

`persist.Commit` and `persist.BareClone` reuse a repository's logger through the
optional `persist.DiagnosticLoggerProvider` capability; custom repositories can
implement it without changing the existing repository interfaces. Clone prefers
the destination's logger, falling back to the source's. Progress records distinguish
written objects from completed reference updates; clone summaries count successfully
written branches, tags, and transactions without logging every visited object.

Debug logs can contain non-index filesystem paths and object IDs. They do not include
financial payloads, comments, user details, attachment contents, bank-record IDs, or
bank-index filenames (which reversibly encode those IDs). Treat debug logs as
potentially sensitive and choose their destination appropriately.

Repository options are applied after configuration detection, so failures before
logger injection remain returned errors without library logs. Stream-open records
do not mean contents have been consumed, and copy records do not guarantee durable
storage. Directory-listing summaries describe emitted entries; the underlying
enumerator does not expose filesystem errors.

Adding optional logger fields preserves keyed literals but requires updating any
unkeyed `FileSystem` or `FilesystemBankRecordIDIndex` literals.
Repository options now reach the backing object store as well as the returned
filesystem. A new repository saves its selected object layout; changing the layout
of an existing repository is rejected rather than leaving its objects inaccessible.

## model overview

### balances

The [`Balance`](https://pkg.go.dev/github.com/marstr/envelopes?tab=doc#Balance) type represents an amount of funds
available.

Let's say you're working in Euros, you could initialize a balance of €10,76 the following way:

``` Go
mySimpleBalance := envelopes.Balance{
    "EUR": big.NewRat(1076, 100),
}
```

You'll notice above that `Balance` isn't a scalar type, but rather a dictionary. This is so accounts which hold multiple
types of assets (like brokerage accounts) can have their contents fully represented. For instance, what if you need to
represent the balance of a brokerage account that has 90 shares of T-Mobile (TMUS), and 89.143 shares of Tesla (TSLA)?
 
``` Go
myComplexBalance := envelopes.Balance{
    "TSLA": big.NewRat(89143, 1000),
    "TMUS": big.NewRat(90, 1),
}
```

When you do arithmetic with balances, each term will be combined with like terms. That is to say, when you [`Add`](https://pkg.go.dev/github.com/marstr/envelopes?tab=doc#Balance.Add)
or [`Sub`](https://pkg.go.dev/github.com/marstr/envelopes?tab=doc#Balance.Sub) balances, the `"EUR"` component of one 
balance will be summed against the other balance's `"EUR"` component, etc. Notably, because different stocks and 
currencies have different values, greater than and less than operations can't be done directly on instances of `
Balance`. You'll need to [`Normalize`](https://pkg.go.dev/github.com/marstr/envelopes?tab=doc#Balance.Normalize) them 
first.

While not all modern currencies in the world are decimalized, even the exceptions are subdivided only once; by a factor 
of five. The two exceptions are the [Malagasy ariary](https://en.wikipedia.org/wiki/Malagasy_ariary) and the 
[Mauritanian ouguiya](https://en.wikipedia.org/wiki/Mauritanian_ouguiya)). So at the time of writing, this library 
should work reasonably well for any currency.

### accounts

The [`Accounts`](https://pkg.go.dev/github.com/marstr/envelopes?tab=doc#Accounts) type seeks to help answer the question
"Where is my money?" It is a collection of account names to the [Balance](#balances) in each account. Accounts are flat,
and do not contain other accounts. For accounts like bank accounts and brokerage accounts, which hold assets, the
magnitudes should be positive. For accounts like credit cards, which hold liabilities, the intention is to have balances
be negative. In this regard, the sum of the balances of your accounts should be the total of your tracked value. As you 
spend money (be that on a credit card, or with a debit card) the total goes down. This is a little simpler than 
traditional doubly-entry accounting, which has credits/debits get swapped based on the type of the account. 

### budgets

The [`Budget`](https://pkg.go.dev/github.com/marstr/envelopes?tab=doc#Budget) type seeks to help answer the question
"How do I want to spend my money?" Unlike [accounts](#accounts), Budgets can be nested (a budget can live inside another
budget.). Because they can be nested, they have both an immediate balance and a recursive balance, which is the sum of 
its balance and all the balances of its children. The system is designed to use both accounts and budgets at the same 
time, as we'll discuss in [#states](#states). 

### states

A [`State`](https://pkg.go.dev/github.com/marstr/envelopes?tab=doc#State) combines all your account balances with a root
budget. This reinforces the way the fundamental abstraction that is alluded to in the [#accounts](#accounts) and 
[#balances](#balances) sections above. A state allows you to separate the "where" and the "for what" of you money. The
idea here is that you may want to have more control than "all of my stocks are being saved for a down payment", or
"the money in this savings account is my emergency budget."

### transactions

A [`Transaction`](https://pkg.go.dev/github.com/marstr/envelopes?tab=doc#Transaction) captures metadata around a change 
in the current `State`. It could be associated with a financial institution, or it may just capture funds being 
transferred between two budgets. This library was inspired by [Git](https://git-scm.com), and borrows a lot its 
architecture and ideas. One of the most notable consequences is that instead of using amount of a transaction to figure
out the current state, it uses the current state (and the one immediately proceeding it) to figure out the amount of
the transaction.

### immutability

Conceptually, each of the models above are immutable. A SHA1 uniquely identifies each one so that it can be stored to
disk and referred to later unambiguously. This immutability has not made its way into the code, however. This allows
objects to be built up and modified more easily between being committed to disk.

## related projects

### baronial
[baronial](https://github.com/marstr/baronial/tree/main/README.md) is a command line interface that exposes the 
concepts enumerated above in a way that is ready to be scripted against, or just interacted with by a person comfortable
working in a terminal.

### envelopes-azure

[envelopes-azure](https://github.com/marstr/envelopes-azure/tree/master/README.md) implements a client for storing
serialized `IDer`s in Azure, instead of just using filesystem constructs.

## logo

The nerdy gopher is artwork by the amazing Ashley Willis (McNamara), and is based on the gopher drawn by Renée French. It is 
licensed under the [
Attribution-NonCommercial-ShareAlike 4.0 International](https://github.com/ashleymcnamara/gophers/blob/4ddd92f3f0830f5d9a9eab50c410878249fe6515/LICENSE)
License. You can find this gopher, and many more like it, at: https://github.com/ashleymcnamara/gophers 
