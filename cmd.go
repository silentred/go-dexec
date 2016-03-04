package dexec

import (
	"bytes"
	"errors"
	"io"

	"github.com/fsouza/go-dockerclient"
)

// Docker contains connection to Docker API.
type Docker struct {
	*docker.Client
}

// Command returns the Cmd struct to execute the named program with given
// arguments using specified execution method.
//
// A new method instance should be used for each new Cmd.
func (d Docker) Command(method Execution, name string, arg ...string) *Cmd {
	return &Cmd{Method: method, Path: name, Args: arg, docker: d}
}

type Cmd struct {
	// Method provides the execution strategy for the context of the Cmd.
	// An instance of Method should not be reused between Cmds.
	Method Execution

	// Path is the path or name of the command in the container.
	Path string

	// Arguments to the command in the container, excluding the command
	// name as the first argument.
	Args []string

	// Env is environment variables to the command. If Env is nil, Run will use
	// Env specified on Method or pre-built container image.
	Env []string

	// Dir specifies the working directory of the command. If Dir is the empty
	// string, Run uses Dir specified on Method or pre-built container image.
	Dir string

	// Stdin specifies the process's standard input.
	// If Stdin is nil, the process reads from the null device (os.DevNull).
	//
	// Run will not close the underlying handle if the Reader is an *os.File
	// differently than os/exec.
	Stdin io.Reader

	// Stdout and Stderr specify the process's standard output and error.
	// If either is nil, they will be redirected to the null device (os.DevNull).
	//
	// Run will not close the underlying handles if they are *os.File differently
	// than os/exec.
	Stdout io.Writer
	Stderr io.Writer

	docker  Docker
	started bool
}

// Start starts the specified command but does not wait for it to complete.
func (c *Cmd) Start() error {
	if c.Dir != "" {
		if err := c.Method.setDir(c.Dir); err != nil {
			return err
		}
	}
	if c.Env != nil {
		if err := c.Method.setEnv(c.Env); err != nil {
			return err
		}
	}

	if c.started {
		return errors.New("dexec: already started")
	}
	c.started = true

	if c.Stdin == nil {
		c.Stdin = empty
	}
	if c.Stdout == nil {
		c.Stdout = discard
	}
	if c.Stderr == nil {
		c.Stderr = discard
	}

	cmd := append([]string{c.Path}, c.Args...)
	if err := c.Method.Create(c.docker, cmd); err != nil {
		return err
	}
	if err := c.Method.Run(c.docker, c.Stdin, c.Stdout, c.Stderr); err != nil {
		return err
	}
	return nil
}

// Wait waits for the command to exit. It must have been started by Start.
//
// If the container exits with a non-zero exit code, the error is of type
// *ExitError. Other error types may be returned for I/O problems and such.
//
// Different than os/exec.Wait, this method will not release any resources
// associated with Cmd (such as file handles).
func (c *Cmd) Wait() error {
	if !c.started {
		return errors.New("dexec: not started")
	}
	ec, err := c.Method.Wait(c.docker)
	if err != nil {
		return err
	}
	if ec != 0 {
		return &ExitError{ExitCode: ec}
	}
	return nil
}

// Run starts the specified command and waits for it to complete.
//
// If the command runs successfully and copying streams are done as expected,
// the error is nil.
//
// If the container exits with a non-zero exit code, the error is of type
// *ExitError. Other error types may be returned for I/O problems and such.
func (c *Cmd) Run() error {
	if err := c.Start(); err != nil {
		return err
	}
	return c.Wait()
}

func (c *Cmd) StderrPipe() (io.ReadCloser, error) { return nil, nil }
func (c *Cmd) StdinPipe() (io.WriteCloser, error) { return nil, nil }
func (c *Cmd) StdoutPipe() (io.ReadCloser, error) { return nil, nil }

func (c *Cmd) CombinedOutput() ([]byte, error) {
	if c.Stdout != nil {
		return nil, errors.New("dexec: Stdout already set")
	}
	if c.Stderr != nil {
		return nil, errors.New("dexec: Stderr already set")
	}
	var b bytes.Buffer
	c.Stdout, c.Stderr = &b, &b
	err := c.Run()
	return b.Bytes(), err
}

// Output runs the command and returns its standard output.
//
// If the container exits with a non-zero exit code, the error is of type
// *ExitError. Other error types may be returned for I/O problems and such.
//
// If c.Stderr was nil, Output populates ExitError.Stderr.
func (c *Cmd) Output() ([]byte, error) {
	if c.Stdout != nil {
		return nil, errors.New("dexec: Stdout already set")
	}
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout

	captureErr := c.Stderr == nil
	if captureErr {
		c.Stderr = &stderr
	}
	err := c.Run()
	if err != nil && captureErr {
		if ee, ok := err.(*ExitError); ok {
			ee.Stderr = stderr.Bytes()
		}
	}
	return stdout.Bytes(), err
}
