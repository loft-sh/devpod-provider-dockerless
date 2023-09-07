package options

import (
	"fmt"
	"os"
)

type Options struct {
	DevContainerID string
	TargetDir      string
}

func FromEnv() (*Options, error) {
	retOptions := &Options{}

	var err error

	// required
	retOptions.DevContainerID, err = fromEnvOrError("DEVCONTAINER_ID")
	if err != nil {
		return nil, err
	}

	// required
	retOptions.TargetDir, err = fromEnvOrError("TARGET_DIR")
	if err != nil {
		return nil, err
	}

	return retOptions, nil
}

func fromEnvOrError(name string) (string, error) {
	val := os.Getenv(name)
	if val == "" {
		return "", fmt.Errorf(
			"couldn't find option %s in environment, please make sure %s is defined",
			name,
			name,
		)
	}

	return val, nil
}
