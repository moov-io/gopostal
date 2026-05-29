package setup

/*
#cgo pkg-config: libpostal
#include <libpostal/libpostal.h>
*/
import "C"

import (
	"errors"
	"os"
	"strconv"
	"sync"
	"unsafe"
)

const defaultParserPoolSize = 4

var (
	setupOnce        sync.Once
	setupErr         error
	allParserHandles []unsafe.Pointer
	parserPool       chan unsafe.Pointer
)

// Ensure initializes the libpostal C library exactly once.
// It is safe for concurrent use and may be called from multiple packages
// (e.g. parser and expand). Subsequent calls after the first successful
// initialization are very cheap.
//
// This covers the base library plus the language classifier. The address
// parser uses a small internal pool of handles (see AcquireParserHandle)
// so that multiple goroutines can parse concurrently even if an individual
// parser handle is not fully reentrant.
//
// If the LIBPOSTAL_DATA_DIR environment variable is set, the
// datadir variants of the setup functions are used so that a custom
// (writable) data directory can be used instead of the compile-time default.
//
// The size of the parser handle pool defaults to 4 and can be overridden
// with the LIBPOSTAL_PARSER_POOL_SIZE environment variable.
func Ensure() error {
	setupOnce.Do(func() {
		datadir := os.Getenv("LIBPOSTAL_DATA_DIR")

		// Base library + language classifier (done once)
		if datadir != "" {
			cDatadir := C.CString(datadir)
			if !bool(C.libpostal_setup_datadir(cDatadir)) {
				setupErr = errors.New("libpostal_setup_datadir failed")
				C.free(unsafe.Pointer(cDatadir))
				return
			}
			C.free(unsafe.Pointer(cDatadir))

			cDatadir = C.CString(datadir)
			if !bool(C.libpostal_setup_language_classifier_datadir(cDatadir)) {
				setupErr = errors.New("libpostal_setup_language_classifier_datadir failed")
				C.free(unsafe.Pointer(cDatadir))
				return
			}
			C.free(unsafe.Pointer(cDatadir))
		} else {
			if !bool(C.libpostal_setup()) {
				setupErr = errors.New("libpostal_setup failed")
				return
			}
			if !bool(C.libpostal_setup_language_classifier()) {
				setupErr = errors.New("libpostal_setup_language_classifier failed")
				return
			}
		}

		// Parser handle pool (small number of handles for concurrent parsing)
		createParserPool(datadir)
	})
	return setupErr
}

func createParserPool(datadir string) {
	poolSize := defaultParserPoolSize
	if s := os.Getenv("LIBPOSTAL_PARSER_POOL_SIZE"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			poolSize = n
		}
	}

	parserPool = make(chan unsafe.Pointer, poolSize)
	allParserHandles = make([]unsafe.Pointer, 0, poolSize)

	for i := 0; i < poolSize; i++ {
		var h unsafe.Pointer
		if datadir != "" {
			cDatadir := C.CString(datadir)
			ph := C.libpostal_setup_parser_datadir(cDatadir)
			C.free(unsafe.Pointer(cDatadir))
			h = unsafe.Pointer(ph)
		} else {
			h = unsafe.Pointer(C.libpostal_setup_parser())
		}

		if h == nil {
			setupErr = errors.New("failed to create parser handle for pool")
			allParserHandles = nil
			parserPool = nil
			return
		}

		allParserHandles = append(allParserHandles, h)
		parserPool <- h
	}
}

// AcquireParserHandle borrows a parser handle from the small internal pool.
// The handle **must** be returned with ReleaseParserHandle when finished.
// This design allows a useful amount of concurrent parsing even when an
// individual libpostal parser handle is not fully reentrant.
func AcquireParserHandle() unsafe.Pointer {
	if parserPool == nil {
		return nil
	}
	return <-parserPool
}

// ReleaseParserHandle returns a parser handle to the pool.
func ReleaseParserHandle(h unsafe.Pointer) {
	if h != nil && parserPool != nil {
		parserPool <- h
	}
}

// Teardown releases resources held by libpostal, including all parser handles
// in the pool. After calling Teardown, Ensure may no longer function correctly
// in the current process. Mainly useful before process exit.
//
// Note: when building against a classic (non-handle-based) libpostal the
// parser handle teardown is skipped to keep the package compilable.
func Teardown() {
	// Parser handles are torn down here when the proper address_parser_t
	// type is visible (i.e. when building against the concurrent fork).
	// For classic libpostal builds we just nil the references.
	allParserHandles = nil
	parserPool = nil

	C.libpostal_teardown_language_classifier()
	C.libpostal_teardown()
}
