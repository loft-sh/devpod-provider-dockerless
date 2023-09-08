package dockerless

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/loft-sh/devpod/pkg/driver"
)

func (p *DockerlessProvider) ExecuteCommand(ctx context.Context, workspaceId, user, command string, stdin io.Reader, stdout, stderr io.Writer) error {
	runOptions := &driver.RunOptions{}

	err := json.Unmarshal([]byte(os.Getenv("DEVCONTAINER_RUN_OPTIONS")), runOptions)
	if err != nil {
		return fmt.Errorf("unmarshal run options: %w", err)
	}

	ppid, err := GetPid(workspaceId)
	if err != nil {
		return fmt.Errorf("container %s is not running", workspaceId)
	}

	// We want to enter the namespace of the PID1 inside the container
	pid, err := exec.Command("pgrep", "-P", strconv.Itoa(ppid)).Output()
	if err != nil {
		return fmt.Errorf("container %s is not running", workspaceId)
	}

	pid = bytes.TrimSpace(pid)

	nsenter := "nsenter"
	args := []string{
		"-m",
		"-u",
		"-i",
		"-p",
		"-U",
		"-t",
		string(pid),
	}

	if user == "" {
		args = append(args, command)
	} else {
		containerDIR := filepath.Join(p.Config.TargetDir, "rootfs", workspaceId)
		uid := findUserPasswd(containerDIR, user)

		args = append(args, []string{"su", uid, "-c", command}...)
	}

	cmd := exec.Command(nsenter, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	environB, err := os.ReadFile(filepath.Join("/proc", string(pid), "environ"))
	if err == nil {
		environ := strings.Split(string(environB), "\000")
		cmd.Env = environ
	}

	return cmd.Run()
}

func findUserPasswd(path, user string) string {
	passwd, err := os.ReadFile(filepath.Join(path, "/etc/passwd"))
	if err != nil {
		return "root"
	}

	// find in /etc/passwd either ":uid:" or "username:"
	pattern := regexp.MustCompile(".*:" + user + ":.*")
	match := pattern.FindString(string(passwd))

	if len(match) == 0 {
		return user
	}

	return strings.Split(match, ":")[0]
}
