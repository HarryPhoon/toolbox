package podman

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/mcuadros/go-version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	// ErrNonExistent signals one of the specified containers does not exist (applies to `podman rm/rmi`)
	ErrNonExistent = errors.New("exit status 1")
	// ErrRunningContainer signals one of the specified containers is paused or running (applies to `podman rm`)
	ErrRunningContainer = errors.New("exit status 2")
	// ErrHasChildren signals one of the specified images has child images or is used by a container (applies to `podman rmi`)
	ErrHasChildren = errors.New("exit status 2")
	// ErrInternal signals an error in Podman itself
	ErrInternal = errors.New("exit status 125")
	// ErrCmdCantInvoke signals a contained command cannot be invoked (applies to `podman run/exec`)
	ErrCmdCantInvoke = errors.New("exit status 126")
	// ErrCmdNotFound signals a contained command cannot be found (applies to `podman run/exec`)
	ErrCmdNotFound = errors.New("exit status 127")
)

func IsPathBindMount(path string, containerInfo map[string]interface{}) bool {
	containerMounts := containerInfo["Mounts"].([]interface{})
	for _, mount := range containerMounts {
		dest := fmt.Sprint(mount.(map[string]interface{})["Destination"])
		if dest == path {
			return true
		}
	}

	return false
}

// CheckPodmanVersion compares provided version with the version of Podman.
//
// Takes in one string parameter that should be in the format that is used for versioning (eg. 1.0.0, 2.5.1-dev).
//
// Returns true if Podman version is at least equal to the provided version.
// Returns false if Podman version is not sufficient.
func CheckPodmanVersion(requiredVersion string) bool {
	args := []string{"version", "-f", "json"}
	output, err := CmdOutput(args...)
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

// PodmanInfo is a wrapper around `podman info` command
func PodmanInfo() (map[string]interface{}, error) {
	args := []string{"info", "--format", "json"}
	output, err := CmdOutput(args...)
	if err != nil {
		return nil, err
	}

	var podmanInfo map[string]interface{}

	err = json.Unmarshal(output, &podmanInfo)
	if err != nil {
		return nil, err
	}

	return podmanInfo, nil
}

// PodmanInspect is a wrapper around 'podman inspect' command
//
// Parameter 'typearg' takes in values 'container' or 'image' that is passed to the --type flag
func PodmanInspect(typearg string, target string) (map[string]interface{}, error) {
	args := []string{"inspect", "--format", "json", "--type", typearg, target}
	output, err := CmdOutput(args...)
	if err != nil {
		return nil, err
	}

	var info []map[string]interface{}

	err = json.Unmarshal(output, &info)
	if err != nil {
		return nil, err
	}

	return info[0], nil
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
	output, err := CmdOutput(args...)
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
	output, err := CmdOutput(args...)
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

	err := CmdRun(args...)
	if err != nil {
		return false
	}

	return true
}

// ContainerExists checks using Podman if a container with given ID/name exists.
//
// Parameter container is a name or an id of a container.
func ContainerExists(containerName string) bool {
	args := []string{"container", "exists", containerName}

	err := CmdRun(args...)
	if err != nil {
		return false
	}

	return true
}

// PullImage pulls an image
func PullImage(imageName string) bool {
	args := []string{"pull", imageName}

	logLevel := fmt.Sprint(logrus.GetLevel())
	if viper.GetBool("log-podman") {
		args = append([]string{"--log-level", logLevel}, args...)
	}

	cmd := exec.Command("podman", args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		return false
	}

	return true
}

// CmdOutput is a wrapper around Podman that returns the output of the invoked command.
//
// Parameter args accepts an array of strings to be passed to Podman.
//
// If no problem while executing a command occurs, then the output of the command is returned in the first value.
// If a problem occurs, then the error code is returned in the second value.
func CmdOutput(args ...string) ([]byte, error) {
	logLevel := fmt.Sprint(logrus.GetLevel())
	if viper.GetBool("log-podman") {
		args = append([]string{"--log-level", logLevel}, args...)
	}
	cmd := exec.Command("podman", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if logLevel == "debug" || logLevel == "trace" {
		fmt.Fprint(os.Stderr, stderr.String())
	}

	if err != nil {
		return stderr.Bytes(), err
	}

	return stdout.Bytes(), nil
}

// CmdRun is a wrapper around Podman that does not return the output of the invoked command.
//
// Parameter args accepts an array of strings to be passed to Podman.
//
// If no problem while executing a command occurs, then the returned value is nil.
// If a problem occurs, then the error code is returned.
func CmdRun(args ...string) error {
	logLevel := fmt.Sprint(logrus.GetLevel())
	if viper.GetBool("log-podman") {
		args = append([]string{"--log-level", logLevel}, args...)
	}
	cmd := exec.Command("podman", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if logLevel == "debug" || logLevel == "trace" {
		fmt.Fprint(os.Stderr, stderr.String())
	}

	if err != nil {
		return err
	}

	return nil
}

func CmdInto(args ...string) error {
	logLevel := fmt.Sprint(logrus.GetLevel())
	if viper.GetBool("log-podman") {
		args = append([]string{"--log-level", logLevel}, args...)
	}
	cmd := exec.Command("podman", args...)

	cmd.Stdout = os.Stdout
	// Seems like there is no need to pipe the command stderr to the system one by default
	if viper.GetBool("log-podman") {
		cmd.Stderr = os.Stderr
	}
	cmd.Stdin = os.Stdin

	err := cmd.Run()

	if err != nil {
		return err
	}

	return nil
}
