package dockerless

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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
	containerDIR := filepath.Join(p.Config.TargetDir, "rootfs", workspaceId)

	// check if the rootfs exists
	_, err := os.Stat(containerDIR)
	if err != nil {
		return nil, fmt.Errorf("container %s does not exist", workspaceId)
	}

	// check if the containerDetails exits
	_, err = os.Stat(containerDIR + "/containerDetails")
	if err != nil {
		return nil, fmt.Errorf("container %s does not exist", workspaceId)
	}

	containerDetailsBytes, err := os.ReadFile(containerDIR + "/containerDetails")
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
	pid, err := GetPid(workspaceId)
	if err != nil {
		return err
	}

	return exec.Command("kill", "-9", string(pid)).Run()
}

func (p *DockerlessProvider) Delete(ctx context.Context, workspaceId string) error {
	err := p.Stop(ctx, workspaceId)
	if err != nil {
		return err
	}

	containerDIR := filepath.Join(p.Config.TargetDir, "rootfs", workspaceId)

	return os.RemoveAll(containerDIR)
}
