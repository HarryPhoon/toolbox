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
	createFlags struct {
		containerName string
		imageName     string
		releaseVer    int
	}
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new toolbox container",
	Run: func(cmd *cobra.Command, args []string) {
		create()
	},
}

func init() {
	rootCmd.AddCommand(createCmd)

	flags := createCmd.Flags()
	flags.StringVarP(&createFlags.containerName, "container", "c", "", "Assign a different name to the toolbox container")
	flags.StringVarP(&createFlags.imageName, "image", "i", "", "Change the name of the base image used to create the toolbox container")
	flags.IntVarP(&createFlags.releaseVer, "release", "r", 30, "Create a toolbox container for a different operating system release than the host")
}

func create() {
	fmt.Println("function 'create'")
}
