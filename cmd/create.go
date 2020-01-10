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
	"fmt"
	"path/filepath"
	"strings"

	"github.com/containers/toolbox/utils"

	"github.com/godbus/dbus/v5"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	createFlags struct {
		image   string
		release string
	}
	ulimitHost               = []string{}
	homeCanonical            = ""
	toolboxProfileBind       = []string{}
	sudoGroup                = ""
	kcmSocket                = ""
	kcmSocketBind            = []string{}
	usrMountPoint            = ""
	usrMountSourceFlags      = ""
	usrMountDestinationFlags = "ro"
	dbusSystemBusAddress     = ""
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

	// If no container name is specified then use the image name and it's version
	var containerName string
	if len(args) != 0 {
		containerName = args[0]
	} else {
		containerName = createFlags.image + "-" + createFlags.release
	}

	logrus.Infof("Checking if container %s already exists", containerName)
	containerList, err := utils.GetContainers("--all", "--filter", fmt.Sprintf("name=%s", containerName))
	if err != nil {
		logrus.Error(err)
	}
	if len(containerList) != 0 {
		logrus.Fatalf("Container %s already exists", containerName)
	}

	imageName := createFlags.image + ":" + createFlags.release
	logrus.Infof("Used image will be: %s", imageName)

	// Look for the toolbox image on local machine
	imageFound := findLocalToolboxImage(imageName)

	if !imageFound {
		logrus.Fatalf("Image '%s' was not found", imageName)
	}

	logrus.Infof("Image '%s' was found", imageName)

	// If the image was not pulled that check if it is a Toolbox image
	if imageFound {
		logrus.Infof("Checking if '%s' is a Toolbox image", imageName)
		inspectInfo, err := utils.PodmanInspect(imageName)
		if err != nil {
			logrus.Fatalf("Unable to inspect image '%s'", imageName)
		}
		imageLabels := inspectInfo["Labels"].(map[string]interface{})

		isToolboxImage := false
		if imageLabels["com.github.debarshiray.toolbox"] == "true" {
			isToolboxImage = true
		}
		if imageLabels["com.github.containers.toolbox"] == "true" {
			isToolboxImage = true
		}

		if !isToolboxImage {
			logrus.Fatalf("Image '%s' is not a Toolbox image", imageName)
		} else {
			logrus.Infof("Image '%s' is a Toolbox image", imageName)
		}
	}

	logrus.Info("Looking for group for sudo")
	sudoGroup = utils.GetGroupForSudo()
	if sudoGroup == "" {
		logrus.Fatal("Group for sudo was not found")
	}

	// Start assembling the arguments for Podman
	createArgs := []string{
		"create",
		"--dns", "none",
		"--group-add", sudoGroup,
		"--hostname", "toolbox",
		"--ipc", "host",
		"--label", "com.github.containers.toolbox=true",
		"--name", containerName,
		"--network", "host",
		"--no-hosts",
		"--pid", "host",
		"--privileged",
		"--security-opt", "label=disable",
		"--userns=keep-id",
		"--user", "root:root"}

	logrus.Info("Checking if /etc/profile.d/toolbox.sh exists")
	if utils.FileExists("/etc/profile.d/toolbox.sh") {
		logrus.Info("File /etc/profile.d/toolbox.sh exists")
		toolboxProfileBind = []string{"--volume", "/etc/profile.d/toolbox.sh:/etc/profile.d/toolbox.sh:ro"}
		createArgs = append(createArgs, toolboxProfileBind...)
	}

	logrus.Info("Checking if /usr is mounted read-only or read-write")
	usrMountPoint, err = utils.GetMountPoint("/usr")
	if err != nil {
		logrus.Error(err)
		logrus.Fatal("Failed to get the mount-point of /usr")
	}

	logrus.Infof("Mount-point of /usr is %ss", usrMountPoint)
	usrMountSourceFlags, err = utils.GetMountOptions(usrMountPoint)
	if err != nil {
		logrus.Error(err)
		logrus.Fatalf("Failed to get the mount options of %s", usrMountPoint)
	}

	logrus.Infof("Mount flags of /usr on the host are %s", usrMountSourceFlags)
	if !strings.Contains(usrMountSourceFlags, "ro") {
		usrMountDestinationFlags = "rw"
	}

	// Inside of a toolbox we want to be able to access dbus for using flatpak-spawn and for users, who work with dbus.
	logrus.Info("Preparing dbus system bus address")
	dbusSystemBusAddress = strings.Split(viper.GetString("DBUS_SYSTEM_BUS_ADDRESS"), "=")[1]
	dbusSystemBusAddress, err = filepath.EvalSymlinks(dbusSystemBusAddress)
	if err != nil {
		logrus.Error(err)
	}

	dbusSystemBusAddressBind := []string{"--volume", fmt.Sprintf("%s:%s", dbusSystemBusAddress, dbusSystemBusAddress)}
	createArgs = append(createArgs, dbusSystemBusAddressBind...)

	logrus.Info("Preparing sssd-kcm socket")
	args = []string{"show", "--value", "--property", "Listen", "sssd-kcm.socket"}
	output, err := utils.SystemctlOutput(args...)
	if err != nil {
		logrus.Error("Failed to use 'systemctl show'")
	}

	kcmSocket = strings.Trim(string(output), "\n")

	if kcmSocket == "" {
		logrus.Error("Failed to read property Listen from sssd-kcm.socket")
	} else {
		logrus.Infof("Checking value %s of property Listen in sssd-kcm.socket", kcmSocket)
		if !strings.Contains(kcmSocket, " (Stream)") {
			kcmSocket = ""
			logrus.Error("Unknown socket in sssd-kcm.socket\nExpected SOCK_STREAM")
		}
		if !strings.Contains(kcmSocket, "/") {
			kcmSocket = ""
			logrus.Error("Unknown socket in sssd-kcm.socket\nExpected file system socket in the AF_UNIX family")
		}
	}

	logrus.Infof("Parsing value %s of property Listen in sssd-kcm.socket", kcmSocket)
	if kcmSocket != "" {
		kcmSocket = strings.TrimSuffix(kcmSocket, " (Stream)")
		kcmSocketBind = []string{"--volume", fmt.Sprintf("%s:%s", kcmSocket, kcmSocket)}
		createArgs = append(createArgs, kcmSocketBind...)
	}

	logrus.Info("Checking if 'podman create' supports option '--ulimit host'")
	if utils.CheckPodmanVersion("1.5.0") {
		logrus.Info("Option '--ulimit host' is supported")
		ulimitHost = []string{"--ulimit", "host"}
	} else {
		logrus.Info("Option '--ulimit host' is not supported")
	}

	homeEnv := strings.Split(viper.GetString("HOME"), "=")[0]
	homeCanonical, err = filepath.EvalSymlinks(homeEnv)
	if err != nil {
		logrus.Fatalf("Failed to canonicalize %s", homeEnv)
	}
	logrus.Infof("Canonicalized %s to %s", homeEnv, homeCanonical)

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

	createArgs = append(createArgs, []string{
		"--volume", fmt.Sprintf("%s:%s", viper.GetString("XDG_RUNTIME_DIR"), viper.GetString("XDG_RUNTIME_DIR")),
		"--volume", fmt.Sprintf("%s/.flatpak-helper/monitor:/run/host/monitor", viper.GetString("XDG_RUNTIME_DIR")),
		"--volume", fmt.Sprintf("%s:%s:rslave", homeCanonical, homeCanonical),
		"--volume", "/etc:/run/host/etc",
		"--volume", "/dev:/dev:rslave",
		"--volume", "/media:/media:rslave",
		"--volume", "/mnt:/mnt:rslave",
		"--volume", "/run:/run/host/run:rslave",
		"--volume", "/tmp:/run/host/tmp:rslave",
		"--volume", fmt.Sprintf("/usr:/run/host/usr:%s,rslave", usrMountDestinationFlags),
		"--volume", "/var:/run/host/var:rslave",
		imageName,
		"sleep", "99999999999"}...)

	output, err = utils.PodmanOutput(createArgs...)
	if err != nil {
		logrus.Fatalf("Failed to create container %s", containerName)
	}

	return nil
}

func findLocalToolboxImage(imageName string) bool {
	logrus.Info("Looking for the image locally")

	if utils.ImageExists(imageName) {
		return true
	}

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
