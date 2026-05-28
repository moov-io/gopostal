package postal

import (
	"encoding/json"
	"reflect"
	"sync"
	"testing"
)

func testParse(t *testing.T, address string, expectedOutput []ParsedComponent, expectedJSON string) {
	parsedComponents := ParseAddress(address)

	if len(parsedComponents) != len(expectedOutput) || !reflect.DeepEqual(parsedComponents, expectedOutput) {
		t.Error("parsed != expected: ", parsedComponents, "!=", expectedOutput)
	}

	// Test JSON marshaling.
	marshaledJSON, err := json.Marshal(parsedComponents)
	if err != nil {
		t.Error("JSON.marshal error: " + err.Error())
	}

	if string(marshaledJSON) != expectedJSON {
		t.Error("json != expected: ", string(marshaledJSON), "!=", expectedJSON)
	}

	// Test JSON unmarshaling.
	var unmarshaledComponents []ParsedComponent
	if err := json.Unmarshal(marshaledJSON, &unmarshaledComponents); err != nil {
		t.Error("JSON.unmarshal error: " + err.Error())
	}
	if !reflect.DeepEqual(unmarshaledComponents, expectedOutput) {
		t.Error("unmarshaled != expected: ", unmarshaledComponents, "!=", expectedOutput)
	}
}

func TestParseUSAddress(t *testing.T) {
	t.Log("Testing US address")

	testParse(t, "781 Franklin Ave Crown Heights Brooklyn NYC NY 11216 USA",
		[]ParsedComponent{
			{"house_number", "781"},
			{"road", "franklin ave"},
			{"suburb", "crown heights"},
			{"city_district", "brooklyn"},
			{"city", "nyc"},
			{"state", "ny"},
			{"postcode", "11216"},
			{"country", "usa"},
		},
		`[{"label":"house_number","value":"781"},{"label":"road","value":"franklin ave"},{"label":"suburb","value":"crown heights"},{"label":"city_district","value":"brooklyn"},{"label":"city","value":"nyc"},{"label":"state","value":"ny"},{"label":"postcode","value":"11216"},{"label":"country","value":"usa"}]`,
	)
}

// TestConcurrentParse verifies that concurrent calls to ParseAddress do not
// corrupt data or return incorrect results. All goroutines must observe
// correct, independent results.
//
// The underlying parser lock is exclusive because libpostal's address parser
// is not reentrant / thread-safe for concurrent calls.
func TestConcurrentParse(t *testing.T) {
	const goroutines = 100
	const iterations = 50

	address := "781 Franklin Ave Crown Heights Brooklyn NYC NY 11216 USA"
	expected := []ParsedComponent{
		{"house_number", "781"},
		{"road", "franklin ave"},
		{"suburb", "crown heights"},
		{"city_district", "brooklyn"},
		{"city", "nyc"},
		{"state", "ny"},
		{"postcode", "11216"},
		{"country", "usa"},
	}

	var wg sync.WaitGroup
	wg.Add(goroutines)

	errCh := make(chan error, goroutines*iterations)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				components := ParseAddress(address)
				if !reflect.DeepEqual(components, expected) {
					errCh <- &parseMismatchError{got: components, want: expected}
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

type parseMismatchError struct {
	got  []ParsedComponent
	want []ParsedComponent
}

func (e *parseMismatchError) Error() string {
	return "concurrent parse mismatch"
}

// BenchmarkParse benchmarks single-threaded address parsing.
func BenchmarkParse(b *testing.B) {
	address := "781 Franklin Ave Crown Heights Brooklyn NYC NY 11216 USA"

	// Init C library
	ParseAddress(address)

	for b.Loop() {
		_ = ParseAddress(address)
	}
}

// BenchmarkParseParallel benchmarks address parsing under high concurrency.
// Note: the parser uses an exclusive lock (not RWMutex) because libpostal's
// address parser (CRF + feature extraction) is not safe for concurrent calls.
func BenchmarkParseParallel(b *testing.B) {
	address := "781 Franklin Ave Crown Heights Brooklyn NYC NY 11216 USA"
	ParseAddress(address)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = ParseAddress(address)
		}
	})
}
