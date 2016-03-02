package dexec

import (
	"io"
)

type Cmd struct {
	// TODO implement
}

func (c *Cmd) CombinedOutput() ([]byte, error)    { return nil, nil }
func (c *Cmd) Output() ([]byte, error)            { return nil, nil }
func (c *Cmd) Run() error                         { return nil }
func (c *Cmd) Start() error                       { return nil }
func (c *Cmd) StderrPipe() (io.ReadCloser, error) { return nil, nil }
func (c *Cmd) StdinPipe() (io.WriteCloser, error) { return nil, nil }
func (c *Cmd) StdoutPipe() (io.ReadCloser, error) { return nil, nil }
func (c *Cmd) Wait() error                        { return nil }
