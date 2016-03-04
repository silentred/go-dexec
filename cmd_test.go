package dexec_test

import (
	"bytes"
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

func testContainer() string { return fmt.Sprintf("%s%d", testPrefix, rand.Int63()) }

type CmdTestSuite struct {
	d dexec.Docker
}

func (s *CmdTestSuite) SetUpSuite(c *C) {
	cl, err := docker.NewClient("unix:///var/run/docker.sock")
	c.Assert(err, IsNil)
	s.d = dexec.Docker{cl}
	cleanupContainers(c, s.d)
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

func baseOpts() docker.CreateContainerOptions {
	return docker.CreateContainerOptions{
		Name: testContainer(),
		Config: &docker.Config{
			Image: "busybox",
		}}
}

func baseContainer(c *C) dexec.Execution {
	e, err := dexec.ByCreatingContainer(baseOpts())
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

func (s *CmdTestSuite) TestConfigNotSet(c *C) {
	opts := baseOpts()
	opts.Config = nil
	_, err := dexec.ByCreatingContainer(opts)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "dexec: Config is nil")
}

func (s *CmdTestSuite) TestDoubleStart(c *C) {
	cmd := s.d.Command(baseContainer(c), "echo")

	_ = cmd.Start()
	err := cmd.Start()
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "dexec: already started")
}

func (s *CmdTestSuite) TestWaitBeforestart(c *C) {
	cmd := s.d.Command(baseContainer(c), "echo")

	err := cmd.Wait()
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "dexec: not started")
}

func (s *CmdTestSuite) TestDirAlreadySet(c *C) {
	opts := baseOpts()
	opts.Config.WorkingDir = "/tmp"
	e, err := dexec.ByCreatingContainer(opts)
	c.Assert(err, IsNil)

	cmd := s.d.Command(e, "pwd")
	cmd.Dir = "/"
	err = cmd.Start()
	c.Assert(err, ErrorMatches, "dexec: Config.WorkingDir already set")
}

func (s *CmdTestSuite) TestEnvAlreadySet(c *C) {
	opts := baseOpts()
	opts.Config.Env = []string{"A=B"}
	e, err := dexec.ByCreatingContainer(opts)
	c.Assert(err, IsNil)

	cmd := s.d.Command(e, "env")
	cmd.Env = []string{"C=D"}
	err = cmd.Start()
	c.Assert(err, ErrorMatches, "dexec: Config.Env already set")
}

func (s *CmdTestSuite) TestEntrypointAlreadySet(c *C) {
	opts := baseOpts()
	opts.Config.Entrypoint = []string{"date"}
	e, err := dexec.ByCreatingContainer(opts)
	c.Assert(err, IsNil)

	cmd := s.d.Command(e, "echo")
	err = cmd.Start()
	c.Assert(err, ErrorMatches, "dexec: Config.Entrypoint already set")
}

func (s *CmdTestSuite) TestCmdAlreadySet(c *C) {
	opts := baseOpts()
	opts.Config.Cmd = []string{"date", "-u"}
	e, err := dexec.ByCreatingContainer(opts)
	c.Assert(err, IsNil)

	cmd := s.d.Command(e, "echo")
	err = cmd.Start()
	c.Assert(err, ErrorMatches, "dexec: Config.Cmd already set")
}

func (s *CmdTestSuite) TestDefaultHandles(c *C) {
	cmd := s.d.Command(baseContainer(c), "echo")
	err := cmd.Start()
	c.Assert(err, IsNil)
	c.Assert(cmd.Stdin, NotNil)
	c.Assert(cmd.Stdout, NotNil)
	c.Assert(cmd.Stderr, NotNil)
}

func (s *CmdTestSuite) TestHandlesPreserved(c *C) {
	stdin := strings.NewReader("foo")
	var b bytes.Buffer
	stdout, stderr := &b, &b

	cmd := s.d.Command(baseContainer(c), "echo", "arg1", "arg2")
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	c.Assert(cmd.Start(), IsNil)
	c.Assert(cmd.Stdin, Equals, stdin)
	c.Assert(cmd.Stdout, Equals, stdout)
	c.Assert(cmd.Stderr, Equals, stderr)
}

func (s *CmdTestSuite) TestNonExisitingCommand(c *C) {
	cmd := s.d.Command(baseContainer(c), "no-such-program")
	err := cmd.Run()
	c.Assert(err, NotNil)
	c.Assert(strings.HasPrefix(err.Error(), `dexec: failed to start container:`), Equals, true)
}

func (s *CmdTestSuite) TestFailedCommandReturnsExitError(c *C) {
	cmd := s.d.Command(baseContainer(c), "false")
	err := cmd.Run()
	c.Assert(err, NotNil)
	c.Assert(err, FitsTypeOf, &dexec.ExitError{})
}

func (s *CmdTestSuite) TestNonZeroExitCodeReturnedInError(c *C) {
	cmd := s.d.Command(baseContainer(c), "sh", "-c", ">&2 echo error; exit 3")
	err := cmd.Run()
	c.Assert(err, NotNil)
	c.Assert(err, FitsTypeOf, &dexec.ExitError{})
	c.Assert(err, ErrorMatches, "dexec: exit status: 3")

	ecErr := err.(*dexec.ExitError)
	c.Assert(ecErr.ExitCode, Equals, 3)
	c.Assert(ecErr.Stderr, IsNil) // Run() shouldn't set ExitError.Stderr
}

