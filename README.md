# Got

### Fluent dependency injection in Go.

Got provides a declarative way of composing dependencies. It integrates nicely with the common constructor patterns in Go applications.

The main advantage of `got` is removing the need to manually pass references around when instantiating types that depend on one another. 

## Installation

```sh
go get github.com/eriicafes/got
```

## Usage

`got` has a very small API surface. You typically only need to use two functions from this library.

### Create a constructor

Constructor return values are cached the first time they are requested and future calls return the cached value instead of re-running the constructor. See [Transient constructors](#transient-constructors) to opt out of caching.

> Constructors are global variables with their names prefixed with `Get` as a convention.

```go
// printer.go
package main

import "github.com/eriicafes/got"

type Printer interface { 
    Print(string) string
}

var GetPrinter = got.Using(func(c *got.Container) Printer {
    return &CapsPrinter{}
})

// CapsPrinter is an implementation of Printer interface
type CapsPrinter struct{}

func (*CapsPrinter) Print(s string) string { 
    return strings.ToUpper(s)
}
```

### Use in another constructor

Retrieve an instance from a constructor by calling `GetXXX.From(container)`

```go
// office.go
package main

import "github.com/eriicafes/got"

type Office struct {
    Printer Printer
}

var GetOffice = got.Using(func(c *got.Container) *Office {
    return &Office{
        Printer: GetPrinter.From(c),
    }
})
```

### Use in application

Create a container to hold cached instances.

When calling a constructor, any dependencies it has will also be cached, ensuring that shared dependencies use the same instance.

```go
// main.go
package main

import "github.com/eriicafes/got"

func main() {
    c := got.New()

    office := GetOffice.From(c)
    // or using the From function from got
    office := got.From(c, GetOffice)

    office.Printer.Print()
}
```

## Transient constructors
Transient constructors create a new instance each time it is requested.

If you want your constructor to act as a transient, use `GetXXX.New(container)` to opt out of caching its return value.

```go
// office.go
package main

import "github.com/eriicafes/got"

type Office struct {
    Printer Printer
}

var GetOffice = got.Using(func(c *got.Container) *Office {
    return &Office{
        // a new printer is created each time
        Printer: GetPrinter.New(c),
    }
})
```

## Multiple return value constructors

Constructors may return two values, for example an instance and an error. Use `got.Using2` to create such a constructor.

```go
// bad_office.go
package main

import "github.com/eriicafes/got"

var GetBadOffice = got.Using2(func(c *got.Container) (*Office, error) {
    return nil, fmt.Errorf("failed to create office")
})
```

## Mocking

You can mock a constructor using `got.Mock` or `got.Mock2`.

```go
// main.go
package main

import "github.com/eriicafes/got"

type MockPrinter struct{}

var GetMockPrinter = got.Using(func(c *got.Container) Printer {
    return &MockPrinter{}
})

func (*MockPrinter) Print(s string) string {
    return fmt.Sprintf("mocked %s", s)
}

func main() {
    mc := got.New()
    got.Mock(mc, GetPrinter, GetMockPrinter.New(mc))
    office := GetOffice.From(mc)

    office.Printer.Print()
}
```

## Circular dependency errors

Go prevents you from creating circular dependencies as long as you maintain the convention and use global vars as constructors.