package main

import "github.com/eriicafes/got"

type A struct{ *B }
type B struct{ *A }

var GetA got.Constructor[*A]
var GetB got.Constructor[*B]

func init() {
	GetA = got.Using(func(container *got.Container) *A {
		return &A{B: GetB.From(container)}
	})

	GetB = got.Using(func(container *got.Container) *B {
		return &B{A: GetA.From(container)}
	})
}

func main() {
	GetA.From(got.New())
}
