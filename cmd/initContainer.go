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
	"os"
	"os/exec"
	"os/user"
	"time"

	"github.com/containers/toolbox/pkg/utils"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	initContFlags struct {
		home        string
		homeLink    bool
		mediaLink   bool
		mntLink     bool
		monitorHost bool
		shell       string
		uid         int
		user        string
	}
)

var initContainerCmd = &cobra.Command{
	Use:   "initContainer",
	Short: "Initialize a running container",
	Run: func(cmd *cobra.Command, args []string) {
		initContainer(args)
	},
}

func init() {
	rootCmd.AddCommand(initContainerCmd)
	initContainerCmd.Hidden = true

	flags := initContainerCmd.Flags()
	flags.StringVar(&initContFlags.home, "home", "", "Create a user inside the toolbox container whose login directory is HOME.")
	flags.BoolVar(&initContFlags.homeLink, "home-link", false, "Make /home a symbolic link to /var/home.")
	flags.BoolVar(&initContFlags.mediaLink, "media-link", false, "Make /media a symbolic link to /run/media")
	flags.BoolVar(&initContFlags.mntLink, "mnt-link", false, "Make /mnt a symbolic link to /var/mnt")
	flags.BoolVar(&initContFlags.monitorHost, "monitor-host", false, `Ensure that key configuration files (hosts, timezone,..) inside the toolbox container are in sync with the host.`)
	flags.StringVar(&initContFlags.shell, "shell", "", "Create a user inside the toolbox container whose login shell is SHELL.")
	flags.IntVar(&initContFlags.uid, "uid", 0, "Create a user inside the toolbox container whose numerical user ID is UID.")
	flags.StringVar(&initContFlags.user, "user", "", "Create a user inside the toolbox container whose login name is USER.")
}

