package cmd

import (
	"context"

	"github.com/loft-sh/devpod-provider-dockerless/pkg/dockerless"
	"github.com/loft-sh/devpod-provider-dockerless/pkg/options"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// EnterCmd holds the cmd flags
type EnterCmd struct{
}

// NewEnterCmd defines a command
func NewEnterCmd() *cobra.Command {
	cmd := &EnterCmd{}
	enterCmd := &cobra.Command{
		Use:   "enter",
		Short: "Enter a container",
		RunE: func(_ *cobra.Command, args []string) error {
			options, err := options.FromEnv()
			if err != nil {
				return err
			}

			return cmd.Run(context.Background(), options, log.Default)
		},
	}

	return enterCmd
}

// Run runs the command logic
func (cmd *EnterCmd) Run(ctx context.Context, options *options.Options, log log.Logger) error {
	dockerlessProvider, err := dockerless.NewProvider(ctx, options, log)
	if err != nil {
		return err
	}

	return dockerlessProvider.Enter(ctx, options.DevContainerID)
}
