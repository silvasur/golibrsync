package librsync

import (
	"io"
)

// Some helper functions to make things more convenient.

// CreateSignature wraps around a signature generation job and copies the result to the signature writer.
func CreateSignature(basis io.Reader, signature io.Writer) error {
	siggen, err := NewDefaultSignatureGen(basis)
	if err != nil {
		return err
	}
	defer siggen.Close()

	_, err = io.Copy(signature, siggen)
	return err
}

// CreateDelta wraps around a delta generation job and copies the result to the delta writer.
func CreateDelta(signature, newfile io.Reader, delta io.Writer) error {
	sig, err := LoadSignature(signature)
	if err != nil {
		return err
	}
	defer sig.Close()

	deltagen, err := NewDeltaGen(sig, newfile)
	if err != nil {
		return err
	}
	defer deltagen.Close()

	_, err = io.Copy(delta, deltagen)
	return err
}

// InstantDelta creates a delta file without the extra step of creating a signature.
func InstantDelta(basis, newfile io.Reader, delta io.Writer) error {
	siggen, err := NewDefaultSignatureGen(basis)
	if err != nil {
		return err
	}
	defer siggen.Close()

	sig, err := LoadSignature(siggen)
	if err != nil {
		return err
	}
	defer sig.Close()

	deltagen, err := NewDeltaGen(sig, newfile)
	if err != nil {
		return err
	}
	defer deltagen.Close()

	_, err = io.Copy(delta, deltagen)
	return err
}

// Patch wraps around a Patcher job and copies the result to newfile.
func Patch(basis io.ReaderAt, delta io.Reader, newfile io.Writer) error {
	patcher, err := NewPatcher(delta, basis)
	if err != nil {
		return err
	}
	defer patcher.Close()

	_, err = io.Copy(newfile, patcher)
	return err
}
