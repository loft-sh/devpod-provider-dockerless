package main

import (
	_ "embed"
	"log"
	"os"
	"path/filepath"

	"github.com/loft-sh/devpod-provider-dockerless/cmd"
)

//go:embed rootlesskit
var rootlesskit []byte
var rootlesskitPath = filepath.Join("/tmp/dockerless", "rootlesskit")

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

	err = os.Setenv("PATH", os.Getenv("PATH")+":/tmp/dockerless")
	if err != nil {
		log.Fatal(err)
	}

	cmd.Execute()
}
