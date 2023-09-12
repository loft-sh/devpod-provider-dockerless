package dockerless

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/legacy"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/driver"
)

// CreateRootfs will generate a chrootable rootfs from input oci image reference, with input name and config.
// If input image is not found it will be automatically pulled.
// This function will read the oci-image manifest and properly unpack the layers in the right order to generate
// a valid rootfs.
// Untarring process will follow the keep-id option if specified in order to ensure no permission problems.
// Generated config will be saved inside the container's dir. This will NOT be an oci-compatible container config.
func (p *DockerlessProvider) Create(ctx context.Context, workspaceId string, runOptions *driver.RunOptions) error {
	image := runOptions.Image

	ref, err := name.ParseReference(image)
	if err == nil {
		image = ref.Name()
	}

	imageDir := filepath.Join(p.Config.TargetDir, "images", ref.Name())
	containerDIR := filepath.Join(p.Config.TargetDir, "rootfs", workspaceId)
	statusDIR := filepath.Join(p.Config.TargetDir, "status", workspaceId)

	// save the config to file
	configPath := filepath.Join(statusDIR, "runOptions")

	// if the container already exists, exit
	_, err = os.Stat(configPath)
	if err == nil {
		return nil
	}

	err = os.MkdirAll(containerDIR, os.ModePerm)
	if err != nil {
		return err
	}

	err = os.MkdirAll(statusDIR, os.ModePerm)
	if err != nil {
		return err
	}

	// get manifest
	manifestFile, err := os.ReadFile(filepath.Join(imageDir, "manifest.json"))
	if err != nil {
		return err
	}

	var manifest v1.Manifest

	err = json.Unmarshal(manifestFile, &manifest)
	if err != nil {
		return err
	}

	configFile, err := os.ReadFile(filepath.Join(imageDir, "config.json"))
	if err != nil {
		return err
	}

	var layerConfig legacy.LayerConfigFile

	err = json.Unmarshal(configFile, &layerConfig)
	if err != nil {
		return err
	}

	p.Log.Info("preparing container rootfs")

	for index, layer := range manifest.Layers {
		layerDigest := strings.Split(layer.Digest.String(), ":")[1] + ".tar.gz"

		p.Log.Infof("unpacking layer %d of %d", index+1, len(manifest.Layers))

		err = UntarFile(
			filepath.Join(imageDir, layerDigest),
			containerDIR,
		)
		if err != nil {
			return err
		}
	}

	p.Log.Info("done")

	p.Log.Info("preparing runoptions")

	if runOptions.Env == nil {
		runOptions.Env = make(map[string]string)
	}

	// Merge container's default environment with the custom one
	containerEnv := config.ListToObject(layerConfig.Config.Env)
	for k, v := range containerEnv {
		if runOptions.Env[k] == "" {
			runOptions.Env[k] = v
		}
	}

	// fallback TERM to xterm if not defined
	if runOptions.Env["TERM"] == "" {
		runOptions.Env["TERM"] = "xterm"
	}

	// set entrypoint if empty
	if runOptions.Entrypoint == "" {
		runOptions.Entrypoint = layerConfig.Config.Cmd[0]
		runOptions.Cmd = layerConfig.Config.Cmd[1:]
	}

	file, err := json.MarshalIndent(runOptions, "", " ")
	if err != nil {
		return err
	}

	err = os.WriteFile(configPath, file, 0o644)
	if err != nil {
		return err
	}

	containerDetails := initializeContainerDetails(ctx, workspaceId, runOptions)
	detailsPath := filepath.Join(statusDIR, "containerDetails")
	file, err = json.MarshalIndent(containerDetails, "", " ")
	if err != nil {
		return err
	}

	err = os.WriteFile(detailsPath, file, 0o644)
	if err != nil {
		return err
	}

	p.Log.Info("done")

	return nil
}

func initializeContainerDetails(ctx context.Context, workspaceId string, runOptions *driver.RunOptions) *config.ContainerDetails {
	return &config.ContainerDetails{
		ID:      workspaceId,
		Created: time.Now().String(),
		State: config.ContainerDetailsState{
			Status:    "exited",
			StartedAt: "",
		},
		Config: config.ContainerDetailsConfig{
			Labels: config.ListToObject(runOptions.Labels),
		},
	}
}
