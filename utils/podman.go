package utils

import (
	"encoding/json"
	"errors"
	"os/exec"
	"syscall"

	"github.com/mcuadros/go-version"

	"github.com/sirupsen/logrus"
)

// CheckPodmanVersion compares provided version with the version of Podman.
//
// Takes in one string parameter that should be in the format that is used for versioning (eg. 1.0.0, 2.5.1-dev).
//
// Returns true if Podman version is at least equal to the provided version.
// Returns false if Podman version is not sufficient.
func CheckPodmanVersion(requiredVersion string) bool {
	args := []string{"version", "-f", "json"}
	output, err := PodmanOutput(args...)
	if err != nil {
		logrus.Error(err)
		return false
	}

	var jsonoutput map[string]interface{}
	err = json.Unmarshal(output, &jsonoutput)
	if err != nil {
		logrus.Error(err)
		return false
	}

	podmanVersion := jsonoutput["Client"].(map[string]interface{})["Version"].(string)
	podmanVersion = version.Normalize(podmanVersion)
	requiredVersion = version.Normalize(requiredVersion)

	if version.CompareSimple(podmanVersion, requiredVersion) >= 0 {
		return true
	}

	return false
}

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
		return nil, err
	}

	var containers []map[string]interface{}

	err = json.Unmarshal(output, &containers)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	var images []map[string]interface{}

	err = json.Unmarshal(output, &images)
	if err != nil {
		return nil, err
	}

	return images, nil
}

// ImageExists checks using Podman if an image with given ID/name exists.
//
// Parameter image is a name or an id of an image.
func ImageExists(image string) bool {
	args := []string{"image", "exists", image}

	err := PodmanRun(args...)
	if err != nil {
		return false
	}

	return true
}

// ContainerExists checks using Podman if a container with given ID/name exists.
//
// Parameter container is a name or an id of a container.
func ContainerExists(container string) bool {
	args := []string{"container", "exists", container}

	err := PodmanRun(args...)
	if err != nil {
		return false
	}

	return true
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

// FIXME: Handling exit codes globally is not really the best idea
func handleErrorCode(err error) error {
	if exitError, ok := err.(*exec.ExitError); ok {
		ws := exitError.Sys().(syscall.WaitStatus)
		switch ws.ExitStatus() {
		case 1:
			return errors.New("No such container/image")
		case 2:
			return errors.New("Container is running")
		case 125:
			return errors.New("Failed to inspect container")
		default:
			return err
		}
	}
	return err
}
