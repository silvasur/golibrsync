package librsync

/*
#include <stdio.h>
#include <librsync.h>
#include <stdlib.h>
*/
import "C"

import (
	"io"
	"unsafe"
)

//export patchCallbackGo
func patchCallbackGo(_patcher uintptr, pos C.rs_long_t, buflen *C.size_t, buf *unsafe.Pointer) C.rs_result {
	patcher := getPatcher(_patcher)

	if patcher.buf != nil {
		C.free(patcher.buf)
	}
	patcher.buf = C.malloc(*buflen)
	// https://github.com/golang/go/wiki/cgo#turning-c-arrays-into-go-slices
	s := (*[1 << 30]byte)(patcher.buf)[:*buflen:*buflen]
	n, err := patcher.basis.ReadAt(s, int64(pos))
	if n < int(*buflen) {
		if err != io.EOF {
			panic(jobInternalPanic{err})
		} else {
			return C.RS_INPUT_ENDED
		}
	}
	*buflen = C.size_t(n)
	*buf = patcher.buf

	return C.RS_DONE
}
