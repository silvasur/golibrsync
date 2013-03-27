package librsync

import (
	"fmt"
	"io"
	"os"
)

// Some functions to support testing

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
