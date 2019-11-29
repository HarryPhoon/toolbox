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
	rmiFlags struct {
		deleteAll   bool
		forceDelete bool
	}
)

var rmiCmd = &cobra.Command{
	Use:   "rmi",
	Short: "Remove one or more toolbox images",
	Run: func(cmd *cobra.Command, args []string) {
		rmi(args)
	},
}

func init() {
	rootCmd.AddCommand(rmiCmd)

	flags := rmiCmd.Flags()
	flags.BoolVarP(&rmiFlags.deleteAll, "all", "a", false, "Remove all toolbox containers")
	flags.BoolVarP(&rmiFlags.forceDelete, "force", "f", false, "Force the removal of running and paused toolbox containers")
}

func rmi(args []string) {
	if rmiFlags.deleteAll {
		logrus.Info("Fetching images with label=com.github.debarshiray.toolbox=true")
		args := []string{"--filter", "label=com.github.debarshiray.toolbox=true"}
		Dimages, err := utils.GetImages(args...)
		if err != nil {
			logrus.Fatal(err)
		}

		logrus.Info("Fetching images with label=com.github.containers.toolbox=true")
		args = []string{"--filter", "label=com.github.containers.toolbox=true"}
		Cimages, err := utils.GetImages(args...)
		if err != nil {
			logrus.Fatal(err)
		}

		images := utils.JoinJSON("id", Dimages, Cimages)
		for _, image := range images {
			logrus.Infof("Deleting image %s", image["id"].(string))
			err = removeImage(image["id"].(string))
			if err != nil {
				logrus.Error(err)
			}
		}
	} else {
		if len(args) == 0 {
			logrus.Fatal("Missing argument")
		}

		for _, imageID := range args {
			// Check if the container exists
			logrus.Infof("Inspecting image %s", imageID)
			args := []string{"inspect", "--format", "json", "--type", "image", imageID}
			output, err := utils.PodmanOutput(args...)
			if err != nil {
				logrus.Fatal(err)
			}

			var info []map[string]interface{}

			err = json.Unmarshal(output, &info)
			if err != nil {
				panic(err)
			}

			// Check if it is a toolbox image
			var labels map[string]interface{}

			logrus.Info("Checking if the image is a toolbox image")
			labels, _ = info[0]["Config"].(map[string]interface{})["Labels"].(map[string]interface{})

			if labels["com.github.debarshiray.toolbox"] != "true" && labels["com.github.containers.toolbox"] != "true" {
				logrus.Fatal("This is not a toolbox image")
			}

			// Try to remove it
			logrus.Infof("Removing image %s", imageID)
			err = removeImage(imageID)
			if err != nil {
				logrus.Fatal(err)
			}
		}
	}
}

func removeImage(image string) error {
	args := []string{"rmi", image}
	if rmiFlags.forceDelete {
		args = append(args, "--force")
	}
	err := utils.PodmanRun(args...)
	if err != nil {
		logrus.Error(err)
	}
	return nil
}
