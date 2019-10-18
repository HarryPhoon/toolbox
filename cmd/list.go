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
	listFlags struct {
		onlyContainers bool
		onlyImages     bool
	}
)

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
	flags.BoolVarP(&listFlags.onlyContainers, "containers", "c", false, "List only toolbox containers, not images")
	flags.BoolVarP(&listFlags.onlyImages, "images", "i", false, "List only toolbox images, not containers")
}

func list(cmd *cobra.Command, args []string) {
	fmt.Println("function 'list'")
	fmt.Println("OnlyImages:", listFlags.onlyImages)
	fmt.Println("OnlyContainers:", listFlags.onlyContainers)
}
