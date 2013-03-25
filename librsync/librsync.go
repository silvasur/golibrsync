// Package librsync allows you to create binary deltas.
package librsync

/*
#cgo LDFLAGS: -lrsync
#include <stdio.h>
#include <librsync.h>
#include <stdlib.h>

rs_buffers_t* new_rs_buffers() {
	return (rs_buffers_t*) malloc(sizeof(rs_buffers_t));
}
*/
import "C"

import (
	"errors"
	"fmt"
	"io"
	"unsafe"
)

const (
	inbufSize  = 16 * 1024
	outbufSize = 16 * 1024
)

var (
	ErrInputEnded = errors.New("Input ended (possibly unexpected)")
	ErrBadMagic   = errors.New("Bad magic number. Probably not an librsync file.")
	ErrCorrupt    = errors.New("Input stream corrupted")
	ErrInternal   = errors.New("Internal error (library bug?)")
)

// SignatureGen holds information to generate a librsync file signature.
// It must be constructed with the NewSignatureGen or NewDefaultSignatureGen functions.
type SignatureGen struct {
	rsbufs *C.rs_buffers_t
	job    *C.rs_job_t

	running bool
	err     error

	inbuf []byte
	in    io.Reader

	outbufTotal []byte
	outbuf      []byte

	blocklen, stronglen uint
}

// NewDefaultSignatureGen is like NewSignatureGen, but uses default values for blocklen and stronglen.
func NewDefaultSignatureGen(input io.Reader) (siggen *SignatureGen, err error) {
	siggen, err = NewSignatureGen(C.RS_DEFAULT_BLOCK_LEN, C.RS_DEFAULT_STRONG_LEN, input)
	return
}

// NewSignatureGen creates a new SignatureGen instance.
// 
// blocklen is the length of a block.
// stronglen is the length of the stong hash.
// input is an io.Reader that provides the input data.
func NewSignatureGen(blocklen, stronglen uint, input io.Reader) (siggen *SignatureGen, err error) {
	siggen = new(SignatureGen)

	siggen.blocklen, siggen.stronglen = blocklen, stronglen
	siggen.in = input
	siggen.inbuf = make([]byte, inbufSize)
	siggen.outbufTotal = make([]byte, outbufSize)

	siggen.rsbufs = C.new_rs_buffers()
	if siggen.rsbufs == nil {
		return nil, fmt.Errorf("Could not allocate memory for rs_buffers_t object")
	}

	siggen.job = C.rs_sig_begin(C.size_t(blocklen), C.size_t(stronglen))
	if siggen.job == nil {
		siggen.Close()
		return nil, fmt.Errorf("rs_sig_begin failed")
	}

	siggen.running = true
	return
}

// Close will free memory that Go's garbage collector would not be able to free.
func (siggen *SignatureGen) Close() error {
	C.free(unsafe.Pointer(siggen.rsbufs))
	C.rs_job_free(siggen.job)

	return nil
}

func jobIter(job *C.rs_job_t, rsbufs *C.rs_buffers_t) (running bool, err error) {
	switch res := C.rs_job_iter(job, rsbufs); res {
	case C.RS_DONE:
	case C.RS_BLOCKED:
		running = true
	case C.RS_INPUT_ENDED:
		err = ErrInputEnded
	case C.RS_BAD_MAGIC:
		err = ErrBadMagic
	case C.RS_CORRUPT:
		err = ErrCorrupt
	case C.RS_INTERNAL_ERROR:
		err = ErrInternal
	default:
		err = fmt.Errorf("Unexpected result from library: %d", res)
	}
	return
}

// Read reads len(p) or less bytes of the generated signature.
func (siggen *SignatureGen) Read(p []byte) (readN int, outerr error) {
	if len(siggen.outbuf) > 0 {
		if len(siggen.outbuf) > len(p) {
			readN = len(p)
		} else {
			readN = len(siggen.outbuf)
		}

		copy(p[:readN], siggen.outbuf[:readN])
		p = p[:readN]
		siggen.outbuf = siggen.outbuf[readN:]
		return
	}

	if !siggen.running {
		if siggen.err != nil {
			return 0, siggen.err
		}

		return 0, io.EOF
	}

	// Fill input buffer
	if (siggen.rsbufs.avail_in == 0) && (siggen.rsbufs.eof_in == 0) {
		n, err := siggen.in.Read(siggen.inbuf[0:inbufSize])

		switch err {
		case nil:
		case io.EOF:
			siggen.rsbufs.eof_in = 1
		default:
			outerr = err
			siggen.err = err
			siggen.running = false
			return
		}

		siggen.rsbufs.next_in = (*C.char)(unsafe.Pointer(&(siggen.inbuf[0])))
		siggen.rsbufs.avail_in = C.size_t(n)
	}

	siggen.outbuf = siggen.outbufTotal
	siggen.rsbufs.next_out = (*C.char)(unsafe.Pointer(&(siggen.outbuf[0])))
	siggen.rsbufs.avail_out = C.size_t(len(siggen.outbuf))

	var err error
	siggen.running, err = jobIter(siggen.job, siggen.rsbufs)

	outN := int(uintptr(unsafe.Pointer(siggen.rsbufs.next_out)) - uintptr(unsafe.Pointer(&(siggen.outbuf[0]))))
	siggen.outbuf = siggen.outbuf[:outN]

	if err != nil {
		return outN, err
	}
	return
}
