package dockerless

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// UntarFile will untar target file to target directory.
// If userns is specified and it is keep-id, it will perform the
// untarring in a new user namespace with user id maps set, in order to prevent
// permission errors.
func UntarFile(workspaceId, path, target string) error {
	// first ensure we can write
	err := syscall.Access(path, 2)
	if err != nil {
		return err
	}

	command := ""
	var args []string

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
		"tar", "--exclude=dev/*", "-xpf", path, "-C", target,
	}...,
	)

	cmd := exec.Command(command, args...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(out))
	}

	return nil
}

// GetFileDigest will return the sha256sum of input file. Empty if error occurs.
func GetFileDigest(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}

	defer func() { _ = file.Close() }()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return ""
	}

	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// CheckFileDigest will compare input digest to the checksum of input file.
// Returns whether the input digest is equal to the input file's one.
func CheckFileDigest(path string, digest string) bool {
	checksum := GetFileDigest(path)

	return "sha256:"+checksum == digest
}

// Exist returns if a path exists or not.
func Exist(path string) bool {
	var stat syscall.Stat_t
	err := syscall.Stat(path, &stat)

	return err == nil
}

// Mount will bind-mount src to dest, using input mode.
func Mount(src, dest string, mode uintptr) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		_ = os.MkdirAll(dest, 0o755)
	} else {
		file, _ := os.Create(dest)

		defer func() { _ = file.Close() }()
	}

	return syscall.Mount(src,
		dest,
		"bind",
		mode,
		"")
}

// MountShm will mount a new shm tmpfs to dest path.
// Said mount will be created with mode: noexec,nosuid,nodev,mode=1777,size=65536k.
func MountShm(dest string) error {
	_ = os.MkdirAll(dest, 0o777)

	return syscall.Mount("shm",
		dest,
		"tmpfs",
		syscall.MS_NOEXEC|syscall.MS_NOSUID|syscall.MS_NODEV,
		"mode=1777,size=65536k")
}

// MountMqueue will mount a new mqueue tmpfs in dest path.
// Said mount will be created with mode: noexec,nosuid,nodev.
func MountMqueue(dest string) error {
	_ = os.MkdirAll(dest, 0o777)

	return syscall.Mount("mqueue",
		dest,
		"mqueue",
		syscall.MS_NOEXEC|syscall.MS_NOSUID|syscall.MS_NODEV,
		"")
}

// MountTmpfs will mount a new tmpfs in dest path.
func MountTmpfs(dest string) error {
	_ = os.MkdirAll(dest, 0o777)

	return syscall.Mount("tmpfs",
		dest,
		"tmpfs",
		uintptr(0),
		"")
}

// MountProc will mount a new procfs in dest path.
// Said mount will be created with mode: noexec,nosuid,nodev.
func MountProc(dest string) error {
	_ = os.MkdirAll(dest, 0o755)

	return syscall.Mount("proc",
		dest,
		"proc",
		syscall.MS_NOEXEC|syscall.MS_NOSUID|syscall.MS_NODEV,
		"")
}

// MountDevPts will mount a new devpts in dest path.
// Said mount will be created with mode: noexec,nosuid,newinstance,ptmxmode=0666,mode=0620.
func MountDevPts(dest string) error {
	_ = os.MkdirAll(dest, 0o755)

	return syscall.Mount("devpts",
		dest,
		"devpts",
		syscall.MS_NOEXEC|syscall.MS_NOSUID,
		"newinstance,ptmxmode=0666,mode=0620")
}

// MountBind will bind-mount src path in dest path.
// Said mount will be created with mode: rbind,rprivate.
func MountBind(src, dest string) error {
	return Mount(src, dest, syscall.MS_BIND|syscall.MS_REC|syscall.MS_PRIVATE)
}
