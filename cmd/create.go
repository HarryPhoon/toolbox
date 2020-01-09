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
	"path/filepath"
	"strings"

	"github.com/containers/toolbox/utils"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	createFlags struct {
		image   string
		release string
	}
	ulimitHost                 = ""
)

var createCmd = &cobra.Command{
	Use:   "create [NAME]",
	Short: "Create a new toolbox container",
	Run: func(cmd *cobra.Command, args []string) {
		create(cmd, args)
	},
	Args: cobra.MaximumNArgs(1),
}

func init() {
	rootCmd.AddCommand(createCmd)

	flags := createCmd.Flags()
	flags.StringVarP(&createFlags.image, "image", "i", "", "Change the name of the base image used to create the toolbox container")
	flags.StringVarP(&createFlags.release, "release", "r", "", "Create a toolbox container for a different operating system release than the host")

	viper.SetDefault("DBUS_SYSTEM_BUS_ADDRESS", "unix:path=/var/run/dbus/system_bus_socket")
}

func create(cmd *cobra.Command, args []string) error {
	// If an image name was not specified use the one that matches the system.
	if len(createFlags.image) == 0 {
		hostPlatform := utils.GetHostPlatform()

		if hostPlatform == "fedora" {
			createFlags.image = "fedora-toolbox"
		}

		// If the system is unknown, use Fedora
		createFlags.image = "fedora-toolbox"
	}

	// If no version is specified and the selected image is the same as host, use the host version ID
	// In all other cases, use the latest stable version of each supported system
	if len(createFlags.release) == 0 {
		hostPlatform := utils.GetHostPlatform()
		hostVersionID := utils.GetHostVersionID()
		if hostPlatform == "fedora" && createFlags.image == "fedora-toolbox" {
			if hostVersionID == "rawhide" {
				createFlags.release = "32"
			} else {
				createFlags.release = utils.GetHostVersionID()
			}
		} else if createFlags.image == "fedora-toolbox" {
			createFlags.release = "31"
		}
	}

	logrus.Infof("Used image will be: %s:%s", createFlags.image, createFlags.release)

	// Look for the toolbox image on local machine
	imageFound := findLocalToolboxImage(createFlags.image)

	logrus.Info("Image was found!!")
	logrus.Info("Preparing dbus system bus address")
	// Inside of a toolbox we want to be able to access dbus for using flatpak-spawn and for users, who work with dbus.
	dbusSystemBusPath := strings.Split(viper.GetString("DBUS_SYSTEM_BUS_ADDRESS"), "=")[1]
	dbusSystemBusPath, err = filepath.EvalSymlinks(dbusSystemBusPath)
	if err != nil {
		logrus.Error(err)
	}
	viper.Set("DBUS_SYSTEM_BUS_ADDRESS", dbusSystemBusPath)

	logrus.Info("Checking if 'podman create' supports '--ulimit host'")
	if utils.CheckPodmanVersion("1.5.0") {
		logrus.Info("'podman create' supports '--ulimit host'")
		ulimitHost = "--ulimit host"
	} else {
		logrus.Info("'podman create' does not support '--ulimit host'")
	}

	return nil
}

func findLocalToolboxImage(imageName string) bool {
	logrus.Info("Looking for the image locally")

	if utils.ReferenceCanBeID(imageName) {
		logrus.Infof("Looking for image %s", imageName)

		if utils.ImageExists(imageName) {
			return true
		}
	}

	hasDomain := utils.ReferenceHasDomain(imageName)

	if !hasDomain {
		imageName = "localhost/" + imageName
		logrus.Infof("Looking for image %s", imageName)

		if utils.ImageExists(imageName) {
			return true
		}
	}

	return false
}
}
