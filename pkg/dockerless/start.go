package dockerless

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/driver"
)

func (p *DockerlessProvider) Start(ctx context.Context, workspaceId string) error {
	containerDIR := filepath.Join(p.Config.TargetDir, "rootfs", workspaceId)

	containerDetailsBytes, err := os.ReadFile(containerDIR + "/containerDetails")
	if err != nil {
		return err
	}

	containerDetails := config.ContainerDetails{}

	err = json.Unmarshal(containerDetailsBytes, &containerDetails)
	if err != nil {
		return err
	}

	// return early if the container is already running
	if containerDetails.State.Status == "running" {
		return nil
	}

	runOptionsBytes, err := os.ReadFile(containerDIR + "/runOptions")
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
		return errors.New("unsupported option by the dockerless driver: SecurityOpt")
	}

	if len(runOptions.CapAdd) > 0 {
		return errors.New("unsupported option by the dockerless driver: CapAdd")
	}

	command := "rootlesskit"
	args := []string{
		"--pidns",
		"--cgroupns",
		"--utsns",
		"--ipcns",
		"--reaper",
		"true",
		"--net",
		"host",
		"--state-dir",
		filepath.Join("/tmp", "dockerless", workspaceId),
		os.Args[0],
		"enter",
		base64.StdEncoding.EncodeToString([]byte(workspaceId)),
	}

	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	return cmd.Run()
}