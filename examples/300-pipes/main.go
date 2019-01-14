package main

import (
	"fmt"
	"os"

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

	cmd := d.Command(m, "tr", "[:lower:]", "[:upper:]")
	w, err := cmd.StdinPipe() // <--
	if err != nil {
		panic(err)
	}
	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		panic(err)
	}

	go func() {
		fmt.Fprintln(w, "Hello world") // <--
		fmt.Fprintln(w, "from")        // <--
		fmt.Fprintln(w, "container")   // <--
		w.Close()
	}()

	if err := cmd.Wait(); err != nil {
		panic(err)
	}
	// Output:
	//   HELLO WORLD
	//   FROM
	//   CONTAINER
}
