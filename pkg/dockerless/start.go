package dockerless

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/loft-sh/devpod/pkg/devcontainer/config"
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
		return errors.New("unsupported option by the dockerless driver: SecurityOpt")
	}

	if len(runOptions.CapAdd) > 0 {
		return errors.New("unsupported option by the dockerless driver: CapAdd")
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
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	fmt.Println(cmd.Args)

	err = cmd.Start()
	if err != nil {
		return err
	}

	err = setStatus(statusDIR, "running")
	if err != nil {
		return err
	}

	cmd.Wait()

	return setStatus(statusDIR, "stopped")
}

func setStatus(statusDIR string, status string) error {
	containerDetailsBytes, err := os.ReadFile(statusDIR + "/containerDetails")
	if err != nil {
		return err
	}

	containerDetails := config.ContainerDetails{}

	err = json.Unmarshal(containerDetailsBytes, &containerDetails)
	if err != nil {
		return err
	}

	containerDetails.State.Status = status
	containerDetails.State.StartedAt = time.Now().String()

	file, err := json.MarshalIndent(containerDetails, "", " ")
	if err != nil {
		return err
	}

	err = os.WriteFile(statusDIR+"/containerDetails", file, 0o644)
	if err != nil {
		return err
	}

	return nil
}
