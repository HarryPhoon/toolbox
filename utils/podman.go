package utils

import (
	"encoding/json"
	"os/exec"
	"syscall"

	"github.com/pkg/errors"
)

// GetContainers is a wrapper function around `podman ps --format json` command.
//
// Parameter args accepts an array of strings to be passed to the wrapped command (eg. ["-a", "--filter", "123"]).
//
// Returned value is a slice of dynamically unmarshalled json, so it needs to be treated properly.
//
// If a problem happens during execution, first argument is nil and second argument holds the error message.
func GetContainers(args ...string) ([]map[string]interface{}, error) {
	args = append([]string{"ps", "--format", "json"}, args...)
	output, err := PodmanOutput(args...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var containers []map[string]interface{}

	err = json.Unmarshal(output, &containers)
	if err != nil {
		return nil, errors.Wrap(err, "Problem while unmarshalling json with toolbox containers")
	}

	return containers, nil
}

// GetImages is a wrapper function around `podman images --format json` command.
//
// Parameter args accepts an array of strings to be passed to the wrapped command (eg. ["-a", "--filter", "123"]).
//
// Returned value is a slice of dynamically unmarshalled json, so it needs to be treated properly.
//
// If a problem happens during execution, first argument is nil and second argument holds the error message.
func GetImages(args ...string) ([]map[string]interface{}, error) {
	args = append([]string{"images", "--format", "json"}, args...)
	output, err := PodmanOutput(args...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var images []map[string]interface{}

	err = json.Unmarshal(output, &images)
	if err != nil {
		return nil, errors.Wrap(err, "Problem while unmarshalling json with toolbox images")
	}

	return images, nil
}

func PodmanOutput(args ...string) ([]byte, error) {
	cmd := exec.Command("podman", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, handleErrorCode(err)
	}
	return output, nil
}

func PodmanRun(args ...string) error {
	cmd := exec.Command("podman", args...)
	err := cmd.Run()
	if err != nil {
		return handleErrorCode(err)
	}
	return nil
}

func handleErrorCode(err error) error {
	if exitError, ok := err.(*exec.ExitError); ok {
		ws := exitError.Sys().(syscall.WaitStatus)
		switch ws.ExitStatus() {
		case 1:
			return errors.Errorf("No such container")
		case 2:
			return errors.Errorf("Container is running")
		case 125:
			return errors.Errorf("Failed to inspect container")
		default:
			return err
		}
	}
	return err
}
