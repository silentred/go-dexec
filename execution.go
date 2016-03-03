package dexec

import (
	"errors"
	"fmt"
	"io"

	"github.com/fsouza/go-dockerclient"
)

var (
	ErrEnvSet = errors.New("dexec: createContainer.Config.Env already set")
	ErrDirSet = errors.New("dexec: createContainer.Config.WorkingDir already set")
)

type Execution interface {
	Create(d Docker, cmd []string) error
	Run(d Docker, stdin io.Reader, stdout, stderr io.Writer) error
	Wait() error

	setEnv(env []string) error
	setDir(dir string) error
}

// createContainer is a context to launch containers with specified
// options for execution.
type createContainer struct {
	opt docker.CreateContainerOptions
	cmd []string
	id  string // created container id
	cw  docker.CloseWaiter
}

func ByCreatingContainer(opts docker.CreateContainerOptions) (Execution, error) {
	if opts.Config == nil {
		return nil, errors.New("dexec: Config is nil")
	}
	return &createContainer{opt: opts}, nil
}

func (c *createContainer) setEnv(env []string) error {
	if c.opt.Config == nil {
		return errors.New("dexec: Config not set")
	}

	// TODO test if user can provide empty env explicitly just fine.
	if len(c.opt.Config.Env) > 0 {
		return ErrEnvSet
	}
	c.opt.Config.Env = env
	return nil
}

func (c *createContainer) setDir(dir string) error {
	if c.opt.Config.WorkingDir != "" {
		return ErrDirSet
	}
	c.opt.Config.WorkingDir = dir
	return nil
}

func (c *createContainer) Create(d Docker, cmd []string) error {
	c.cmd = cmd

	if len(c.opt.Config.Cmd) > 0 {
		return errors.New("dexec: createContainer.Config.Cmd already set")
	}
	if len(c.opt.Config.Entrypoint) > 0 {
		return errors.New("dexec: createContainer.Config.Entrypoint already set")
	}

	c.opt.Config.AttachStdin = true
	c.opt.Config.AttachStdout = true
	c.opt.Config.AttachStderr = true
	c.opt.Config.OpenStdin = true
	c.opt.Config.StdinOnce = true
	c.opt.Config.Cmd = nil        // clear cmd
	c.opt.Config.Entrypoint = cmd // set new entrypoint

	container, err := d.Client.CreateContainer(c.opt)
	if err != nil {
		return fmt.Errorf("dexec: failed to create container: %v", err)
	}

	c.id = container.ID
	return nil
}

func (c *createContainer) Run(d Docker, stdin io.Reader, stdout, stderr io.Writer) error {
	if c.id == "" {
		return errors.New("dexec: container is not created")
	}
	if err := d.Client.StartContainer(c.id, nil); err != nil {
		return fmt.Errorf("dexec: failed to start container %q: %v", c.id, err)
	}

	opts := docker.AttachToContainerOptions{
		Container:    c.id,
		Stdin:        true,
		Stdout:       true,
		Stderr:       true,
		InputStream:  stdin,
		OutputStream: stdout,
		ErrorStream:  stderr,
		Stream:       true,
		Logs:         true, // include produced output so far
	}
	cw, err := d.Client.AttachToContainerNonBlocking(opts)
	if err != nil {
		return fmt.Errorf("dexec: failed to attach container %q: %v", err)
	}
	c.cw = cw
	return nil
}

func (c *createContainer) Wait() error {
	if c.cw == nil {
		return errors.New("dexec: container is not attached")
	}
	if err := c.cw.Wait(); err != nil {
		return fmt.Errorf("dexec: attach error: %v", err)
	}
	return nil
}
