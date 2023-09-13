package dockerless

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/loft-sh/devpod-provider-dockerless/pkg/options"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/log"
)

func NewProvider(ctx context.Context, options *options.Options, logs log.Logger) (*DockerlessProvider, error) {
	// create provider
	provider := &DockerlessProvider{
		Config: options,
		Log:    logs,
	}

	return provider, nil
}

type DockerlessProvider struct {
	Config *options.Options
	Log    log.Logger
}

func (p *DockerlessProvider) Find(ctx context.Context, workspaceId string) (*config.ContainerDetails, error) {
	statusDIR := filepath.Join(p.Config.TargetDir, "status", workspaceId)

	// check if the rootfs exists
	_, err := os.Stat(statusDIR)
	if err != nil {
		return nil, fmt.Errorf("container %s does not exist", workspaceId)
	}

	// check if the containerDetails exits
	_, err = os.Stat(statusDIR + "/containerDetails")
	if err != nil {
		return nil, fmt.Errorf("container %s does not exist", workspaceId)
	}

	containerDetailsBytes, err := os.ReadFile(statusDIR + "/containerDetails")
	if err != nil {
		return nil, err
	}

	containerDetails := config.ContainerDetails{}

	err = json.Unmarshal(containerDetailsBytes, &containerDetails)
	if err != nil {
		return nil, err
	}

	status := "stopped"

	pid, err := GetPid(workspaceId)
	if err == nil && pid > 1 {
		// file exists, pid is running
		status = "running"
	}

	containerDetails.State.Status = status

	return &containerDetails, nil
}

func (p *DockerlessProvider) Stop(ctx context.Context, workspaceId string) error {
	p.Log.Infof("stopping: %s", workspaceId)

	pid, err := GetPid(workspaceId)
	if err != nil {
		return err
	}

	p.Log.Debugf("found parent process: %d", pid)

	return exec.Command("kill", "-9", strconv.Itoa(pid)).Run()
}

func (p *DockerlessProvider) Delete(ctx context.Context, workspaceId string) error {
	p.Log.Infof("deleting: %s", workspaceId)

	p.Stop(ctx, workspaceId)

	containerDIR := filepath.Join(p.Config.TargetDir, "rootfs", workspaceId)
	statusDIR := filepath.Join(p.Config.TargetDir, "status", workspaceId)

	err := os.RemoveAll(statusDIR)
	if err != nil {
		return err
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
		"rm", "-rf", containerDIR,
	}...)

	cmd := exec.Command(command, args...)
	return cmd.Run()
}
