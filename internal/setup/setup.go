package setup

/*
#cgo pkg-config: libpostal
#include <libpostal/libpostal.h>
*/
import "C"

import (
	"errors"
	"sync"
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
func Ensure() error {
	setupOnce.Do(func() {
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
