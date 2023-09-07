package dockerless

import (
	"context"
	"io"

	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod-provider-dockerless/pkg/options"
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
	return nil, nil
}

func (p *DockerlessProvider) Start(ctx context.Context, workspaceId string) error {
	return nil
}

func (p *DockerlessProvider) Stop(ctx context.Context, workspaceId string) error {
	return nil
}

func (p *DockerlessProvider) Delete(ctx context.Context, workspaceId string) error {
	return nil
}

func (p *DockerlessProvider) Create(ctx context.Context, workspaceId string, runOptions *driver.RunOptions) error {
	return nil
}

func (p *DockerlessProvider) Pull(ctx context.Context, runOptions *driver.RunOptions) error {
	return nil
}

func (p *DockerlessProvider) ExecuteCommand(ctx context.Context, workspaceId, user, command string, stdin io.Reader, stdout, stderr io.Writer) error {
	return nil
}