func (s *CmdTestSuite) TestRunBasicCommandReadStdout(c *C) {
	cmd := s.d.Command(baseContainer(c), "echo", "arg1", "arg2")
	var b bytes.Buffer
	cmd.Stdout = &b
	cmd.Stderr = &b

	err := cmd.Run()
	c.Assert(err, IsNil)
	c.Assert(string(b.Bytes()), Equals, "arg1 arg2\n")
}

func (s *CmdTestSuite) TestRunBasicCommandWithStdin(c *C) {
	in := `lazy
	fox
jumped`

	var b bytes.Buffer
	cmd := s.d.Command(baseContainer(c), "cat")
	cmd.Stdin = strings.NewReader(in)
	cmd.Stdout, cmd.Stderr = &b, &b

	err := cmd.Run()
	c.Assert(err, IsNil)
	c.Assert(string(b.Bytes()), Equals, in)
}

func (s *CmdTestSuite) TestRunWithDir(c *C) {
	cmd := s.d.Command(baseContainer(c), "pwd")
	cmd.Dir = "/tmp"

	var b bytes.Buffer
	cmd.Stdout, cmd.Stderr = &b, &b
	err := cmd.Run()
	c.Assert(err, IsNil)
	c.Assert(string(b.Bytes()), Equals, cmd.Dir+"\n")
}

func (s *CmdTestSuite) TestRunWithEnv(c *C) {
	cmd := s.d.Command(baseContainer(c), "env")
	cmd.Env = []string{"A=B", "C=D"}

	var b bytes.Buffer
	cmd.Stdout, cmd.Stderr = &b, &b
	err := cmd.Run()
	c.Assert(err, IsNil)

	out := string(b.Bytes())
	c.Logf("Output=%q", out)
	c.Assert(strings.Contains(out, "A=B\n"), Equals, true)
	c.Assert(strings.Contains(out, "C=D\n"), Equals, true)
}

func (s *CmdTestSuite) TestRunStdoutStderrDontMix(c *C) {
	cmd := s.d.Command(baseContainer(c), "sh", "-c", "echo out; >&2 echo err;")
	var outS, errS bytes.Buffer
	cmd.Stdout, cmd.Stderr = &outS, &errS
	c.Assert(cmd.Run(), IsNil)

	c.Assert(string(outS.Bytes()), Equals, "out\n")
	c.Assert(string(errS.Bytes()), Equals, "err\n")
}

func (s *CmdTestSuite) TestCombinedOutputStdoutAlreadySet(c *C) {
	var b bytes.Buffer
	cmd := s.d.Command(baseContainer(c), "env")
	cmd.Stdout = &b
	_, err := cmd.CombinedOutput()
	c.Assert(err, ErrorMatches, "dexec: Stdout already set")
}

func (s *CmdTestSuite) TestCombinedOutputStderrAlreadySet(c *C) {
	var b bytes.Buffer
	cmd := s.d.Command(baseContainer(c), "env")
	cmd.Stderr = &b
	_, err := cmd.CombinedOutput()
	c.Assert(err, ErrorMatches, "dexec: Stderr already set")
}

func (s *CmdTestSuite) TestCombinedOutput(c *C) {
	cmd := s.d.Command(baseContainer(c), "sh", "-c", "echo out; >&2 echo err;")
	b, err := cmd.CombinedOutput()
	c.Assert(err, IsNil)
	c.Assert(string(b), Equals, "out\nerr\n")
}

func (s *CmdTestSuite) TestOutputStdoutAlreadySet(c *C) {
	var b bytes.Buffer
	cmd := s.d.Command(baseContainer(c), "env")
	cmd.Stdout = &b
	_, err := cmd.Output()
	c.Assert(err, ErrorMatches, "dexec: Stdout already set")
}

func (s *CmdTestSuite) TestOutputSuccessfulCommand(c *C) {
	cmd := s.d.Command(baseContainer(c), "sh", "-c", "echo out; >&2 echo err;")
	b, err := cmd.Output()
	c.Assert(err, IsNil)
	c.Assert(string(b), Equals, "out\n")
}

func (s *CmdTestSuite) TestOutputExitErrorStderrNotCollected(c *C) {
	var b bytes.Buffer
	cmd := s.d.Command(baseContainer(c), "sh", "-c", "echo out; >&2 echo err; exit 1")
	cmd.Stderr = &b // set Stderr so that ExitError doesn't collect
	_, err := cmd.Output()
	c.Assert(err, FitsTypeOf, &dexec.ExitError{})
	ee := err.(*dexec.ExitError)
	c.Assert(ee.Stderr, IsNil)
}

func (s *CmdTestSuite) TestOutputExitErrorStderrCollected(c *C) {
	cmd := s.d.Command(baseContainer(c), "sh", "-c", "echo out; >&2 echo err; exit 1")
	b, err := cmd.Output()
	c.Assert(string(b), Equals, "out\n")
	c.Assert(err, FitsTypeOf, &dexec.ExitError{})
	ee := err.(*dexec.ExitError)
	c.Assert(ee.Stderr, NotNil)
}
