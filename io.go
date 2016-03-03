package dexec

import (
	"io"
	"io/ioutil"
)

type emptyReader struct{}

func (_ emptyReader) Read(_ []byte) (int, error) { return 0, io.EOF }

var (
	discard io.Writer = ioutil.Discard
	empty   io.Reader = emptyReader{}
)
