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
	"fmt"
	"os"
	"os/exec"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/containers/toolbox/utils"
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
		images, err = getImages()
		if err != nil {
			panic(err)
		}
	}

	if listFlags.listContainers {
		containers, err = getContainers()
		if err != nil {
			panic(err)
		}
	}

	err = outputList(images, containers)
	if err != nil {
		panic(err)
	}

	return nil
}

func getContainers() ([]map[string]interface{}, error) {
	args := []string{"ps", "--all", "--sort", "names", "--filter", "label=com.github.debarshiray.toolbox=true", "--format", "json"}
	cmd := exec.Command("podman", args...)
	output, err := cmd.Output()

	var containers_debarshi []map[string]interface{}

	err = json.Unmarshal(output, &containers_debarshi)
	if err != nil {
		return nil, err
	}

	args = []string{"ps", "--all", "--sort", "names", "--filter", "label=com.github.containers.toolbox=true", "--format", "json"}
	cmd = exec.Command("podman", args...)
	output, err = cmd.Output()

	var containers_containers []map[string]interface{}

	err = json.Unmarshal(output, &containers_containers)
	if err != nil {
		panic(err)
	}

	containers := utils.JoinJson("ID", containers_debarshi, containers_containers)
	containers = utils.SortJson(containers, "Names", false)
	return containers, err
}

func getImages() ([]map[string]interface{}, error) {
	args := []string{"images", "--all", "--sort", "names", "--filter", "label=com.github.debarshiray.toolbox=true", "--format", "json"}
	cmd := exec.Command("podman", args...)
	output, err := cmd.Output()

	var imgs_debarshi []map[string]interface{}

	err = json.Unmarshal(output, &imgs_debarshi)
	if err != nil {
		return nil, err
	}

	args = []string{"images", "--all", "--sort", "names", "--filter", "label=com.github.containers.toolbox=true", "--format", "json"}
	cmd = exec.Command("podman", args...)
	output, err = cmd.Output()

	var imgs_containers []map[string]interface{}

	err = json.Unmarshal(output, &imgs_containers)
	if err != nil {
		panic(err)
	}

	images := utils.JoinJson("id", imgs_debarshi, imgs_containers)
	images = utils.SortJson(images, "names", true)

	return images, err
}

func outputList(images, containers []map[string]interface{}) error {
	if len(images) != 0 {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "%s\t%s\t%s\n", "IMAGE ID", "IMAGE NAME", "CREATED")
		for _, image := range images {
			id := image["id"].(string)[:12]
			name := image["names"].([]interface{})[0]
			created := image["created"]
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
			name := container["Names"]
			created := container["Created"]
			status := container["Status"]
			imageName := container["Image"]
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", id, name, created, status, imageName)
		}
		w.Flush()
	}
	return nil
}
