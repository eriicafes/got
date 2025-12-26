package got

import "sync"

// Container is a dependency injection container that caches constructor results.
// It is safe for concurrent use by multiple goroutines.
//
// The zero Container is empty and ready for use.
type Container struct {
	cache sync.Map
}

// New creates a new Container.
// While the zero value of Container is ready to use, New() is provided for API clarity.
func New() *Container {
	return &Container{}
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
	if v, ok := c.cache.Load(ct); ok {
		return v.(T)
	}
	v := ct.New(c)
	actual, loaded := c.cache.LoadOrStore(ct, v)
	if loaded {
		return actual.(T)
	}
	return v
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
// Use Using2 when a constructor returns multiple values for example an instance and an error.
func Using2[T, U any](fn func(*Container) (T, U)) Constructor2[T, U] {
	return &constructor2[T, U]{fn}
}

// From2 returns an instance of a constructor's value from the container.
// The constructor's New method is called the first time and the return values are cached.
// Future calls will return the cached values.
func From2[T, U any](c *Container, ct Constructor2[T, U]) (T, U) {
	if v, ok := c.cache.Load(ct); ok {
		f2 := v.(from2[T, U])
		return f2.v1, f2.v2
	}
	v1, v2 := ct.New(c)
	val := from2[T, U]{v1, v2}
	actual, loaded := c.cache.LoadOrStore(ct, val)
	if loaded {
		f2 := actual.(from2[T, U])
		return f2.v1, f2.v2
	}
	return v1, v2
}

type from2[T, U any] struct {
	v1 T
	v2 U
}

// Mock modifies the container cache to return a mocked instance for the constructor.
func Mock[T any](c *Container, ct Constructor[T], v T) {
	c.cache.Store(ct, v)
}

// Mock2 modifies the container cache to return a mocked instance for the constructor.
func Mock2[T, U any](c *Container, ct Constructor2[T, U], v1 T, v2 U) {
	c.cache.Store(ct, from2[T, U]{v1, v2})
}
