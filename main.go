package main

import (
	_ "embed"
	"log"
	"os"
	"path/filepath"

	"github.com/loft-sh/devpod-provider-dockerless/cmd"
)

//nolint: typecheck
//go:embed rootlesskit
var rootlesskit []byte
var rootlesskitPath = filepath.Join("/tmp/dockerless", "rootlesskit")

//nolint: typecheck
//go:embed slirp4netns
var slirp4netns []byte
var slirp4netnsPath = filepath.Join("/tmp/dockerless", "slirp4netns")

func main() {
	_, err := os.Stat(rootlesskitPath)
	if err != nil {
		err = os.MkdirAll("/tmp/dockerless", 0o755)
		if err != nil {
			log.Fatal(err)
		}

		err = os.WriteFile(rootlesskitPath, rootlesskit, 0o755)
		if err != nil {
			log.Fatal(err)
		}
	}

	_, err = os.Stat(slirp4netnsPath)
	if err != nil {
		err = os.MkdirAll("/tmp/dockerless", 0o755)
		if err != nil {
			log.Fatal(err)
		}

		err = os.WriteFile(slirp4netnsPath, slirp4netns, 0o755)
		if err != nil {
			log.Fatal(err)
		}
	}

	err = os.Setenv("PATH", os.Getenv("PATH")+":/tmp/dockerless")
	if err != nil {
		log.Fatal(err)
	}

	cmd.Execute()
}
