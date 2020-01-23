/*
Copyright © 2019 Red Hat Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/containers/toolbox/utils"
	"github.com/godbus/dbus/v5"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	runFlags struct {
		containerName  string
		releaseVersion string
	}
)

var runCmd = &cobra.Command{
	Use:   "run [container] [command...]",
	Short: "Run a command in an existing toolbox container",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd, args)
	},
	Args: cobra.MinimumNArgs(2),
}

func init() {
	rootCmd.AddCommand(runCmd)

	flags := runCmd.Flags()
	flags.StringVarP(&runFlags.releaseVersion, "release", "r", "", "Run command inside a toolbox container with the release version")
}

func run(cmd *cobra.Command, args []string) error {
	containerName := args[0]
	commands := args[1:]
	imageName := ""

	containerName, imageName = utils.UpdateContainerAndImageNames(containerName, imageName, runFlags.releaseVersion)

	logrus.Debugf("Container: '%s' Image: '%s'", containerName, imageName)

	// Check the existence of a container
	logrus.Infof("Checking if container '%s' exists", containerName)
	if !utils.ContainerExists(containerName) {
		logrus.Fatalf("Container '%s' not found", containerName)

		// FIXME: Here can be placed the offer to create a container
	}

	// Prepare Flatpak session-helper
	conn, err := dbus.SessionBus()
	if err != nil {
		logrus.Error("Failed to connect to Session Bus")
	}
	defer conn.Close()

	logrus.Info("Calling org.freedesktop.Flatpak.SessionHelper.RequestSession")
	SessionHelper := conn.Object("org.freedesktop.Flatpak", "/org/freedesktop/Flatpak/SessionHelper")
	call := SessionHelper.Call("org.freedesktop.Flatpak.SessionHelper.RequestSession", 0)
	if call.Err != nil {
		logrus.Fatal("Failed to call org.freedesktop.Flatpak.SessionHelper.RequestSession")
	}

	logrus.Infof("Inspecting container '%s'", containerName)
	containerInfo, err := utils.PodmanInspect("container", containerName)
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Infof("Starting container '%s'", containerName)

	err = containerStart(containerName)
	// FIXME: This error handling shouldn't be left too general
	if err != nil {
		logrus.Fatal(err)
	}

	containerConfig := containerInfo["Config"].(map[string]interface{})
	containerState := containerInfo["State"].(map[string]interface{})
	containerEntryPoint := containerConfig["Cmd"].([]interface{})[0]

	logrus.Infof("Entry point of container '%s' is '%s'", containerName, containerEntryPoint)

	if containerEntryPoint == "toolbox" {
		logrus.Infof("Waiting for container '%s' to finish initializing", containerName)

		entryPointPIDString := fmt.Sprint(containerState["Pid"])
		entryPointPID, err := strconv.Atoi(entryPointPIDString)
		if err != nil {
			logrus.Debugf("PID: %s", entryPointPIDString)
			logrus.Fatalf("Failed to parse entry point PID of container '%s'", containerName)
		}

		if entryPointPID <= 0 {
			logrus.Debugf("PID: %d", entryPointPID)
			logrus.Fatalf("Invalid entry point PID of container '%s'", containerName)
		}

		// TODO: Finish initialization of a container (need to ask Rishi about this)
	} else {
		logrus.Warnf("Container '%s' uses deprecated features", containerName)
		logrus.Warn("Consider recreating it with Toolbox version 0.0.17 or newer")
	}

	args = []string{"exec", "--user", "root:root", containerName, "touch", "/run/.toolboxenv"}
	err = utils.PodmanRun(args...)
	if err != nil {
		logrus.Fatalf("Failed to create /run/.toolboxenv in container %s", containerName)
	}

	logrus.Infof("Looking for program '%s' in container %s", commands[0], containerName)

	// TODO: Finish searching for a command in a container

	logrus.Infof("Running in container '%s': %v", containerName, commands)

	args = []string{"exec",
		"--interactive",
		"--tty",
		"--user", viper.GetString("USER"),
		"--workdir", viper.GetString("PWD")}

	// Add the environment variables that hold a value
	for _, env := range preservedEnvVars {
		value := viper.GetString(env)
		if len(value) != 0 {
			args = append(args, fmt.Sprintf("--env=%s=%s", env, value))
		}
	}

	args = append(args, []string{
		containerName,
		"capsh", "--caps=", "--", "-c"}...)

	args = append(args, commands...)

	err = utils.PodmanInto(args...)
	if err != nil {
		logrus.Fatal(err)
	}

	return nil
}

func containerStart(containerName string) error {
	args := []string{"start", containerName}
	output, err := utils.PodmanOutput(args...)
	if err != nil {
		if strings.Contains(string(output), "use system migrate to mitigate") {
			logrus.Info("Checking if 'podman system migrate' support '--new-runtime' option")

			if utils.CheckPodmanVersion("1.6.2") {
				return errors.New("Podman doesn't support --new-runtime option")
			}

			cgroupVersion, err := utils.GetCgroupsVersion()
			if err != nil {
				return err
			}

			ociRuntimeRequired := "runc"
			if cgroupVersion == "v2" {
				ociRuntimeRequired = "crun"
			}

			logrus.Infof("Migrating containers to OCI runtime %s", ociRuntimeRequired)

			err = utils.PodmanRun("system", "migrate", "--new-runtime", ociRuntimeRequired)
			if err != nil {
				return fmt.Errorf("Failed to migrate containers to OCI runtime '%s'", ociRuntimeRequired)
			}

			err = utils.PodmanRun("start", containerName)
			if err != nil {
				return fmt.Errorf("Container '%s' doesn't support cgroups %s", containerName, cgroupVersion)
			}
		}

		return errors.New("Failed to start container")
	}

	return nil
}
