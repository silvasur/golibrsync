package librsync

/*
#include <stdio.h>
#include <librsync.h>
*/
import "C"

import (
	"io"
	"unsafe"
)

//export patchCallbackGo
func patchCallbackGo(_patcher uintptr, pos C.rs_long_t, len *C.size_t, _buf *unsafe.Pointer) C.rs_result {
	patcher := getPatcher(_patcher)

	patcher.buf = make([]byte, int(*len))
	n, err := patcher.basis.ReadAt(patcher.buf, int64(pos))
	if n < int(*len) {
		if err != io.EOF {
			panic(jobInternalPanic{err})
		} else {
			return C.RS_INPUT_ENDED
		}
	}
	*len = C.size_t(n)
	*_buf = unsafe.Pointer(&(patcher.buf[0]))

	return C.RS_DONE
}