func initContainer(args []string) {
	if viper.GetString("XDG_RUNTIME_DIR") == "" {
		logrus.Info("XDG_RUNTIME_DIR is unset")

		viper.Set("XDG_RUNTIME_DIR", fmt.Sprintf("/run/user/%d", initContFlags.uid))
		logrus.Infof("XDG_RUNTIME_DIR set to %s", viper.GetString("XDG_RUNTIME_DIR"))
	}

	toolboxRuntimeDirectory := fmt.Sprintf("%s/toolbox", viper.GetString("XDG_RUNTIME_DIR"))
	containerInitializedStamp := fmt.Sprintf("%s/container-initialized-%d", toolboxRuntimeDirectory, os.Getpid())

	logrus.Info("Creating /run/.toolboxenv")

	_, err := os.Create("/run/.toolboxenv")
	if err != nil {
		logrus.Fatal("Failed to create /run/.toolboxenv")
	}

	if initContFlags.monitorHost {
		logrus.Info("Monitoring host")
		// workingDirectory := viper.GetString("PWD")

		if utils.PathExists("/run/host/etc") {
			logrus.Info("Path /run/host/etc exists. Mount binding to that location will happen now.")

			err = mountBind("/run/host/etc/machine-id", "/etc/machine-id", "ro")
			if err != nil {
				logrus.Fatal(err)
			}

			err = mountBind("/run/host/run/libvirt", "/run/libvirt", "")
			if err != nil {
				logrus.Fatal(err)
			}

			err = mountBind("/run/host/run/systemd/journal", "/run/systemd/journal", "")
			if err != nil {
				logrus.Fatal(err)
			}

			if utils.PathExists("/sys/fs/selinux") {
				err = mountBind("/usr/share/empty", "/sys/fs/selinux", "")
				if err != nil {
					logrus.Fatal(err)
				}
			}

			err = mountBind("/run/host/var/lib/flatpak", "/var/lib/flatpak", "ro")
			if err != nil {
				logrus.Fatal(err)
			}

			err = mountBind("/run/host/var/log/journal", "/var/log/journal", "ro")
			if err != nil {
				logrus.Fatal(err)
			}

			err = mountBind("/run/host/var/mnt", "/var/mnt", "rslave")
			if err != nil {
				logrus.Fatal(err)
			}
		}

		if utils.PathExists("/run/host/monitor") {
			logrus.Info("Path /run/host/monitor exists. Mount binding to that path will happen now.")
			localtimeTarget, err := os.Readlink("/etc/localtime")
			if err != nil || localtimeTarget != "/run/host/monitor/localtime" {
				err = redirectPath("/run/host/monitor/localtime", "/etc/localtime", false)
				if err != nil {
					logrus.Fatal(err)
				}
			}

			timezoneTarget, err := os.Readlink("/etc/timezone")
			if err != nil || timezoneTarget != "/run/host/monitor/timezone" {
				err = redirectPath("/run/host/monitor/timezone", "/etc/timezone", false)
				if err != nil {
					logrus.Fatal(err)
				}
			}

			hostconfTarget, err := os.Readlink("/etc/host.conf")
			if err != nil || hostconfTarget != "/run/host/monitor/host.conf" {
				err = redirectPath("/run/host/monitor/host.conf", "/etc/host.conf", false)
				if err != nil {
					logrus.Fatal(err)
				}
			}

			hostsTarget, err := os.Readlink("/etc/hosts")
			if err != nil || hostsTarget != "/run/host/monitor/hosts" {
				err = redirectPath("/run/host/monitor/hosts", "/etc/hosts", false)
				if err != nil {
					logrus.Fatal(err)
				}
			}

			resolvconfTarget, err := os.Readlink("/etc/resolv.conf")
			if err != nil || resolvconfTarget != "/run/host/monitor/resolv.conf" {
				err = redirectPath("/run/host/monitor/resolv.conf", "/etc/resolv.conf", false)
				if err != nil {
					logrus.Fatal(err)
				}
			}
		}
	}

	if initContFlags.mediaLink {
		if _, err := os.Readlink("/media"); err != nil {
			err = redirectPath("/run/media", "/media", true)
			if err != nil {
				logrus.Fatal(err)
			}
		}
	}

	if initContFlags.mntLink {
		if _, err = os.Readlink("/mnt"); err != nil {
			err = redirectPath("/run/mnt", "/mnt", true)
			if err != nil {
				logrus.Fatal(err)
			}
		}
	}

	_, err = user.Lookup(initContFlags.user)
	if err != nil {
		if initContFlags.homeLink {
			err = redirectPath("/var/home", "/home", true)
			if err != nil {
				logrus.Fatal(err)
			}
		}

		sudoGroup := utils.GetGroupForSudo()
		if sudoGroup == "" {
			logrus.Fatal("Group for sudo was not found")
		}
		logrus.Infof("Group for sudo is %s", sudoGroup)

		logrus.Infof("Adding user %s with UID %d", initContFlags.user, initContFlags.uid)
		args := []string{
			"--home-dir", initContFlags.home,
			"--no-create-home",
			"--shell", initContFlags.shell,
			"--uid", fmt.Sprint(initContFlags.uid),
			"--groups", sudoGroup,
			initContFlags.user,
		}
		useraddCmd := exec.Command("useradd", args...)
		err = useraddCmd.Run()
		if err != nil {
			logrus.Debugf("Arguments passed to 'useradd': %v", args)
			logrus.Debug(err)
			logrus.Fatalf("Failed to add user %s with UID %d", initContFlags.user, initContFlags.uid)
		}

		logrus.Infof("Removing password for user %s", initContFlags.user)
		passwdCmd := exec.Command("passwd", []string{"--delete", initContFlags.user}...)
		err = passwdCmd.Run()
		if err != nil {
			logrus.Debug(err)
			logrus.Fatalf("Failed to remove password for user %s", initContFlags.user)
		}

		logrus.Info("Removing password for user root")
		passwdCmd = exec.Command("passwd", []string{"--delete", "root"}...)
		err = passwdCmd.Run()
		if err != nil {
			logrus.Debug(err)
			logrus.Fatal("Failed to remove password for root")
		}
	}

	if utils.PathExists("/etc/krb5.conf.d") && !utils.PathExists("/etc/krb5.conf.d/kcm_default_ccache") {
		logrus.Info("Setting KCM as the default Kerberos credential cache")

		file, err := os.OpenFile("/etc/krb5.conf.d/kcm_default_ccache", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			logrus.Debug(err)
			logrus.Fatal("Failed to open file /etc/krb5.conf.d/kcm_default_ccache")
		}

		defer file.Close()

		text := `# Written by Toolbox
# https://github.com/debarshiray/toolbox
#
# # To disable the KCM credential cache, comment out the following lines.

[libdefaults]
    default_ccache_name = KCM:`

		if _, err = file.WriteString(text); err != nil {
			logrus.Debug(err)
			logrus.Fatal("Failed to set KCM as the defult Kerberos credential cache")
		}
	}

	logrus.Infof("Creating runtime directory %s", toolboxRuntimeDirectory)
	err = os.MkdirAll(toolboxRuntimeDirectory, 0700)
	if err != nil {
		logrus.Debug(err)
		logrus.Fatalf("Failed to create runtime directory %s", toolboxRuntimeDirectory)
	}

	err = os.Chown(toolboxRuntimeDirectory, initContFlags.uid, initContFlags.uid)
	if err != nil {
		logrus.Debug(err)
		logrus.Fatal("Could not change ownership of the runtime directory")
	}

	logrus.Infof("Creating initialization stamp %s", containerInitializedStamp)
	_, err = os.Create(containerInitializedStamp)
	if err != nil {
		logrus.Fatal("Failed to create initialization stamp")
	}

	err = os.Chown(containerInitializedStamp, initContFlags.uid, initContFlags.uid)
	if err != nil {
		logrus.Debug(err)
		logrus.Fatal("Could not change ownership of the initialization stamp")
	}

	logrus.Info("Finished initializing container")

	logrus.Info("Going to sleep")
	t := time.NewTicker(24 * time.Hour)
	for range t.C {
		time.Sleep(time.Second)
	}
}

func redirectPath(source string, target string, folder bool) error {
	logrus.Infof("Redirecting %s to %s", source, target)

	_, err := os.Stat(target)
	if !os.IsNotExist(err) {
		logrus.Infof("Path %s exists. Deleting it before redirecting.", target)
		err = os.Remove(target)
		if err != nil {
			return err
		}
	}

	if folder {
		os.MkdirAll(source, 0755)
	}

	err = os.Symlink(source, target)
	if err != nil {
		return err
	}

	return nil
}

func mountBind(source string, target string, mountFlags string) error {
	fi, err := os.Stat(source)
	if os.IsNotExist(err) {
		return nil
	}

	if fi.IsDir() {
		logrus.Infof("Creating %s", target)

		err = os.MkdirAll(target, 0755)
		if err != nil {
			return fmt.Errorf("Failed to create %s", target)
		}
	}

	args := []string{
		"--rbind",
	}
	if mountFlags != "" {
		args = append(args, []string{"-o", mountFlags}...)
	}
	args = append(args, []string{source, target}...)

	logrus.Infof("Binding %s to %s", target, source)
	mountCmd := exec.Command("mount", args...)
	err = mountCmd.Run()
	if err != nil {
		return fmt.Errorf("Failed to bind %s to %s", target, source)
	}

	// FIXME: We want to use the golang way of mounting volumes
	/* err = syscall.Mount(source, target, "", "", options)
	if err != nil {
		return fmt.Errorf("Failed to bind %s to %s", target, source)
	} */

	return nil
}
