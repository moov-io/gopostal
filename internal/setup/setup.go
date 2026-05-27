package setup

/*
#cgo pkg-config: libpostal
#include <libpostal/libpostal.h>
*/
import "C"

import (
	"errors"
	"os"
	"sync"
	"unsafe"
)

var (
	setupOnce sync.Once
	setupErr  error
)

// Ensure initializes the libpostal C library exactly once.
// It is safe for concurrent use and may be called from multiple packages
// (e.g. parser and expand). Subsequent calls after the first successful
// initialization are very cheap.
//
// This covers the base library plus the parser and language classifier
// components. Additional components can be added here in the future.
//
// If the LIBPOSTAL_DATA_DIR environment variable is set, the
// datadir variants of the setup functions are used so that a custom
// (writable) data directory can be used instead of the compile-time default.
func Ensure() error {
	setupOnce.Do(func() {
		datadir := os.Getenv("LIBPOSTAL_DATA_DIR")
		if datadir != "" {
			cDatadir := C.CString(datadir)
			if !bool(C.libpostal_setup_datadir(cDatadir)) {
				setupErr = errors.New("libpostal_setup_datadir failed")
				C.free(unsafe.Pointer(cDatadir))
				return
			}
			C.free(unsafe.Pointer(cDatadir))

			cDatadir = C.CString(datadir)
			if !bool(C.libpostal_setup_parser_datadir(cDatadir)) {
				setupErr = errors.New("libpostal_setup_parser_datadir failed")
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
			return
		}

		if !bool(C.libpostal_setup()) {
			setupErr = errors.New("libpostal_setup failed")
			return
		}
		if !bool(C.libpostal_setup_parser()) {
			setupErr = errors.New("libpostal_setup_parser failed")
			return
		}
		if !bool(C.libpostal_setup_language_classifier()) {
			setupErr = errors.New("libpostal_setup_language_classifier failed")
			return
		}
	})
	return setupErr
}
