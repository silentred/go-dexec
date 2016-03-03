package dexec_test

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/ahmetalpbalkan/dexec"
	"github.com/fsouza/go-dockerclient"
	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&CmdTestSuite{})

// container prefix used in testing
const testPrefix = "dexec_test_"

type CmdTestSuite struct {
	d dexec.Docker
}

func (s *CmdTestSuite) SetUpSuite(c *C) {
	cl, err := docker.NewClient("unix:///var/run/docker.sock")
	c.Assert(err, IsNil)
	s.d = dexec.Docker{cl}
}

func (s *CmdTestSuite) TearDownSuite(c *C) {
	cleanupContainers(c, s.d)
}

func (s *CmdTestSuite) TearDownTest(c *C) {
	cleanupContainers(c, s.d)
}

func cleanupContainers(c *C, cl dexec.Docker) {
	l, err := cl.ListContainers(docker.ListContainersOptions{All: true})
	c.Assert(err, IsNil)
	for _, v := range l {
		for _, n := range v.Names {
			if strings.HasPrefix(strings.TrimPrefix(n, "/"), testPrefix) {
				err = cl.RemoveContainer(docker.RemoveContainerOptions{
					ID:    v.ID,
					Force: true})
				c.Assert(err, IsNil)
				c.Logf("removed container %s", n)
			}
		}
	}
}

func baseContainer(c *C) dexec.Execution {
	e, err := dexec.ByCreatingContainer(docker.CreateContainerOptions{
		Name: fmt.Sprintf("%s%d", testPrefix, rand.Int63()),
		Config: &docker.Config{
			Image: "busybox",
		}})
	c.Assert(err, IsNil)
	return e
}

func (s *CmdTestSuite) TestNewCommand(c *C) {
	cc := baseContainer(c)
	cmd := s.d.Command(cc, "cat", "arg1", "arg2")
	c.Assert(cmd, NotNil)
	c.Assert(cmd.Method, Equals, cc)
	c.Assert(cmd.Path, Equals, "cat")
	c.Assert(cmd.Args, DeepEquals, []string{"arg1", "arg2"})
}

// TODO test errors if dir is set
// TODO test errors if env is set

func (s *CmdTestSuite) TestJustStart(c *C) {
	cmd := s.d.Command(baseContainer(c), "echo", "arg1", "arg2")

	err := cmd.Start()
	c.Assert(err, IsNil)
}

func (s *CmdTestSuite) TestDoubleStart(c *C) {
	cmd := s.d.Command(baseContainer(c), "echo", "arg1", "arg2")

	_ = cmd.Start()
	err := cmd.Start()
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "dexec: already started")
}
