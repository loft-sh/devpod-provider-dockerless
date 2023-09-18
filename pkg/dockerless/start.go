package dockerless

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/loft-sh/devpod/pkg/driver"
)

func (p *DockerlessProvider) Start(ctx context.Context, workspaceId string) error {
	statusDIR := filepath.Join(p.Config.TargetDir, "status", workspaceId)

	// return early if the container is already running
	containerDetails, err := p.Find(ctx, workspaceId)
	if err == nil && containerDetails.State.Status == "running" {
		return nil
	}

	p.Log.Debugf("container %s is not running, starting", workspaceId)

	p.Log.Debugf("retrieving runOptions")

	runOptionsBytes, err := os.ReadFile(statusDIR + "/runOptions")
	if err != nil {
		return err
	}

	runOptions := driver.RunOptions{}

	err = json.Unmarshal(runOptionsBytes, &runOptions)
	if err != nil {
		return err
	}

	// fail early for unsupported options
	if len(runOptions.SecurityOpt) > 0 {
		p.Log.Warn("unsupported option by the dockerless driver: SecurityOpt")
	}

	if len(runOptions.CapAdd) > 0 {
		p.Log.Warn("unsupported option by the dockerless driver: CapAdd")
	}

	command := ""
	var args []string

	if os.Getuid() > 0 {
		command = "rootlesskit"
		args = []string{
			"--pidns",
			"--cgroupns",
			"--utsns",
			"--ipcns",
			"--state-dir",
			filepath.Join("/tmp", "dockerless", workspaceId),
		}

		// Default to use slip4netns if we have /dev/net/tun access
		_, err = os.Stat("/dev/net/tun")
		if err == nil {
			args = append(args, []string{
				"--net",
				"slirp4netns",
				"--port-driver",
				"slirp4netns",
				"--disable-host-loopback",
				"--copy-up",
				"/etc",
			}...)
		}
	} else {
		command = "unshare"
		args = []string{
			"-m",
			"-p",
			"-u",
			"-f",
			"--mount-proc",
		}
	}

	args = append(args, []string{
		os.Args[0],
		"enter",
		base64.StdEncoding.EncodeToString([]byte(workspaceId)),
	}...)

	cmd := exec.Command(command, args...)
	cmd.Env = os.Environ()

	p.Log.Infof("starting the container")

	p.Log.Debugf("executing helper command: %s %s", command, strings.Join(args, " "))

	err = cmd.Start()
	if err != nil {
		return err
	}

	return cmd.Process.Release()
}
