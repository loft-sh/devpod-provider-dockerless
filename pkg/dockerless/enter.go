package dockerless

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/driver"
)

func (p *DockerlessProvider) Enter(ctx context.Context, workspaceId string) error {
	containerDIR := filepath.Join(p.Config.TargetDir, "rootfs", workspaceId)
	statusDIR := filepath.Join(p.Config.TargetDir, "status", workspaceId)

	runOptionsBytes, err := os.ReadFile(statusDIR + "/runOptions")
	if err != nil {
		return err
	}

	runOptions := driver.RunOptions{}

	err = json.Unmarshal(runOptionsBytes, &runOptions)
	if err != nil {
		return err
	}

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

	err = syscall.Chdir(containerDIR)
	if err != nil {
		return err
	}

	// then we set up the hostname.
	err = syscall.Sethostname([]byte(workspaceId))
	if err != nil {
		return fmt.Errorf("error setting hostname for namespace: %w", err)
	}

	cmd := exec.Command(runOptions.Entrypoint, runOptions.Cmd...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Chroot: containerDIR,
	}
	cmd.Env = config.ObjectToList(runOptions.Env)

	return cmd.Run()
}

func prepareMounts(rootfs string) error {
	err := MountBind("/proc", filepath.Join(rootfs, "/proc"))
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

			err = MountBind(mount.Source,
				filepath.Join(rootfs, mount.Target))

			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("unsupported mount type '%s' in mount '%s'", mount.Type, mount.String())
		}
	}

	return nil
}

// PivotRoot will perform pivot root syscall into path.
func PivotRoot(path string) error {
	err := syscall.Mount(path, path, "", syscall.MS_BIND|syscall.MS_REC, "")
	if err != nil {
		return fmt.Errorf("error setting private mount: %s. %v", path, err.Error())
	}

	err = syscall.Mount("", path, "", syscall.MS_PRIVATE|syscall.MS_REC, "")
	if err != nil {
		return fmt.Errorf("error setting private mount: %s. %v", path, err.Error())
	}

	err = syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")
	if err != nil {
		return fmt.Errorf("error setting private mount: %s. %v", path, err.Error())
	}

	// first we set up pivotroot.
	if !Exist(path) {
		return fmt.Errorf("pivotroot: rootfs %s does not exist", path)
	}

	tmpDir := filepath.Join(path, "/")
	pivotDir := filepath.Join(tmpDir, ".pivot_root")

	_ = os.Remove(tmpDir)
	_ = os.Remove(pivotDir)

	err = os.MkdirAll(tmpDir, 0o755)
	if err != nil {
		return fmt.Errorf("pivotroot: can't create tmp dir %s, error %w", tmpDir, err)
	}

	err = os.Mkdir(pivotDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("pivotroot: can't create pivot_root dir %s, error %w", pivotDir, err)
	}

	err = syscall.PivotRoot(path, pivotDir)
	if err != nil {
		return fmt.Errorf("pivotroot: %w", err)
	}

	// path to pivot dir now changed, update
	pivotDir = filepath.Join("/", filepath.Base(pivotDir))

	err = syscall.Unmount(pivotDir, syscall.MNT_DETACH)
	if err != nil {
		return fmt.Errorf("unmount pivot_root dir %w", err)
	}

	err = os.Remove(pivotDir)
	if err != nil {
		return fmt.Errorf("cleanup pivot_root dir %w", err)
	}

	return nil
}
