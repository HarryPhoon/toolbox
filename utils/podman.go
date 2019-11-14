package utils

import (
	"os/exec"
	"syscall"

	"github.com/pkg/errors"
)

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
