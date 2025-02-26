package got_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/eriicafes/got"
)

type Printer interface{ Print(string) string }

type CapsPrinter struct{}

func (*CapsPrinter) Print(s string) string { return strings.ToUpper(s) }

var GetPrinter = got.Using(func(c *got.Container) Printer {
	return &CapsPrinter{}
})

type Office struct{ Printer Printer }

var GetOffice = got.Using(func(c *got.Container) *Office {
	return &Office{
		Printer: GetPrinter.From(c),
	}
})

var GetBadOffice = got.Using2(func(c *got.Container) (*Office, error) {
	return nil, fmt.Errorf("failed to create office")
})

func TestUsing(t *testing.T) {
	c := got.New()
	office := GetOffice.From(c)

	if office != got.From(c, GetOffice) {
		t.Error("office reference not equal")
	}
	if office.Printer != got.From(c, GetPrinter) {
		t.Error("printer reference not equal")
	}
	got := office.Printer.Print("hello")
	expected := "HELLO"
	if got != expected {
		t.Errorf("invalid printer expected: %q got %q", expected, got)
	}
}

func TestUsing2(t *testing.T) {
	c := got.New()
	office, err := GetBadOffice.From(c)
	office2, err2 := got.From2(c, GetBadOffice)

	if office != office2 {
		t.Error("office reference not equal")
	}
	if err != err2 {
		t.Error("error reference not equal")
	}
}
