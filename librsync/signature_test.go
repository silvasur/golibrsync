package librsync

import (
	"bytes"
	"github.com/kch42/golibrsync/librsync/testdata"
	"io"
	"testing"
)

func TestSignature(t *testing.T) {
	data := bytes.NewReader(testdata.RandomData())

	buffer := new(bytes.Buffer)
	siggen, err := NewDefaultSignatureGen(data)
	if err != nil {
		t.Fatalf("could not create a signature generator: %s", err)
	}
	defer siggen.Close()

	if _, err = io.Copy(buffer, siggen); err != nil {
		t.Fatalf("Creating the signature failed: %s", err)
	}

	if !bytes.Equal(buffer.Bytes(), testdata.RandomDataSig()) {
		t.Error("Signatures do not match")
	}
}
