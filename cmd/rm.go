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
	"encoding/json"

	"github.com/containers/toolbox/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	rmFlags struct {
		deleteAll   bool
		forceDelete bool
	}
)

var rmCmd = &cobra.Command{
	Use:   "rm [CONTAINER...]",
	Short: "Remove one or more toolbox containers",
	Run: func(cmd *cobra.Command, args []string) {
		rm(args)
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)

	flags := rmCmd.Flags()
	flags.BoolVarP(&rmFlags.deleteAll, "all", "a", false, "Remove all toolbox containers")
	flags.BoolVarP(&rmFlags.forceDelete, "force", "f", false, "Force the removal of running and paused toolbox containers")
}

func rm(args []string) {
	if rmFlags.deleteAll {
		args := []string{"--filter", "label=com.github.debarshiray.toolbox=true"}
		Dcontainers, err := utils.GetContainers(args...)
		if err != nil {
			logrus.Fatal(err)
		}

		args = []string{"--filter", "label=com.github.containers.toolbox=true"}
		Ccontainers, err := utils.GetContainers(args...)
		if err != nil {
			logrus.Fatal(err)
		}

		containers := utils.JoinJSON("ID", Dcontainers, Ccontainers)

		for _, container := range containers {
			err = removeContainer(container["ID"].(string))
			if err != nil {
				logrus.Error(err)
			}
		}
	} else {
		if len(args) == 0 {
			logrus.Fatal("Missing argument")
		}

		for _, containerName := range args {
			// Check if the container exists
			args := []string{"inspect", "--format", "json", "--type", "container", containerName}
			output, err := utils.PodmanOutput(args...)
			if err != nil {
				logrus.Fatal(err)
			}

			var info []map[string]interface{}

			err = json.Unmarshal(output, &info)
			if err != nil {
				panic(err)
			}

			// Check if it is a toolbox container
			var labels map[string]interface{}

			labels, _ = info[0]["Config"].(map[string]interface{})["Labels"].(map[string]interface{})

			if labels["com.github.debarshiray.toolbox"] != "true" && labels["com.github.containers.toolbox"] != "true" {
				logrus.Fatal("This is not a toolbox container")
			}

			// Try to remove it
			err = removeContainer(containerName)
			if err != nil {
				logrus.Fatal(err)
			}
		}
	}
}

func removeContainer(containerName string) error {
	args := []string{"rm", containerName}
	if rmFlags.forceDelete {
		args = append(args, "--force")
	}
	err := utils.PodmanRun(args...)
	if err != nil {
		logrus.Error(err)
	}
	return nil
}
