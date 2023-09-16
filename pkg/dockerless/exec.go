package dockerless

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func (p *DockerlessProvider) ExecuteCommand(ctx context.Context, workspaceId, user, command string, stdin io.Reader, stdout, stderr io.Writer) error {
	containerDIR := filepath.Join(p.Config.TargetDir, "rootfs", workspaceId)

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
	}

	// user namespace only if we're rootless
	if os.Getuid() > 0 {
		args = append(args, "-U")
		args = append(args, "--preserve-credentials")
	}

	args = append(args, []string{
		"-t",
		string(pid),
		os.Args[0],
		"enter",
		"--entrypoint",
	}...)

	if user != "" && user != "0" && user != "root" {
		uid := findUserPasswd(containerDIR, user)
		command = "su -l " + uid + " -c " + command
	}

	command = base64.StdEncoding.EncodeToString([]byte(command))

	args = append(args, command)

	cmd := exec.Command(nsenter, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

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
