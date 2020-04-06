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
	"os"
	"text/tabwriter"

	"github.com/containers/toolbox/pkg/podman"
	"github.com/containers/toolbox/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var listFlags struct {
	listContainers bool
	listImages     bool
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List existing toolbox containers and images",
	Run: func(cmd *cobra.Command, args []string) {
		list(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	flags := listCmd.Flags()
	flags.BoolVarP(&listFlags.listContainers, "containers", "c", false, "List only toolbox containers, not images")
	flags.BoolVarP(&listFlags.listImages, "images", "i", false, "List only toolbox images, not containers")
}

func list(cmd *cobra.Command, args []string) error {
	if !listFlags.listContainers && !listFlags.listImages {
		listFlags.listContainers = true
		listFlags.listImages = true
	}

	var images []map[string]interface{}
	var containers []map[string]interface{}
	var err error

	if listFlags.listImages {
		images, err = GetImages()
		if err != nil {
			logrus.Error(err)
			logrus.Debugf("%+v", err)
		}
	}

	if listFlags.listContainers {
		containers, err = GetContainers()
		if err != nil {
			logrus.Error(err)
			logrus.Debugf("%+v", err)
		}
	}

	err = outputList(images, containers)
	if err != nil {
		logrus.Fatal(err)
		logrus.Fatalf("%+v", err)
	}

	return err
}

func GetContainers() ([]map[string]interface{}, error) {
	logrus.Info("Fetching containers with label=com.github.debarshiray.toolbox=true")
	args := []string{"-a", "--filter", "label=com.github.debarshiray.toolbox=true"}
	Dcontainers, err := podman.GetContainers(args...)
	if err != nil {
		err = errors.New("Fetching of containers with com.github.debarshiray.toolbox=true failed")
		logrus.Error(err)
	}

	logrus.Info("Fetching containers with label=com.github.containers.toolbox=true")
	args = []string{"-a", "--filter", "label=com.github.containers.toolbox=true"}
	Ccontainers, err := podman.GetContainers(args...)
	if err != nil {
		err = errors.New("Fetching of containers with com.github.containers.toolbox=true failed")
		logrus.Error(err)
	}

	containers := utils.JoinJSON("ID", Dcontainers, Ccontainers)
	containers = utils.SortJSON(containers, "Names", false)
	return containers, err
}

func GetImages() ([]map[string]interface{}, error) {
	logrus.Info("Fetching images with label=com.github.debarshiray.toolbox=true")
	args := []string{"--filter", "label=com.github.debarshiray.toolbox=true"}
	Dimages, err := podman.GetImages(args...)
	if err != nil {
		logrus.Error(err)
	}

	logrus.Info("Fetching images with label=com.github.containers.toolbox=true")
	args = []string{"--filter", "label=com.github.containers.toolbox=true"}
	Cimages, err := podman.GetImages(args...)
	if err != nil {
		logrus.Error(err)
	}

	var images []map[string]interface{}
	if podman.CheckVersion("1.8.2") < 0 {
		images = utils.JoinJSON("ID", Dimages, Cimages)
		images = utils.SortJSON(images, "Names", true)
	} else {
		images = utils.JoinJSON("id", Dimages, Cimages)
		images = utils.SortJSON(images, "names", true)
	}

	return images, err
}

func outputList(images, containers []map[string]interface{}) error {
	if len(images) != 0 {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "%s\t%s\t%s\n", "IMAGE ID", "IMAGE NAME", "CREATED")

		var idKey, nameKey, createdKey string
		if podman.CheckVersion("1.8.2") < 0 {
			idKey = "ID"
			nameKey = "Names"
			createdKey = "Created"
		} else {
			idKey = "id"
			nameKey = "names"
			createdKey = "created"
		}
		for _, image := range images {
			id := utils.ShortID(image[idKey].(string))
			name := image[nameKey].([]interface{})[0].(string)
			created := image[createdKey].(string)
			fmt.Fprintf(w, "%s\t%s\t%s\n", id, name, created)
		}
		w.Flush()
	}

	if len(images) != 0 && len(containers) != 0 {
		fmt.Println()
	}

	if len(containers) != 0 {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", "CONTAINER ID", "CONTAINER NAME", "CREATED", "STATUS", "IMAGE NAME")

		for _, container := range containers {
			id := container["ID"].(string)[:12]
			name := container["Names"].(string)
			created := container["Created"].(string)
			status := container["Status"].(string)
			imageName := container["Image"].(string)
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", id, name, created, status, imageName)
		}
		w.Flush()
	}
	return nil
}
