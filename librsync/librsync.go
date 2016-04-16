// Package librsync allows you to create binary deltas.
package librsync

/*
#cgo LDFLAGS: -lrsync
#include <stdio.h>
#include <librsync.h>
#include <stdlib.h>

static inline rs_buffers_t* new_rs_buffers() {
	return (rs_buffers_t*) malloc(sizeof(rs_buffers_t));
}

rs_result patchCallback(void* _patcher, rs_long_t pos, size_t* len, void** _buf) {
	return patchCallbackGo(_patcher, pos, len, _buf);
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

// Job holds information about a running librsync operation. The output can be accessed with the Read method.
type Job struct {
	rsbufs *C.rs_buffers_t
	job    *C.rs_job_t

	running bool
	err     error

	inbuf []byte
	in    io.Reader

	outbufTotal []byte
	outbuf      []byte
}

func newJob(input io.Reader) (job *Job, err error) {
	job = new(Job)

	job.in = input
	job.inbuf = make([]byte, inbufSize)
	job.outbufTotal = make([]byte, outbufSize)

	job.rsbufs = C.new_rs_buffers()
	if job.rsbufs == nil {
		return nil, errors.New("Could not allocate memory for rs_buffers_t object")
	}

	job.rsbufs.eof_in = 0
	job.rsbufs.avail_in = 0

	job.running = true

	return
}

// NewDefaultSignatureGen is like NewSignatureGen, but uses default values for blocklen and stronglen.
func NewDefaultSignatureGen(basis io.Reader) (job *Job, err error) {
	job, err = NewSignatureGen(C.RS_DEFAULT_BLOCK_LEN, C.RS_DEFAULT_STRONG_LEN, basis)
	return
}

// NewSignatureGen creates a signature generation job.
// 
// blocklen is the length of a block.
// stronglen is the length of the stong hash.
// basis is an io.Reader that provides data of the basis file.
func NewSignatureGen(blocklen, stronglen uint, basis io.Reader) (job *Job, err error) {
	job, err = newJob(basis)
	if err != nil {
		return
	}

	job.job = C.rs_sig_begin(C.size_t(blocklen), C.size_t(stronglen))
	if job.job == nil {
		job.Close()
		return nil, errors.New("rs_sig_begin failed")
	}

	return
}

// Close will free memory that Go's garbage collector would not be able to free.
func (job *Job) Close() error {
	if job.rsbufs != nil {
		C.free(unsafe.Pointer(job.rsbufs))
		job.rsbufs = nil
	}

	if job.job != nil {
		C.rs_job_free(job.job)
		job.job = nil
	}

	return nil
}

// For errors in callbacks
type jobInternalPanic struct {
	err error
}

func (jp jobInternalPanic) Error() string { return jp.err.Error() }

func jobIter(job *C.rs_job_t, rsbufs *C.rs_buffers_t) (running bool, err error) {
	defer func() {
		r := recover()
		if r == nil {
			// there was no panic
			return
		}
		jp, ok := r.(jobInternalPanic)
		if !ok {
			panic(r)
		}

		running = false
		err = jp.err
	}()

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

// Read reads len(p) or less bytes of the generated output.
func (job *Job) Read(p []byte) (readN int, outerr error) {
	if len(job.outbuf) > 0 {
		if len(job.outbuf) > len(p) {
			readN = len(p)
		} else {
			readN = len(job.outbuf)
		}

		copy(p[:readN], job.outbuf[:readN])
		p = p[:readN]
		job.outbuf = job.outbuf[readN:]
		return
	}

	if !job.running {
		if job.err != nil {
			return 0, job.err
		}

		return 0, io.EOF
	}

	// Fill input buffer
	if (job.rsbufs.avail_in == 0) && (job.rsbufs.eof_in == 0) {
		n, err := job.in.Read(job.inbuf[0:inbufSize])

		switch err {
		case nil:
		case io.EOF:
			job.rsbufs.eof_in = 1
		default:
			outerr = err
			job.err = err
			job.running = false
			return
		}

		job.rsbufs.next_in = (*C.char)(unsafe.Pointer(&(job.inbuf[0])))
		job.rsbufs.avail_in = C.size_t(n)
	}

	job.outbuf = job.outbufTotal
	job.rsbufs.next_out = (*C.char)(unsafe.Pointer(&(job.outbuf[0])))
	job.rsbufs.avail_out = C.size_t(len(job.outbuf))

	var err error
	job.running, err = jobIter(job.job, job.rsbufs)

	outN := int(uintptr(unsafe.Pointer(job.rsbufs.next_out)) - uintptr(unsafe.Pointer(&(job.outbuf[0]))))
	job.outbuf = job.outbuf[:outN]

	if err != nil {
		return outN, err
	}
	return
}

// Signature is an in-memory representation of a signature.
type Signature struct {
	sig *C.rs_signature_t
}

// Close will free memory that Go's garbage collector would not be able to free.
func (s Signature) Close() error {
	if s.sig != nil {
		C.rs_free_sumset(s.sig)
		s.sig = nil
	}
	return nil
}

// LoadSignature loads a signature to memory.
func LoadSignature(input io.Reader) (sig Signature, err error) {
	job, err := newJob(input)
	if err != nil {
		return
	}
	defer job.Close()

	job.job = C.rs_loadsig_begin(&(sig.sig))
	if job.job == nil {
		err = errors.New("rs_loadsig_begin failed")
		return
	}

	if _, err = io.Copy(&nirvana{}, job); err != nil {
		return
	}

	rsret := C.rs_build_hash_table(sig.sig)
	if rsret != C.RS_DONE {
		err = fmt.Errorf("rs_build_hash_table returned %d", rsret)
	}

	return
}

// NewDeltaGen creates a delta generation job.
// 
// sig is the signature loaded by LoadSignature.
// newfile is a reades that provides the new, modified data.
func NewDeltaGen(sig Signature, newfile io.Reader) (job *Job, err error) {
	job, err = newJob(newfile)
	if err != nil {
		return
	}

	job.job = C.rs_delta_begin(sig.sig)
	if job.job == nil {
		job.Close()
		return nil, errors.New("rs_delta_begin failed")
	}

	return
}

// Patcher is a job with additional hidden data for patching.
// 
// IMPORTANT: You still need to Close() this!
type Patcher struct {
	*Job
	basis io.ReaderAt
	buf   []byte
}

var patchCallback = C.patchCallback // So we can use the `&` operator in NewPatcher

// NewPatcher creates a Patcher (which basically is a Job object with some hidden extra data).
// 
// delta is a reader that provides the delta.
// basis provides the basis file.
func NewPatcher(delta io.Reader, basis io.ReaderAt) (job *Patcher, err error) {
	_job, e := newJob(delta)
	if e != nil {
		err = e
		return
	}

	job = &Patcher{
		Job:   _job,
		basis: basis}

	job.job = C.rs_patch_begin((*C.rs_copy_cb)(patchCallback), unsafe.Pointer(job))
	if job.job == nil {
		job.Close()
		return nil, errors.New("rs_patch_begin failed")
	}

	return
}
