package main

import (
	"os"
	"strings"

	containertypes "github.com/docker/docker/api/types/container"
	docker "github.com/docker/docker/client"
	dexec "github.com/silentred/go-dexec"
)

func main() {
	input := `Hello world
from
container
container
container
container
container
container
container
asdf
asdf
asdf
asdf
asdf
asdf
asdf
`

	cl, _ := docker.NewEnvClient()
	d := dexec.Docker{cl}

	m, _ := dexec.ByCreatingContainer(dexec.CreateContainerOption{
		Config: &containertypes.Config{Image: "busybox"}},
	)

	cmd := d.Command(m, "tr", "[:lower:]", "[:upper:]")
	cmd.Stdin = strings.NewReader(input) // <--
	cmd.Stdout = os.Stdout               // <--

	if err := cmd.Run(); err != nil {
		panic(err)
	}
	// Output:
	//   HELLO WORLD
	//   FROM
	//   CONTAINER
}
