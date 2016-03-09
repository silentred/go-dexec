# dexec [![Circle CI](https://circleci.com/gh/ahmetalpbalkan/dexec.svg?style=svg&circle-token=8d44d40f5d14602f6d95705d88c3b2c7ecc9bff9)](https://circleci.com/gh/ahmetalpbalkan/dexec)[![GoDoc](https://godoc.org/github.com/ahmetalpbalkan/dexec?status.png)][godoc]

Package dexec provides an interface similar to [`os/exec`][osexec] to run external commands inside containers.

Read godoc at [https://godoc.org/github.com/ahmetalpbalkan/dexec][godoc].

[osexec]: https://godoc.org/os/exec
[godoc]: https://godoc.org/github.com/ahmetalpbalkan/dexec

### Use cases

This utility is intended to provide an execution model that looks and feels
like [`os/exec`][osexec] of Go programming language. Therefore, its semantics
are very similar to `os/exec` package.

You might want to use this library when:

- You want to execute a process, but run it in a container with extra security
  and finer control over resource usage with Docker –and change your code
  minimally.
- You want to execute a piece of work on a remote machine (or even better, a pool
  of machines or a cluster) through Docker. Especially useful to distribute
  computationally expensive workloads.

For such cases, this library abstracts out the details of executing the process
in a container and gives you a cleaner interface you are already familiar with.


### Example

Here is a minimal Go program that runs `echo` in a container:

```go
package main

import (
	"log"

	"github.com/ahmetalpbalkan/dexec"
	"github.com/fsouza/go-dockerclient"
)

func main(){
	cl, _ := docker.NewClient("unix:///var/run/docker.sock")
	d := dexec.Docker{cl}

	m, _ := dexec.ByCreatingContainer(docker.CreateContainerOptions{
	Config: &docker.Config{Image: "busybox"}})

	cmd := d.Command(m, "echo", `I am running inside a container!`)
	b, err := cmd.Output()
	if err != nil { log.Fatal(err) }
	log.Printf("%s", b)
}
```

Output: `I am running inside a container!`
