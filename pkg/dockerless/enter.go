package dockerless

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/driver"
)

func (p *DockerlessProvider) Enter(ctx context.Context, workspaceId string) error {
	containerDIR := filepath.Join(p.Config.TargetDir, "rootfs", workspaceId)

	runOptionsBytes, err := os.ReadFile(containerDIR + "/runOptions")
	if err != nil {
		return err
	}

	runOptions := driver.RunOptions{}

	err = json.Unmarshal(runOptionsBytes, &runOptions)
	if err != nil {
		return err
	}

	args := []string{
		"--",
		runOptions.Entrypoint,
	}
	args = append(args, runOptions.Cmd...)

	cmd := exec.Command("/usr/bin/env", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = "/"
	cmd.Env = config.ObjectToList(runOptions.Env)

	err = prepareMounts(containerDIR)
	if err != nil {
		return err
	}

	mounts := []*config.Mount{
		{
			Source: "/etc/resolv.conf",
			Target: "/etc/resolv.conf",
			Type:   "bind",
		},
		{
			Source: "/etc/hosts",
			Target: "/etc/hosts",
			Type:   "bind",
		},
	}
	mount := runOptions.WorkspaceMount

	if mount != nil {
		if mount.Target == "" {
			return fmt.Errorf("workspace mount target is empty")
		}
		mounts = append(mounts, mount)
	}

	mounts = append(mounts, runOptions.Mounts...)
	err = performMounts(mounts, containerDIR)
	if err != nil {
		return err
	}

	// Set Namespaces with generated value
	cmd.SysProcAttr = &syscall.SysProcAttr{
		GidMappingsEnableSetgroups: true,
		Chroot: containerDIR,
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	err = setStatus(containerDIR, "running")
	if err != nil {
		return err
	}

	// block execution till finished
	cmd.Wait()

	return setStatus(containerDIR, "stopped")
}

func setStatus(containerDIR string, status string) error {
	containerDetailsBytes, err := os.ReadFile(containerDIR + "/containerDetails")
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

	err = os.WriteFile(containerDIR+"/containerDetails", file, 0o644)
	if err != nil {
		return err
	}

	return nil
}

func prepareMounts(rootfs string) error {
	err := MountProc(filepath.Join(rootfs, "/proc"))
	if err != nil {
		return err
	}

	err = MountTmpfs(filepath.Join(rootfs, "/tmp"))
	if err != nil {
		return err
	}

	err = MountBind("/dev", filepath.Join(rootfs, "/dev"))
	if err != nil {
		return err
	}

	err = MountShm(filepath.Join(rootfs, "/dev/shm"))
	if err != nil {
		return err
	}

	err = MountMqueue(filepath.Join(rootfs, "/dev/mqueue"))
	if err != nil {
		return err
	}

	err = MountDevPts(filepath.Join(rootfs, "/dev/pts"))
	if err != nil {
		return err
	}

	err = MountBind(filepath.Join(rootfs, "dev/pts/ptmx"), filepath.Join(rootfs, "dev/ptmx"))
	if err != nil {
		return err
	}

	return nil
}

func performMounts(mounts []*config.Mount, rootfs string) error {
	for _, mount := range mounts {
		if mount.Type == "bind" {
			// bind mount
			info, err := os.Stat(mount.Source)
			if err != nil {
				return err
			}

			if info.IsDir() {
				_ = os.MkdirAll(filepath.Join(rootfs, mount.Target), 0o755)
			} else {
				file, _ := os.Create(filepath.Join(rootfs, mount.Target))

				defer func() { _ = file.Close() }()
			}

			return MountBind(mount.Source,
				filepath.Join(rootfs, mount.Target))

		} else {
			return fmt.Errorf("Unsupported mount type '%s' in mount '%s'", mount.Type, mount.String())
		}
	}

	return nil
}
