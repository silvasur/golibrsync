package librsync

import (
	"bytes"
	"github.com/kch42/golibrsync/librsync/testdata"
	"testing"
)

func TestHelpers(t *testing.T) {
	basis := bytes.NewReader(testdata.RandomData())
	mutation := bytes.NewReader(testdata.Mutation())

	delta := new(bytes.Buffer)
	if err := InstantDelta(basis, mutation, delta); err != nil {
		t.Fatalf("InstantDelta failed: %s", err)
	}

	if !bytes.Equal(delta.Bytes(), testdata.Delta()) {
		if path, err := dump(delta); err == nil {
			t.Fatalf("Deltas do not match. Generated delta dumped to %s", path)
		} else {
			t.Fatalf("Deltas do not match. Could not dump delta: %s", err)
		}
	}

	newfile := new(bytes.Buffer)
	if err := Patch(basis, delta, newfile); err != nil {
		t.Fatalf("Patch failed: %s", err)
	}

	if !bytes.Equal(newfile.Bytes(), testdata.Mutation()) {
		if path, err := dump(newfile); err == nil {
			t.Fatalf("patch result and mutation are not equal. Result dumped to %s", path)
		} else {
			t.Fatalf("patch result and mutation are not equal. Could not dump result: %s", err)
		}
	}
}
