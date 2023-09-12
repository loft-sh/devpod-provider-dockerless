package dockerless

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/loft-sh/devpod/pkg/driver"
)

func (p *DockerlessProvider) Start(ctx context.Context, workspaceId string) error {
	statusDIR := filepath.Join(p.Config.TargetDir, "status", workspaceId)

	// return early if the container is already running
	containerDetails, err := p.Find(ctx, workspaceId)
	if err != nil {
		return err
	}

	if containerDetails.State.Status == "running" {
		return nil
	}

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
	args := []string{}

	if os.Getuid() > 0 {
		command = "rootlesskit"
		args = []string{
			"--pidns",
			"--cgroupns",
			"--utsns",
			"--ipcns",
			"--net",
			"host",
			"--state-dir",
			filepath.Join("/tmp", "dockerless", workspaceId),
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

	err = cmd.Start()
	if err != nil {
		return err
	}

	return cmd.Process.Release()
}
