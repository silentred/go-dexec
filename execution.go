package dexec

import (
	"errors"
	"fmt"
	"io"

	"github.com/fsouza/go-dockerclient"
)

type Execution interface {
	Create(d Docker, cmd []string) error
	Run(d Docker, stdin io.Reader, stdout, stderr io.Writer) error
	Wait() error

	setEnv(env []string) error
	setDir(dir string) error
}

// CreateContainer is a context to launch containers with specified
// options for execution.
type CreateContainer struct {
	docker.CreateContainerOptions

	cmd []string
	id  string // created container id
	cw  docker.CloseWaiter
}

func (c *CreateContainer) setEnv(env []string) error {
	// TODO test if user can provide empty env explicitly just fine.
	if len(c.Config.Env) > 0 {
		return errors.New("dexec: CreateContainer.Config.Env already set")
	}
	c.Config.Env = env
	return nil
}

func (c *CreateContainer) setDir(dir string) error {
	if c.Config.WorkingDir != "" {
		return errors.New("dexec: CreateContainer.Config.WorkingDir already set")
	}
	c.Config.WorkingDir = dir
	return nil
}

func (c *CreateContainer) Create(d Docker, cmd []string) error {
	c.cmd = cmd

	if len(c.Config.Cmd) > 0 {
		return errors.New("dexec: CreateContainer.Config.Cmd already set")
	}
	if len(c.Config.Entrypoint) > 0 {
		return errors.New("dexec: CreateContainer.Config.Entrypoint already set")
	}

	c.Config.AttachStdin = true
	c.Config.AttachStdout = true
	c.Config.AttachStderr = true
	c.Config.Cmd = nil        // clear cmd
	c.Config.Entrypoint = cmd // set new entrypoint

	container, err := d.Client.CreateContainer(c.CreateContainerOptions)
	if err != nil {
		return fmt.Errorf("dexec: failed to create container: %v", err)
	}

	c.id = container.ID
	return nil
}

func (c *CreateContainer) Run(d Docker, stdin io.Reader, stdout, stderr io.Writer) error {
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
	}
	cw, err := d.Client.AttachToContainerNonBlocking(opts)
	if err != nil {
		return fmt.Errorf("dexec: failed to attach container %q: %v", err)
	}
	c.cw = cw
	return nil
}

func (c *CreateContainer) Wait() error {
	if c.cw == nil {
		return errors.New("dexec: container is not attached")
	}
	if err := c.cw.Wait(); err != nil {
		return fmt.Errorf("dexec: attach error: %v", err)
	}
	return nil
}
