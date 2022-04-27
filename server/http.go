package server

import (
	"bytes"
	"errors"
)

type HttpRequestLine struct {
	Method []byte
	URI    []byte
	// Major  int
	// Minor  int
}

// ParseRequestLine ...
func ParseRequestLine(line []byte) (req HttpRequestLine, err error) {
	req.Method, req.URI = bsplit2(line, ' ')

	if req.Method == nil || req.URI == nil {
		return req, errors.New("HttpRequestLine Format error")
	}

	return
}

func bsplit2(bts []byte, sep byte) (b1, b2 []byte) {
	a := bytes.IndexByte(bts, sep)
	b := bytes.IndexByte(bts[a+1:], sep)

	if a == -1 || b == -1 {
		return bts, nil
	}

	b += a + 1

	return bts[:a], bts[a+1 : b]
}
