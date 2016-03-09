package dexec_test

import (
	"log"

	"github.com/ahmetalpbalkan/dexec"
	"github.com/fsouza/go-dockerclient"
)

func ExampleCmd_Run() {
	cl, _ := docker.NewClient("unix:///var/run/docker.sock")
	m, _ := dexec.ByCreatingContainer(docker.CreateContainerOptions{
		Config: &docker.Config{Image: "busybox"}})
	d := dexec.Docker{cl}
	cmd := d.Command(m, "sleep", "5")
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Command executed in container.")
}
