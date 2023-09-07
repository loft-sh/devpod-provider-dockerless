package dockerless

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/loft-sh/devpod-provider-dockerless/pkg/options"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/driver"
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

	return &containerDetails, nil
}

func (p *DockerlessProvider) Stop(ctx context.Context, workspaceId string) error {
	pidPath := filepath.Join("/tmp", "dockerless", workspaceId, "child_pid")

	_, err := os.Stat(pidPath)
	if err != nil {
		// does not exist, means it's not running
		return nil
	}

	pid, err := os.ReadFile(pidPath)
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

func (p *DockerlessProvider) ExecuteCommand(ctx context.Context, workspaceId, user, command string, stdin io.Reader, stdout, stderr io.Writer) error {
	runOptions := &driver.RunOptions{}

	err := json.Unmarshal([]byte(os.Getenv("DEVCONTAINER_RUN_OPTIONS")), runOptions)
	if err != nil {
		return fmt.Errorf("unmarshal run options: %w", err)
	}

	pidPath := filepath.Join("/tmp", "dockerless", workspaceId, "child_pid")
	containerDIR := filepath.Join(p.Config.TargetDir, "rootfs", workspaceId)

	_, err = os.Stat(pidPath)
	if err != nil {
		// does not exist, means it's not running
		return fmt.Errorf("container %s is not running", workspaceId)
	}

	pid, err := os.ReadFile(pidPath)
	if err != nil {
		return err
	}

	nsenter := "nsenter"
	args := []string{
		"-m",
		"-u",
		"-i",
		"-p",
		"-U",
		"-r" + containerDIR,
		"-w" + containerDIR,
		"-t",
		string(pid),
		command,
	}

	cmd := exec.Command(nsenter, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	return cmd.Run()
}
