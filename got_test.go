package got_test

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/eriicafes/got"
)

type Printer interface {
	Print(string) string
}

type CapsPrinter struct{}

func (*CapsPrinter) Print(s string) string {
	return strings.ToUpper(s)
}

type MockPrinter struct{}

func (*MockPrinter) Print(s string) string {
	return fmt.Sprintf("mocked %s", s)
}

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

type Counter struct {
	count int
}

var GetCounter = got.Using(func(c *got.Container) *Counter {
	return &Counter{count: 0}
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

func TestMockOverwritesCache(t *testing.T) {
	c := got.New()

	// Create real instance via From()
	real := GetPrinter.From(c)
	realOutput := real.Print("test")

	// Verify real instance works
	if realOutput != "TEST" {
		t.Errorf("expected 'TEST', got %q", realOutput)
	}

	// Mock after caching
	var mockPrinter Printer = &MockPrinter{}
	got.Mock(c, GetPrinter, mockPrinter)

	// Subsequent calls should get mock
	result := GetPrinter.From(c)
	if result != mockPrinter {
		t.Error("expected mocked instance after Mock()")
	}

	mockOutput := result.Print("test")
	if mockOutput != "mocked test" {
		t.Errorf("expected 'mocked test', got %q", mockOutput)
	}
}

func TestMockBypassesConstructor(t *testing.T) {
	var constructorCalled bool

	GetTracked := got.Using(func(c *got.Container) *Counter {
		constructorCalled = true
		return &Counter{count: 42}
	})

	c := got.New()

	// Mock before any access
	mockedCounter := &Counter{count: 99}
	got.Mock(c, GetTracked, mockedCounter)

	// Access via From()
	result := GetTracked.From(c)

	// Constructor should never have been called
	if constructorCalled {
		t.Error("constructor was called despite mocking before first access")
	}

	// Should get mocked instance
	if result != mockedCounter {
		t.Error("expected mocked instance")
	}
	if result.count != 99 {
		t.Errorf("expected count 99, got %d", result.count)
	}
}

func TestMock2(t *testing.T) {
	type Database struct{ Host string }

	GetDB := got.Using2(func(c *got.Container) (*Database, error) {
		return &Database{Host: "prod.db"}, nil
	})

	c := got.New()

	// Mock with custom values
	mockDB := &Database{Host: "test.db"}
	mockErr := fmt.Errorf("connection failed")
	got.Mock2(c, GetDB, mockDB, mockErr)

	// Verify mocked values are returned
	db, err := GetDB.From(c)
	if db != mockDB {
		t.Error("expected mocked database instance")
	}
	if err != mockErr {
		t.Error("expected mocked error")
	}
}

func TestConstructorNewDirectCall(t *testing.T) {
	c := got.New()

	// Call .New() directly multiple times
	instance1 := GetCounter.New(c)
	instance2 := GetCounter.New(c)

	// Should create different instances (not cached)
	if instance1 == instance2 {
		t.Error("expected New() to create different instances, got same pointer")
	}

	// From() should return cached instance
	cached1 := GetCounter.From(c)
	cached2 := GetCounter.From(c)
	if cached1 != cached2 {
		t.Error("expected From() to return same cached instance")
	}
}

func TestConstructor2NewDirectCall(t *testing.T) {
	c := got.New()

	// Call .New() directly
	_, err1 := GetBadOffice.New(c)
	_, err2 := GetBadOffice.New(c)

	// Each call creates new error (different pointers)
	if err1 == err2 {
		t.Error("expected different error instances from New()")
	}

	// From() should cache both values
	cachedOffice, cachedErr := GetBadOffice.From(c)
	cachedOffice2, cachedErr2 := got.From2(c, GetBadOffice)

	if cachedOffice != cachedOffice2 || cachedErr != cachedErr2 {
		t.Error("expected From() to return same cached values")
	}
}

func TestDifferentTypeParameters(t *testing.T) {
	c := got.New()

	// Test with int
	GetInt := got.Using(func(c *got.Container) int {
		return 42
	})

	// Test with string
	GetString := got.Using(func(c *got.Container) string {
		return "hello"
	})

	// Test with slice
	GetSlice := got.Using(func(c *got.Container) []byte {
		return []byte{1, 2, 3}
	})

	// Test with map
	GetMap := got.Using(func(c *got.Container) map[string]int {
		return map[string]int{"key": 123}
	})

	// Verify each type works and caches correctly
	int1 := GetInt.From(c)
	int2 := GetInt.From(c)
	if int1 != int2 || int1 != 42 {
		t.Error("int caching failed")
	}

	str1 := GetString.From(c)
	str2 := GetString.From(c)
	if str1 != str2 || str1 != "hello" {
		t.Error("string caching failed")
	}

	// Note: slices and maps compare by reference
	slice1 := GetSlice.From(c)
	slice2 := GetSlice.From(c)
	if &slice1[0] != &slice2[0] {
		t.Error("slice not cached (different backing arrays)")
	}

	map1 := GetMap.From(c)
	map2 := GetMap.From(c)
	// Maps are reference types - modify one and check if both change
	map1["new"] = 999
	if map2["new"] != 999 {
		t.Error("map not cached (different map instances)")
	}
}

func TestNilValues(t *testing.T) {
	type NilStruct struct{ Value string }

	GetNilStruct := got.Using(func(c *got.Container) *NilStruct {
		return nil
	})

	c := got.New()

	// Get nil value
	p1 := GetNilStruct.From(c)
	if p1 != nil {
		t.Error("expected nil, got non-nil")
	}

	// Verify nil is cached (same nil returned)
	p2 := GetNilStruct.From(c)
	if p2 != nil {
		t.Error("expected cached nil")
	}
}

func TestConstructor2SuccessCase(t *testing.T) {
	type Config struct{ Port int }

	GetConfig := got.Using2(func(c *got.Container) (*Config, error) {
		return &Config{Port: 8080}, nil
	})

	c := got.New()

	// Success case
	config, err := GetConfig.From(c)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if config == nil {
		t.Error("expected config, got nil")
	}
	if config.Port != 8080 {
		t.Errorf("expected port 8080, got %d", config.Port)
	}

	// Verify both value and error are cached
	config2, err2 := got.From2(c, GetConfig)
	if config != config2 {
		t.Error("config not cached")
	}
	if err != err2 {
		t.Error("error not cached")
	}
}

func TestConstructor2BothValuesCached(t *testing.T) {
	GetBoth := got.Using2(func(c *got.Container) (string, int) {
		return "data", 42
	})

	c := got.New()

	s1, i1 := GetBoth.From(c)
	s2, i2 := got.From2(c, GetBoth)

	if s1 != s2 || s1 != "data" {
		t.Error("string value not cached correctly")
	}
	if i1 != i2 || i1 != 42 {
		t.Error("int value not cached correctly")
	}
}

func TestZeroValueContainer(t *testing.T) {
	// Zero value should be ready to use
	var c got.Container

	result1 := GetCounter.From(&c)
	result2 := GetCounter.From(&c)

	// Should cache properly
	if result1 != result2 {
		t.Error("zero value container not caching properly")
	}

	if result1.count != 0 {
		t.Errorf("expected count 0, got %d", result1.count)
	}
}

func TestConcurrency(t *testing.T) {
	c := got.New()
	var wg sync.WaitGroup
	numGoroutines := 100

	// Launch multiple goroutines that all try to get the same instance
	instances := make([]*Counter, numGoroutines)
	for i := range numGoroutines {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			instances[idx] = GetCounter.From(c)
		}(i)
	}

	wg.Wait()

	// All instances should be the same (singleton)
	first := instances[0]
	for i := range numGoroutines {
		if instances[i] != first {
			t.Errorf("instance %d is not the same as instance 0", i)
		}
	}
}

func TestConcurrencyMultipleConstructors(t *testing.T) {
	c := got.New()
	var wg sync.WaitGroup
	numGoroutines := 50

	// Launch goroutines accessing different constructors concurrently
	for range numGoroutines {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = GetOffice.From(c)
		}()
		go func() {
			defer wg.Done()
			_ = GetPrinter.From(c)
		}()
	}

	wg.Wait()

	// Verify singletons are maintained
	office1 := GetOffice.From(c)
	office2 := GetOffice.From(c)
	if office1 != office2 {
		t.Error("office instances not equal after concurrent access")
	}
}

