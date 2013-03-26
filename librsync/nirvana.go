package librsync

// nirvana is a io.Writer that will discard all input (like /dev/null)
type nirvana struct{}

func (n *nirvana) Write(p []byte) (int, error) {
	return len(p), nil
}
