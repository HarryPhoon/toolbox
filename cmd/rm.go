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
	rmFlags struct {
		deleteAll   bool
		forceDelete bool
	}
)

var rmCmd = &cobra.Command{
	Use:   "rm",
	Short: "Remove one or more toolbox containers",
	Run: func(cmd *cobra.Command, args []string) {
		rm()
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)

	flags := rmCmd.Flags()
	flags.BoolVarP(&rmFlags.deleteAll, "all", "a", false, "Remove all toolbox containers")
	flags.BoolVarP(&rmFlags.forceDelete, "force", "f", false, "Force the removal of running and paused toolbox containers")
}

func rm() {
	fmt.Println("function 'rm'")
}
