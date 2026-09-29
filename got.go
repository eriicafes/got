package got

import (
	"context"
	"sync"
	"sync/atomic"
)

// Container is a dependency injection container that caches constructor results.
// It is safe for concurrent use by multiple goroutines.
type Container struct {
	cache   sync.Map
	context context.Context
}

// New creates a new Container.
func New() *Container {
	return NewContext(context.Background())
}

// NewContext creates a new Container with context available to its constructors.
func NewContext(context context.Context) *Container {
	return &Container{context: context}
}

// Context returns the context associated with the container.
func (container *Container) Context() context.Context {
	return container.context
}

// Clear removes all cached constructor results.
func (container *Container) Clear() {
	container.cache.Clear()
}

// ClearErrors removes cached constructor results whose second value is a non-nil error.
func (container *Container) ClearErrors() {
	container.cache.Range(func(key, value any) bool {
		if result, ok := value.(interface{ cachedError() error }); ok && result.cachedError() != nil {
			container.cache.CompareAndDelete(key, value)
		}
		return true
	})
}

// Constructor is implemented by any type that has
// a New method that accepts a container and returns a value,
// and a convenience From method that accepts a container and returns the value from the container.
//
// Use Using to create a new Constructor.
type Constructor[T any] interface {
	New(*Container) T
	From(*Container) T
}

type constructor[T any] struct{ fn func(*Container) T }

func (ct *constructor[T]) New(c *Container) T { return ct.fn(c) }

func (ct *constructor[T]) From(c *Container) T { return From(c, ct) }

// Using creates a new Constructor from a function that accepts a container and returns a value.
func Using[T any](fn func(*Container) T) Constructor[T] {
	return &constructor[T]{fn}
}

// From returns an instance of a constructor's value from the container.
// The constructor's New method is called the first time and the return value is cached.
// Future calls will return the cached value.
func From[T any](c *Container, ct Constructor[T]) T {
	if value, ok := c.cache.Load(ct); ok {
		return value.(*cacheEntry[T]).get()
	}

	var entry *cacheEntry[T]
	entry = newCacheEntry(func() (value T) {
		defer func() {
			if recovered := recover(); recovered != nil {
				c.cache.CompareAndDelete(ct, entry)
				panic(recovered)
			}
		}()
		return ct.New(c)
	})
	actual, loaded := c.cache.LoadOrStore(ct, entry)
	if !loaded {
		return entry.get()
	}
	return actual.(*cacheEntry[T]).get()
}

// Constructor2 is implemented by any type that has
// a New method that accepts a container and returns two values,
// and a convenience From method that accepts a container and returns the values from the container.
//
// Use Using2 to create a new Constructor2.
type Constructor2[T, U any] interface {
	New(*Container) (T, U)
	From(*Container) (T, U)
}

type constructor2[T, U any] struct{ fn func(*Container) (T, U) }

func (ct *constructor2[T, U]) New(c *Container) (T, U) {
	return ct.fn(c)
}

func (ct *constructor2[T, U]) From(c *Container) (T, U) { return From2(c, ct) }

// Using2 creates a new Constructor2 from a function that accepts a container and returns two values.
//
// Use Using2 when a constructor returns two values.
func Using2[T, U any](fn func(*Container) (T, U)) Constructor2[T, U] {
	return &constructor2[T, U]{fn}
}

// TryUsing creates a new Constructor2 from a function that returns a value and an error.
func TryUsing[T any](fn func(*Container) (T, error)) Constructor2[T, error] {
	return Using2(fn)
}

// From2 returns an instance of a constructor's value from the container.
// The constructor's New method is called the first time and the return values are cached.
// Future calls will return the cached values.
func From2[T, U any](c *Container, ct Constructor2[T, U]) (T, U) {
	if value, ok := c.cache.Load(ct); ok {
		f2 := value.(*cacheEntry[from2[T, U]]).get()
		return f2.v1, f2.v2
	}

	var entry *cacheEntry[from2[T, U]]
	entry = newCacheEntry(func() (value from2[T, U]) {
		defer func() {
			if recovered := recover(); recovered != nil {
				c.cache.CompareAndDelete(ct, entry)
				panic(recovered)
			}
		}()
		v1, v2 := ct.New(c)
		return newFrom2(v1, v2)
	})
	actual, loaded := c.cache.LoadOrStore(ct, entry)
	if !loaded {
		f2 := entry.get()
		return f2.v1, f2.v2
	}
	f2 := actual.(*cacheEntry[from2[T, U]]).get()
	return f2.v1, f2.v2
}

type cacheEntry[T any] struct {
	get  func() T
	done atomic.Bool
}

func newCacheEntry[T any](fn func() T) *cacheEntry[T] {
	entry := &cacheEntry[T]{}
	entry.get = sync.OnceValue(func() T {
		defer entry.done.Store(true)
		return fn()
	})
	return entry
}

func (entry *cacheEntry[T]) cachedError() (err error) {
	if !entry.done.Load() {
		return nil
	}
	defer func() { _ = recover() }()
	if value, ok := any(entry.get()).(interface{ cachedError() error }); ok {
		return value.cachedError()
	}
	return nil
}

type from2[T, U any] struct {
	v1  T
	v2  U
	err error
}

func newFrom2[T, U any](v1 T, v2 U) from2[T, U] {
	err, _ := any(v2).(error)
	return from2[T, U]{v1: v1, v2: v2, err: err}
}

func (f from2[T, U]) cachedError() error {
	return f.err
}

// Mock modifies the container cache to return a mocked instance for the constructor.
func Mock[T any](c *Container, ct Constructor[T], v T) {
	c.cache.Store(ct, newCacheEntry(func() T { return v }))
}

// Mock2 modifies the container cache to return a mocked instance for the constructor.
func Mock2[T, U any](c *Container, ct Constructor2[T, U], v1 T, v2 U) {
	value := newFrom2(v1, v2)
	c.cache.Store(ct, newCacheEntry(func() from2[T, U] { return value }))
}
