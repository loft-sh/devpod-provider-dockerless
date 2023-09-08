package dockerless

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// GetPid will return the pid of the process running the container with input id.
func GetPid(id string) (int, error) {
	idb := []byte(
		os.Args[0] + "\000" +
			"enter" + "\000" +
			base64.StdEncoding.EncodeToString([]byte(id)) + "\000",
		)

	processes, err := os.ReadDir("/proc")
	if err != nil {
		return -1, err
	}

	// manually find in /proc a process that has "lilipod enter" and "id" in cmdline
	for _, proc := range processes {
		cmdline := filepath.Join("/proc", proc.Name(), "cmdline")

		filedata, err := os.ReadFile(cmdline)
		if err != nil {
			continue
		}

		// if the maps file contains the ID of the container, we found it
		if bytes.Equal(filedata, idb) {
			pid, err := strconv.ParseInt(proc.Name(), 10, 32)
			if err != nil {
				return -1, err
			}

			return int(pid), nil
		}
	}

	return -1, fmt.Errorf("container %s is not running", id)
}
