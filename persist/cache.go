package persist

import (
	"context"
	"fmt"
	"reflect"

	"github.com/marstr/envelopes"

	"github.com/marstr/collection/v2"
)

// ErrTypeMismatch is when an
type ErrTypeMismatch struct {
	Got  reflect.Type
	Want reflect.Type
}

func NewErrTypeMismatch(got interface{}, want interface{}) ErrTypeMismatch {
	return ErrTypeMismatch{
		Got:  reflect.TypeOf(got),
		Want: reflect.TypeOf(want),
	}
}

func (err ErrTypeMismatch) Error() string {
	return fmt.Sprintf("object to load is of type %q, expected type %q.", err.Got.Name(), err.Want.Name())
}

// Cache provides a place to stash objects between calls to an actual Loader/Writer which are presumably more expensive.
// It can be used without setting a backing Loader/Writer.
type Cache struct {
	lruCache *collection.LRUCache[envelopes.ID, envelopes.IDer]
	Loader
	Writer
}

// NewCache creates a new empty cache of IDers.
func NewCache(capacity uint) *Cache {
	return &Cache{
		lruCache: collection.NewLRUCache[envelopes.ID, envelopes.IDer](capacity),
	}
}

// WriteTransaction adds a Transaction to this cache. If Writer isn't nil, it is immediately invoked.
func (c Cache) WriteTransaction(ctx context.Context, subject envelopes.Transaction) error {
	c.lruCache.Put(subject.ID(), &subject)
	if c.Writer == nil {
		return nil
	}
	return c.Writer.WriteTransaction(ctx, subject)
}

// WriteState adds a State to this cache. If Writer isn't nil, it is immediately invoked.
func (c Cache) WriteState(ctx context.Context, subject envelopes.State) error {
	c.lruCache.Put(subject.ID(), &subject)
	if c.Writer == nil {
		return nil
	}
	return c.Writer.WriteState(ctx, subject)
}

// WriteBudget adds a Budget to this cache. If Writer isn't nil, it is immediately invoked.
func (c Cache) WriteBudget(ctx context.Context, subject envelopes.Budget) error {
	c.lruCache.Put(subject.ID(), &subject)
	if c.Writer == nil {
		return nil
	}
	return c.Writer.WriteBudget(ctx, subject)
}

// WriteAccounts adds an instance of Accounts to this cache. If Writer isn't nil, it is immediately invoked.
func (c Cache) WriteAccounts(ctx context.Context, subject envelopes.Accounts) error {
	c.lruCache.Put(subject.ID(), subject)
	if c.Writer == nil {
		return nil
	}
	return c.Writer.WriteAccounts(ctx, subject)
}

// Load copies the desired object from the Cache into destination. If the requested option is present in the cache, it
// doesn't invoke Loader. If it is not present, and Loader is not nil, it invokes Loader and adds the result to the
// cache.
func (c Cache) Load(ctx context.Context, subject envelopes.ID, destination envelopes.IDer) error {
	cached, ok := c.lruCache.Get(subject)
	if !ok {
		return c.miss(ctx, subject, destination)
	}
	return c.hit(ctx, cached, destination)
}

func (c Cache) miss(ctx context.Context, subject envelopes.ID, destination envelopes.IDer) error {
	if c.Loader == nil {
		return ErrObjectNotFound(subject)
	}

	err := c.Loader.Load(ctx, subject, destination)
	if err == nil {
		c.lruCache.Put(subject, destination)
	}
	return err
}

func (c Cache) hit(_ context.Context, cached interface{}, destination envelopes.IDer) error {
	switch destination.(type) {
	case *envelopes.Transaction:
		cast, ok := cached.(*envelopes.Transaction)
		if !ok {
			return NewErrTypeMismatch(cached, destination)
		}
		*(destination).(*envelopes.Transaction) = *cast
	case *envelopes.Budget:
		cast, ok := cached.(*envelopes.Budget)
		if !ok {
			return NewErrTypeMismatch(cached, destination)
		}
		*(destination).(*envelopes.Budget) = *cast
	case *envelopes.State:
		cast, ok := cached.(*envelopes.State)
		if !ok {
			return NewErrTypeMismatch(cached, destination)
		}
		*(destination).(*envelopes.State) = *cast
	case *envelopes.Accounts:
		cast, ok := cached.(*envelopes.Accounts)
		if !ok {
			return NewErrTypeMismatch(cached, destination)
		}
		*(destination).(*envelopes.Accounts) = *cast
	default:
		return NewErrUnloadableType(destination)
	}

	return nil
}
