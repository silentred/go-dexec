package main

import (
	"fmt"

	containertypes "github.com/docker/docker/api/types/container"
	docker "github.com/docker/docker/client"
	dexec "github.com/silentred/go-dexec"
)

func main() {
	cl, _ := docker.NewEnvClient()
	d := dexec.Docker{cl}

	m, _ := dexec.ByCreatingContainer(dexec.CreateContainerOption{
		Config: &containertypes.Config{Image: "busybox"}},
	)
	cmd := d.Command(m, "echo", `I am running inside a container!`)
	b, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s", b)
}
