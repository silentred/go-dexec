package dexec

import (
	"context"
	"errors"
	"fmt"
	"io"

	types "github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	networktypes "github.com/docker/docker/api/types/network"
	"github.com/moby/moby/pkg/stdcopy"
)

// Execution determines how the command is going to be executed. Currently
// the only method is ByCreatingContainer.
type Execution interface {
	create(d Docker, cmd []string) error
	run(d Docker, stdin io.Reader, stdout, stderr io.Writer) error
	wait(d Docker) (int, error)

	setEnv(env []string) error
	setDir(dir string) error
}

type CreateContainerOption struct {
	ContainerName    string
	Config           *containertypes.Config
	HostConfig       *containertypes.HostConfig
	NetworkingConfig *networktypes.NetworkingConfig
}

type AttachContainerOption struct {
	ContainerID string
	AttachOpt   types.ContainerAttachOptions
}

type createContainer struct {
	opt CreateContainerOption
	cmd []string
	id  string // created container id
	// cw  *docker.Client
	stdin          io.Reader
	stdout, stderr io.Writer
	hr             types.HijackedResponse
}

// ByCreatingContainer is the execution strategy where a new container with specified
// options is created to execute the command.
//
// The container will be created and started with Cmd.Start and will be deleted
// before Cmd.Wait returns.
func ByCreatingContainer(opts CreateContainerOption) (Execution, error) {
	if opts.Config == nil {
		return nil, errors.New("dexec: Config is nil")
	}
	return &createContainer{opt: opts}, nil
}

func (c *createContainer) setEnv(env []string) error {
	if len(c.opt.Config.Env) > 0 {
		return errors.New("dexec: Config.Env already set")
	}
	c.opt.Config.Env = env
	return nil
}

func (c *createContainer) setDir(dir string) error {
	if c.opt.Config.WorkingDir != "" {
		return errors.New("dexec: Config.WorkingDir already set")
	}
	c.opt.Config.WorkingDir = dir
	return nil
}

func (c *createContainer) create(d Docker, cmd []string) error {
	c.cmd = cmd

	if len(c.opt.Config.Cmd) > 0 {
		return errors.New("dexec: Config.Cmd already set")
	}
	if len(c.opt.Config.Entrypoint) > 0 {
		return errors.New("dexec: Config.Entrypoint already set")
	}

	c.opt.Config.AttachStdin = true
	c.opt.Config.AttachStdout = true
	c.opt.Config.AttachStderr = true
	c.opt.Config.OpenStdin = true
	c.opt.Config.StdinOnce = true
	c.opt.Config.Cmd = nil        // clear cmd
	c.opt.Config.Entrypoint = cmd // set new entrypoint

	ctx := context.Background()
	container, err := d.Client.ContainerCreate(ctx, c.opt.Config, c.opt.HostConfig, c.opt.NetworkingConfig, c.opt.ContainerName)
	if err != nil {
		return fmt.Errorf("dexec: failed to create container: %v", err)
	}

	c.id = container.ID
	return nil
}

func (c *createContainer) run(d Docker, stdin io.Reader, stdout, stderr io.Writer) error {
	// fmt.Println("runing...", c.id)
	if c.id == "" {
		return errors.New("dexec: container is not created")
	}
	ctx := context.Background()
	if err := d.Client.ContainerStart(ctx, c.id, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("dexec: failed to start container:  %v", err)
	}

	c.stdin = stdin
	c.stdout = stdout
	c.stderr = stderr

	opts := AttachContainerOption{
		ContainerID: c.id,
		AttachOpt: types.ContainerAttachOptions{
			Stdin:  true,
			Stdout: true,
			Stderr: true,
			Logs:   true,
			Stream: true,
		},
	}

	// opts := docker.AttachToContainerOptions{
	// 	Container:    c.id,
	// 	Stdin:        true,
	// 	Stdout:       true,
	// 	Stderr:       true,
	// 	InputStream:  stdin,
	// 	OutputStream: stdout,
	// 	ErrorStream:  stderr,
	// 	Stream:       true,
	// 	Logs:         true, // include produced output so far
	// }
	// fmt.Println("attach...")

	hijackResp, err := d.Client.ContainerAttach(ctx, opts.ContainerID, opts.AttachOpt)
	if err != nil {
		return fmt.Errorf("dexec: failed to attach container: %v", err)
	}
	c.hr = hijackResp
	// fmt.Println("after attach...")
	return nil
}

func (c *createContainer) wait(d Docker) (exitCode int, err error) {
	// fmt.Println("waiting...", c.id)
	del := func() error {
		return d.ContainerRemove(context.Background(), c.id, types.ContainerRemoveOptions{Force: true})
		// return d.RemoveContainer(docker.RemoveContainerOptions{ID: c.id, Force: true})
	}
	defer del()
	if c.hr.Conn == nil {
		return -1, errors.New("dexec: container is not attached")
	}

	// keep copying stdin to container
	// var quit = make(chan int, 1)
	go func() {
		// fmt.Println("copy stdin to remote conn")
		if c.stdin != nil && c.hr.Conn != nil {
			_, ioErr := io.Copy(c.hr.Conn, c.stdin)
			if ioErr != nil {
				fmt.Println(ioErr)
			}
			c.hr.CloseWrite()
		}
	}()

	if c.hr.Reader != nil {
		_, err = stdcopy.StdCopy(c.stdout, c.stderr, c.hr.Reader)
		if err != nil {
			return -1, fmt.Errorf("dexec: attach error: %v", err)
		}
	}

	var statusCode int64
	statusCode, err = d.Client.ContainerWait(context.Background(), c.id)
	if err != nil {
		return -1, fmt.Errorf("dexec: cannot wait for container: %v", err)
	}

	if err := del(); err != nil {
		return -1, fmt.Errorf("dexec: error deleting container: %v", err)
	}

	exitCode = int(statusCode)
	return exitCode, nil
}
