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

// LoadTransaction copies the desired object from the Cache into destination. If the requested option is present in the
// cache, it doesn't invoke Loader. If it is not present, and Loader is not nil, it invokes Loader and adds the result
// to the cache.
func (c Cache) LoadTransaction(ctx context.Context, subject envelopes.ID, destination *envelopes.Transaction) error {
	cached, ok := c.lruCache.Get(subject)
	if !ok {
		return c.missTransaction(ctx, subject, destination)
	}
	return c.hit(ctx, cached, destination)
}

func (c Cache) missTransaction(ctx context.Context, subject envelopes.ID, destination *envelopes.Transaction) error {
	if c.Loader == nil {
		return ErrObjectNotFound(subject)
	}

	var cacheCopy envelopes.Transaction
	err := c.Loader.LoadTransaction(ctx, subject, &cacheCopy)
	if err == nil {
		c.lruCache.Put(subject, &cacheCopy)
	}
	(*destination) = cacheCopy
	return err
}

// LoadState copies the desired object from the Cache into destination. If the requested option is present in the
// cache, it doesn't invoke Loader. If it is not present, and Loader is not nil, it invokes Loader and adds the result
// to the cache.
func (c Cache) LoadState(ctx context.Context, subject envelopes.ID, destination *envelopes.State) error {
	cached, ok := c.lruCache.Get(subject)
	if !ok {
		return c.missState(ctx, subject, destination)
	}
	return c.hit(ctx, cached, destination)
}

func (c Cache) missState(ctx context.Context, subject envelopes.ID, destination *envelopes.State) error {
	if c.Loader == nil {
		return ErrObjectNotFound(subject)
	}

	err := c.Loader.LoadState(ctx, subject, destination)
	if err == nil {
		c.lruCache.Put(subject, destination)
	}
	return err
}

// LoadBudget copies the desired object from the Cache into destination. If the requested option is present in the
// cache, it doesn't invoke Loader. If it is not present, and Loader is not nil, it invokes Loader and adds the result
// to the cache.
func (c Cache) LoadBudget(ctx context.Context, subject envelopes.ID, destination *envelopes.Budget) error {
	cached, ok := c.lruCache.Get(subject)
	if !ok {
		return c.missBudget(ctx, subject, destination)
	}
	return c.hit(ctx, cached, destination)
}

func (c Cache) missBudget(ctx context.Context, subject envelopes.ID, destination *envelopes.Budget) error {
	if c.Loader == nil {
		return ErrObjectNotFound(subject)
	}

	err := c.Loader.LoadBudget(ctx, subject, destination)
	if err == nil {
		c.lruCache.Put(subject, destination)
	}
	return err
}

// LoadAccounts copies the desired object from the Cache into destination. If the requested option is present in the
// cache, it doesn't invoke Loader. If it is not present, and Loader is not nil, it invokes Loader and adds the result
// to the cache.
func (c Cache) LoadAccounts(ctx context.Context, subject envelopes.ID, destination *envelopes.Accounts) error {
	cached, ok := c.lruCache.Get(subject)
	if !ok {
		return c.missAccounts(ctx, subject, destination)
	}
	return c.hit(ctx, cached, destination)
}

func (c Cache) missAccounts(ctx context.Context, subject envelopes.ID, destination *envelopes.Accounts) error {
	if c.Loader == nil {
		return ErrObjectNotFound(subject)
	}

	err := c.Loader.LoadAccounts(ctx, subject, destination)
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
