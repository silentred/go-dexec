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

	cmd := d.Command(m, "sh", "-c", "exit 255;")
	err := cmd.Run()
	if err == nil {
		panic("not expecting successful exit")
	}

	if ee, ok := err.(*dexec.ExitError); ok {
		fmt.Printf("exit code=%d\n", ee.ExitCode) // <--
	} else {
		panic(err)
	}
}
