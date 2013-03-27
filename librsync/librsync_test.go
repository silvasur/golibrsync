package librsync

import (
	"bytes"
	"fmt"
	"github.com/kch42/golibrsync/librsync/testdata"
	"io"
	"os"
	"testing"
)

func dump(r io.Reader) (string, error) {
	path := fmt.Sprintf("%s%cgolibrsync_test", os.TempDir(), os.PathSeparator)
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err = io.Copy(file, r); err != nil {
		return "", err
	}

	return path, nil
}

func TestSignatureDeltaPatch(t *testing.T) {
	// Generate signature
	orig := bytes.NewReader(testdata.RandomData())

	sigbuf := new(bytes.Buffer)
	siggen, err := NewDefaultSignatureGen(orig)
	if err != nil {
		t.Fatalf("could not create a signature generator: %s", err)
	}
	defer siggen.Close()

	if _, err = io.Copy(sigbuf, siggen); err != nil {
		t.Fatalf("Creating the signature failed: %s", err)
	}

	if !bytes.Equal(sigbuf.Bytes(), testdata.RandomDataSig()) {
		if path, err := dump(sigbuf); err == nil {
			t.Fatalf("Signatures do not match. Generated signature dumped to %s", path)
		} else {
			t.Fatalf("Signatures do not match. Could not dump signature: %s", err)
		}
	}

	// Loading signature
	sig, err := LoadSignature(sigbuf)
	if err != nil {
		t.Fatalf("Loading signature failed: %s", err)
	}
	defer sig.Close()

	// Generate delta
	mutation := bytes.NewReader(testdata.Mutation())

	deltabuf := new(bytes.Buffer)
	deltagen, err := NewDeltaGen(sig, mutation)
	if err != nil {
		t.Fatalf("could not create a delta generator: %s", err)
	}
	defer deltagen.Close()

	if _, err = io.Copy(deltabuf, deltagen); err != nil {
		t.Fatalf("Creating the delta failed: %s", err)
	}

	if !bytes.Equal(deltabuf.Bytes(), testdata.Delta()) {
		if path, err := dump(deltabuf); err == nil {
			t.Fatalf("deltas do not match. Generated delta dumped to %s", path)
		} else {
			t.Fatalf("deltas do not match. Could not dump delta: %s", err)
		}
	}

	// Apply Patch 
	patchres := new(bytes.Buffer)
	patcher, err := NewPatcher(deltabuf, orig)
	if err != nil {
		t.Fatalf("could not create a patcher: %s", err)
	}
	defer patcher.Close()

	if _, err = io.Copy(patchres, patcher); err != nil {
		t.Fatalf("Applying the patch failed: %s", err)
	}

	if !bytes.Equal(patchres.Bytes(), testdata.Mutation()) {
		if path, err := dump(patchres); err == nil {
			t.Fatalf("patch result and mutation are not equal. Result dumped to %s", path)
		} else {
			t.Fatalf("patch result and mutation are not equal. Could not dump result: %s", err)
		}
	}
}
