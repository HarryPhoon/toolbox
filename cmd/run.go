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
	"time"

	"github.com/containers/toolbox/pkg/podman"
	"github.com/containers/toolbox/pkg/utils"
	"github.com/godbus/dbus/v5"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	runFlags struct {
		useDefaultContainer bool
		releaseVersion      string
		emitEscapeSequence  bool
		fallbackToBash      bool
		promptForCreate     bool
		pedantic            bool
	}
)

var runCmd = &cobra.Command{
	Use:   "run [flags] CONTAINER [COMMAND [ARG...]]",
	Short: "Run a command in an existing toolbox container",
	Run: func(cmd *cobra.Command, args []string) {
		run(args)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	flags := runCmd.Flags()
	// This stops parsing of flags after arguments. Necessary to properly pass the command to the container
	flags.SetInterspersed(false)

	flags.BoolVarP(&runFlags.useDefaultContainer, "default", "d", false, "Run command inside the system default container (eg. on Fedora 31 -> fedora-toolbox-31)")
	flags.StringVarP(&runFlags.releaseVersion, "release", "r", "", "Run command inside a toolbox container with the release version")
	flags.BoolVar(&runFlags.emitEscapeSequence, "escape-sequence", false, "Emit an escape sequence for terminals")
	flags.BoolVar(&runFlags.fallbackToBash, "fallback-to-bash", false, "If requested program does not exist, fallback to bash")
	flags.BoolVar(&runFlags.promptForCreate, "prompt-for-create", true, "Offer to create a container if it does not exist")
	flags.BoolVar(&runFlags.pedantic, "pedantic", true, "")

	// These options dont have to be visible to an everyday user
	flags.MarkHidden("fallback-to-bash")
	flags.MarkHidden("escape-sequence")
	flags.MarkHidden("prompt-for-create")
	flags.MarkHidden("pedantic")
}

func run(args []string) error {
	var containerName string = ""
	var imageName string = ""
	var commands []string = nil

	// When the release version is specified we want to use it over the container name
	if runFlags.releaseVersion != "" {
		if len(args) == 0 {
			logrus.Fatal("You must provide a command to execute")
		}
		commands = args[0:]
	} else if runFlags.useDefaultContainer {
		if len(args) == 0 {
			logrus.Fatal("You must provide a command to execute")
		}
		containerName = ""
		commands = args[0:]
	} else {
		if len(args) == 0 {
			logrus.Fatal("You must provide a name of a container")
		} else if len(args) == 1 {
			logrus.Fatal("You must provide a command to execute")
		}
		containerName = args[0]
		commands = args[1:]
	}

	containerName, imageName, _ = utils.UpdateContainerAndImageNames(containerName, imageName, runFlags.releaseVersion)

	logrus.Debugf("Container: '%s' Image: '%s'", containerName, imageName)

	logrus.Infof("Checking if container '%s' exists", containerName)
	if !podman.ContainerExists(containerName) {
		if !runFlags.pedantic {
			logrus.Errorf("Container '%s' not found", containerName)
			containers, err := GetContainers()
			if err != nil {
				logrus.Fatal("Error while fetching containers")
			}

			containersCount := len(containers)
			logrus.Infof("Found %d containers", containersCount)

			if containersCount == 0 {
				createContainer := false

				if rootFlags.assumeyes {
					runFlags.promptForCreate = false
					createContainer = true
				}

				if runFlags.promptForCreate {
					var response string
					for true {
						fmt.Printf("No toolbox containers found. Create now? [y/N]: ")
						fmt.Scanf("%s", &response)
						response = strings.ToLower(response)

						if response == "y" || response == "yes" {
							createContainer = true
						}

						break
					}
				}

				if !createContainer {
					logrus.Fatal("A container can be created later with the 'create' command.")
				}

				create([]string{containerName})
			} else if containersCount == 1 {
				containerName = fmt.Sprint(containers[0]["Names"])
				logrus.Infof("Entering container %s instead", containerName)
			} else {
				logrus.Fatal("Specify a name of a container")
			}
		} else {
			logrus.Fatalf("Container '%s' not found", containerName)
		}
	}

	// Prepare Flatpak session-helper
	conn, err := dbus.SessionBus()
	if err != nil {
		logrus.Error("Failed to connect to Session Bus")
	}

	logrus.Info("Calling org.freedesktop.Flatpak.SessionHelper.RequestSession")
	SessionHelper := conn.Object("org.freedesktop.Flatpak", "/org/freedesktop/Flatpak/SessionHelper")
	call := SessionHelper.Call("org.freedesktop.Flatpak.SessionHelper.RequestSession", 0)
	if call.Err != nil {
		logrus.Debug(call.Err)
		logrus.Fatal("Failed to call org.freedesktop.Flatpak.SessionHelper.RequestSession")
	}

	logrus.Infof("Starting container '%s'", containerName)
	err = containerStart(containerName)
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Infof("Inspecting container '%s'", containerName)
	containerInfo, err := podman.PodmanInspect("container", containerName)
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

		// Wait for initialization of a container
		containerInitializedStamp := fmt.Sprintf("%s/container-initialized-%d", viper.GetString("TOOLBOX_RUNTIME_DIRECTORY"), entryPointPID)
		logrus.Infof("Checking if initialization stamp %s exists", containerInitializedStamp)
		containerInitializedTimeout := 25
		i := 0
		for !utils.PathExists(containerInitializedStamp) {
			time.Sleep(time.Second)
			i++

			if i == containerInitializedTimeout {
				logrus.Fatalf("Failed to initialize container '%s'", containerName)
			}
		}
		logrus.Infof("Container '%s' is properly initialized", containerName)
	} else {
		logrus.Warnf("Container '%s' uses deprecated features", containerName)
		logrus.Warn("Consider recreating it with Toolbox version 0.0.17 or newer")
	}

	logrus.Infof("Looking for program '%s' in container %s", commands[0], containerName)

	args = []string{"exec",
		"--user", viper.GetString("USER"),
		containerName,
		"sh", "-c", `command -v "$1"`, "sh", commands[0]}
	err = podman.CmdRun(args...)
	if err != nil {
		if runFlags.fallbackToBash {
			logrus.Infof("%s not found in '%s'; using /bin/bash instead", commands[0], containerName)
			commands = nil
			commands = []string{"/bin/bash"}
		} else {
			logrus.Fatalf("%s not found in '%s'", commands[0], containerName)
		}
	}

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
		"capsh", "--caps=", "--", "-c", `exec "$@"`, "/bin/sh"}...)

	args = append(args, commands...)

	if runFlags.emitEscapeSequence {
		fmt.Printf("\033]777;container;push;%s;toolbox\033\\", containerName)
	}

	err = podman.CmdInto(args...)

	if runFlags.emitEscapeSequence {
		fmt.Print("\033]777;container;pop;;\033\\")
	}

	internalPodmanError := errors.New("exit status 125")
	if err != nil {
		logrus.Debug(err)
		if errors.Is(err, internalPodmanError) {
			logrus.Fatal("Internal Podman error")
		}
		logrus.Infof("There was an error while executing command '%s' in container '%s'", commands[0], containerName)
	}

	return nil
}

func containerStart(containerName string) error {
	args := []string{"start", containerName}
	output, err := podman.CmdOutput(args...)
	if err != nil {
		if strings.Contains(string(output), "use system migrate to mitigate") {
			logrus.Info("Checking if 'podman system migrate' support '--new-runtime' option")

			if podman.CheckVersion("1.6.2") < 0 {
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

			err = podman.CmdRun("system", "migrate", "--new-runtime", ociRuntimeRequired)
			if err != nil {
				return fmt.Errorf("Failed to migrate containers to OCI runtime '%s'", ociRuntimeRequired)
			}

			err = podman.CmdRun("start", containerName)
			if err != nil {
				return fmt.Errorf("Container '%s' doesn't support cgroups %s", containerName, cgroupVersion)
			}
		}

		return errors.New("Failed to start container")
	}

	return nil
}
