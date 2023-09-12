package dockerless

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/loft-sh/devpod/pkg/driver"
	"github.com/schollz/progressbar/v3"
)

// Pull will pull a given image and save it to ImageDir.
// This function uses github.com/google/go-containerregistry/pkg/crane to pull
// the image's manifest, and performs the downloading of each layer separately.
// Each layer is deduplicated between images in order to save space, using hardlinks.
func (p *DockerlessProvider) Pull(ctx context.Context, runOptions *driver.RunOptions) error {
	// First we try to get the fully qualified uri of the image
	// eg alpine:latest -> index.docker.io/library/alpine:latest
	image := runOptions.Image
	ref, err := name.ParseReference(image)
	if err == nil {
		image = ref.Name()
	}

	targetDIR := filepath.Join(p.Config.TargetDir, "images", ref.Name())

	// if we already downloaded the image, exit
	_, err = os.Stat(filepath.Join(targetDIR, "manifest.json"))
	if err == nil {
		return nil
	}

	// Pull will just get us the v1.Image struct, from
	// which we get all the information we need
	imageManifest, err := crane.Pull(image)
	if err != nil {
		return err
	}

	// We get the layers
	layers, err := imageManifest.Layers()
	if err != nil {
		return err
	}

	// Prepare the image path
	if !Exist(targetDIR) {
		err := os.MkdirAll(targetDIR, os.ModePerm)
		if err != nil {
			return err
		}
	}

	keepFiles := []string{}
	// Now we download the layers
	for _, layer := range layers {
		fileName, err := downloadLayer(targetDIR, layer)
		if err != nil {
			return err
		}

		keepFiles = append(keepFiles, fileName)
	}

	fileList, err := os.ReadDir(targetDIR)
	if err != nil {
		return err
	}

	for _, file := range fileList {
		if !strings.Contains(
			strings.Join(keepFiles, ":"),
			filepath.Base(file.Name()),
		) {
			err = os.Remove(filepath.Join(targetDIR, file.Name()))
			if err != nil {
				return err
			}
		}
	}

	// we save the manifest.json for later use. This contains
	// the information on how the layers are ordered and
	// how to unpack them
	rawManifest, err := imageManifest.RawManifest()
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(targetDIR, "manifest.json"), rawManifest, 0o644)
	if err != nil {
		return err
	}

	// The config.json file is also saved, indicating lots of information
	// about the image, like default env, entrypoint and so on
	rawConfig, err := imageManifest.RawConfigFile()
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(targetDIR, "config.json"), rawConfig, 0o644)
	if err != nil {
		return err
	}

	// We also save the fully qualified name to retrieve it later
	err = os.WriteFile(filepath.Join(targetDIR, "image_name"), []byte(image), 0o644)
	if err != nil {
		return err
	}

	return nil
}

// downloadLayer will download input layer into targetDIR.
// downloadLayer will first searc hexisting images inside the ImageDir in order
// to find matching layers, and hardlink them in order to save disk space.
//
// Each layer download is verified in order to ensure no corrupted downloads occur.
func downloadLayer(targetDIR string, layer v1.Layer) (string, error) {
	// we use this as a path to download layers, in order to
	// verify them and ensure we do not leave broken files
	tmpdir := filepath.Join(targetDIR, ".temp")

	// always cleanup before
	_ = os.RemoveAll(tmpdir)

	err := os.MkdirAll(tmpdir, 0o750)
	if err != nil {
		return "", err
	}

	// and after
	defer func() { _ = os.RemoveAll(tmpdir) }()

	layerDigest, _ := layer.Digest()

	layerFileName := strings.Split(layerDigest.String(), ":")[1] + ".tar.gz"

	// If a layer already exists, exit
	if Exist(filepath.Join(targetDIR, layerFileName)) &&
		CheckFileDigest(filepath.Join(targetDIR, layerFileName), layerDigest.String()) {

		return layerFileName, nil
	}

	// But if a layer with the same name/digest exists in another directory
	// let's deduplicate the disk usage by using hardlinks
	matchingLayers := findExistingLayer(filepath.Dir(targetDIR), layerFileName)
	if len(matchingLayers) > 0 &&
		CheckFileDigest(matchingLayers[0], layerDigest.String()) {

		return layerFileName, os.Link(matchingLayers[0], filepath.Join(targetDIR, layerFileName))
	}

	// Else we proceed with the download of the layer
	savedLayer, err := os.Create(filepath.Join(tmpdir, layerFileName))
	if err != nil {
		return "", err
	}

	defer func() { _ = savedLayer.Close() }()

	tarLayer, err := layer.Compressed()
	if err != nil {
		return "", err
	}

	layerSize, err := layer.Size()
	if err != nil {
		return "", err
	}

	bar := progressbar.NewOptions64(layerSize,
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(30),
		progressbar.OptionSetVisibility(true),
		progressbar.OptionSetDescription("Copying blob "+layerDigest.String()),
	)

	_, err = io.Copy(io.MultiWriter(savedLayer, bar), tarLayer)
	if err != nil {
		return "", err
	}

	// always verify if the download was correctly done by
	// checking the digest of the file
	if CheckFileDigest(filepath.Join(tmpdir, layerFileName), layerDigest.String()) {
		err = os.Rename(filepath.Join(tmpdir, layerFileName),
			filepath.Join(targetDIR, layerFileName))

		return layerFileName, err
	}

	return "", fmt.Errorf("error getting layer")
}

// findExistingLayer is useful to find layers with matching name/digest in order to
// deduplicate disk usage by using hardlinks later.
func findExistingLayer(targetDIR, filename string) []string {
	var matchingFiles []string

	_ = filepath.WalkDir(targetDIR, func(name string, dirEntry fs.DirEntry, err error) error {
		if dirEntry.Name() == filename {
			matchingFiles = append(matchingFiles, name)
		}

		return nil
	})

	return matchingFiles
}
