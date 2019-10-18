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

	"github.com/spf13/cobra"
)

var (
	initContFlags struct {
		home        string
		homeLink    string
		monitorHost string
		shell       string
		uid         string
		user        string
	}
)

var initContainerCmd = &cobra.Command{
	Use:   "initContainer",
	Short: "Initialize a running container",
	Run: func(cmd *cobra.Command, args []string) {
		initContainer()
	},
}

func init() {
	rootCmd.AddCommand(initContainerCmd)

	flags := initContainerCmd.Flags()
	flags.StringVar(&initContFlags.home, "home", "", "Create a user inside the toolbox container whose login directory is HOME.")
	flags.StringVar(&initContFlags.homeLink, "home-link", "", "Make /home a symbolic link to /var/home.")
	flags.StringVar(&initContFlags.monitorHost, "monitor-host", "", `Ensure that certain configuration files inside the toolbox container are kept synchronized with their
       counterparts on the host. Currently, these files are /etc/hosts and /etc/resolv.conf.`)
	flags.StringVar(&initContFlags.shell, "shell", "", "Create a user inside the toolbox container whose login shell is SHELL.")
	flags.StringVar(&initContFlags.uid, "uid", "", "Create a user inside the toolbox container whose numerical user ID is UID.")
	flags.StringVar(&initContFlags.user, "user", "", "Create a user inside the toolbox container whose login name is LOGIN.")
}

func initContainer() {
	fmt.Println("function 'initContainer'")
}
