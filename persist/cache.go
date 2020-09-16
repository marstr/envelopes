package persist

import (
	"context"
	"fmt"
	"github.com/marstr/collection"
	"github.com/marstr/envelopes"
	"reflect"
)

// ErrTypeMismatch is when an
type ErrTypeMismatch struct {
	Got reflect.Type
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
	lruCache *collection.LRUCache
	Loader
	Writer
}

// NewCache creates a new empty cache of IDers.
func NewCache(capacity uint) *Cache {
	return &Cache{
		lruCache: collection.NewLRUCache(capacity),
	}
}

// Write adds an IDer to this cache. If Writer isn't nil, it is immediately invoked.
func (c Cache) Write(ctx context.Context, subject envelopes.IDer) error {
	// Ensure we cache a pointer, and not a copy of the object.
	switch subject.(type) {
	case envelopes.Transaction:
		cast := subject.(envelopes.Transaction)
		subject = &cast
	case envelopes.State:
		cast := subject.(envelopes.State)
		subject = &cast
	case envelopes.Budget:
		cast := subject.(envelopes.Budget)
		subject = &cast
	case envelopes.Accounts:
		cast := subject.(envelopes.Accounts)
		subject = &cast
	}

	c.lruCache.Put(subject.ID().String(), subject)

	if c.Writer == nil {
		return nil
	}

	return c.Writer.Write(ctx, subject)
}

// Load copies the desired object from the Cache into destination. If the requested option is present in the cache, it
// doesn't invoke Loader. If it is not present, and Loader is not nil, it invokes Loader and adds the result to the
// cache.
func (c Cache) Load(ctx context.Context, subject envelopes.ID, destination envelopes.IDer) error {
	cached, ok := c.lruCache.Get(subject.String())
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
		c.lruCache.Put(subject.String(), destination)
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
		if ! ok {
			return NewErrTypeMismatch(cached, destination)
		}
		*(destination).(*envelopes.Accounts) = *cast
	default:
		return NewErrUnloadableType(destination)
	}

	return nil
}
