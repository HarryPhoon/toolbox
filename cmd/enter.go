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
	"github.com/containers/toolbox/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	enterFlags struct {
		releaseVersion string
	}
)

var enterCmd = &cobra.Command{
	Use:   "enter [flags] CONTAINER",
	Short: "Enter a toolbox container for interactive use",
	PreRun: func(cmd *cobra.Command, args []string) {
		runFlags.fallbackToBash = true
		runFlags.pedantic = false
		runFlags.emitEscapeSequence = true
	},
	Run: func(cmd *cobra.Command, args []string) {
		var containerName string = ""
		if len(args) != 0 {
			containerName = args[0]
		}
		containerName, _ = utils.UpdateContainerAndImageNames(containerName, "", enterFlags.releaseVersion)

		args = []string{containerName, viper.GetString("SHELL")}
		run(args)
	},
	Args: cobra.MaximumNArgs(1),
}

func init() {
	rootCmd.AddCommand(enterCmd)

	flags := enterCmd.Flags()
	flags.StringVarP(&enterFlags.releaseVersion, "release", "r", "", "Run command inside a toolbox container with the release version")
}
