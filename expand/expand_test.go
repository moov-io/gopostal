package postal

import (
	"sync"
	"testing"
)

func testExpansionInOutput(t *testing.T, address string, output string, expansions []string) {
	for i := 0; i < len(expansions); i++ {
		if expansions[i] == output {
			return
		}
	}

	t.Error("expansion", output, "not found in expansions for address", address)
}

func testExpansion(t *testing.T, address string, output string) {
	expansions := ExpandAddress(address)
	testExpansionInOutput(t, address, output, expansions)
}

func testExpansionWithOptions(t *testing.T, address string, output string, options ExpandOptions) {
	expansions := ExpandAddressOptions(address, options)

	testExpansionInOutput(t, address, output, expansions)
}

func TestEnglishExpansions(t *testing.T) {
	testExpansion(t, "123 Main St", "123 main street")

	englishOptions := GetDefaultExpansionOptions()
	englishOptions.Languages = []string{"en"}

	testExpansionWithOptions(t, "30 West Twenty-sixth St Fl No. 7", "30 west 26th street floor number 7", englishOptions)
	testExpansionWithOptions(t, "Thirty W 26th St Fl #7", "30 west 26th street floor number 7", englishOptions)

}

func TestMultilingualExpansions(t *testing.T) {
	multilingualOptions := GetDefaultExpansionOptions()
	multilingualOptions.Languages = []string{"en", "fr", "de"}

	testExpansionWithOptions(t, "st", "sankt", multilingualOptions)
	testExpansionWithOptions(t, "st", "saint", multilingualOptions)
}

func TestNonASCIIExpansions(t *testing.T) {
	testExpansion(t, "Friedrichstraße 128, Berlin, Germany", "friedrich strasse 128 berlin germany")
}

// TestConcurrentExpand verifies that concurrent calls to ExpandAddress do not
// corrupt data or return incorrect results. All goroutines must observe
// correct, independent results.
func TestConcurrentExpand(t *testing.T) {
	const goroutines = 100
	const iterations = 50

	address := "123 Main St"
	expectedOutput := "123 main street"

	var wg sync.WaitGroup
	wg.Add(goroutines)

	errCh := make(chan error, goroutines*iterations)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				expansions := ExpandAddress(address)
				found := false
				for _, e := range expansions {
					if e == expectedOutput {
						found = true
						break
					}
				}
				if !found {
					errCh <- &expandMismatchError{address: address, expansions: expansions, want: expectedOutput}
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Error(err)
	}
}

type expandMismatchError struct {
	address    string
	expansions []string
	want       string
}

func (e *expandMismatchError) Error() string {
	return "concurrent expand mismatch"
}

// BenchmarkExpand benchmarks single-threaded address expansion.
func BenchmarkExpand(b *testing.B) {
	address := "123 Main St"

	// Init C library
	ExpandAddress(address)

	for b.Loop() {
		_ = ExpandAddress(address)
	}
}

// BenchmarkExpandParallel benchmarks address expansion under high concurrency.
// This exercises the lock and demonstrates the benefit of RWMutex over Mutex.
func BenchmarkExpandParallel(b *testing.B) {
	address := "123 Main St"
	ExpandAddress(address)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = ExpandAddress(address)
		}
	})
}