func TestNestedDependenciesConcurrency(t *testing.T) {
	var dbCalls, repoCalls, serviceCalls int64

	type DB struct{ ID int64 }
	type Repo struct{ DB *DB }
	type Service struct{ Repo *Repo }

	GetDB := got.Using(func(c *got.Container) *DB {
		id := atomic.AddInt64(&dbCalls, 1)
		return &DB{ID: id}
	})

	GetRepo := got.Using(func(c *got.Container) *Repo {
		atomic.AddInt64(&repoCalls, 1)
		return &Repo{DB: GetDB.From(c)}
	})

	GetService := got.Using(func(c *got.Container) *Service {
		atomic.AddInt64(&serviceCalls, 1)
		return &Service{Repo: GetRepo.From(c)}
	})

	c := got.New()
	var wg sync.WaitGroup
	services := make([]*Service, 50)

	// 50 goroutines all resolve Service
	for i := range 50 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			services[idx] = GetService.From(c)
		}(i)
	}
	wg.Wait()

	// All services should be the same instance
	first := services[0]
	for i := range 50 {
		if services[i] != first {
			t.Errorf("service %d is different instance", i)
		}
	}

	// All should share same Repo and DB
	if services[0].Repo.DB.ID != services[49].Repo.DB.ID {
		t.Error("different DB instances used")
	}

	// Verify singleton behavior - each constructor should be called exactly once
	if dbCalls != 1 {
		t.Errorf("expected exactly 1 DB call, got %d", dbCalls)
	}
	if repoCalls != 1 {
		t.Errorf("expected exactly 1 Repo call, got %d", repoCalls)
	}
	if serviceCalls != 1 {
		t.Errorf("expected exactly 1 Service call, got %d", serviceCalls)
	}
}
